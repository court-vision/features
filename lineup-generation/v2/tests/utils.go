package tests

import (
	"fmt"
	"runtime"
	"sort"
	"testing"

	d "v2/data"
	p "v2/population"
	l "v2/resources"
	"v2/team"
	"v2/testutil"
	u "v2/utils"
)

// loadSchedule makes sure the bundled schedule is loaded (once per test binary). It uses
// the newest season file present under static/, so the suite keeps passing while a new
// season's schedule is being added.
func loadSchedule(t testing.TB) {
	t.Helper()
	path, err := testutil.SchedulePath()
	if err != nil {
		t.Fatal(err)
	}
	if err := d.InitSchedule(path); err != nil {
		t.Fatalf("load schedule %s: %v", path, err)
	}
}

func rosterPath() string     { return testutil.RepoPath("resources", "mock_roster.json") }
func freeAgentsPath() string { return testutil.RepoPath("resources", "mock_freeagents.json") }

// loadRosterMap returns the mock roster keyed by player name, as stored in the fixture.
func loadRosterMap(t testing.TB) map[string]d.Player {
	t.Helper()
	m := l.LoadRosterMap(rosterPath())
	if len(m) == 0 {
		t.Fatalf("mock roster at %s is empty or unreadable", rosterPath())
	}
	return m
}

// loadRosterSlice returns the mock roster as the []Player slice the API accepts.
func loadRosterSlice(t testing.TB) []d.Player {
	t.Helper()
	return rosterToSlice(loadRosterMap(t))
}

// rosterToSlice flattens a roster map into a slice in a deterministic (name) order.
func rosterToSlice(m map[string]d.Player) []d.Player {
	players := make([]d.Player, 0, len(m))
	for _, player := range m {
		players = append(players, player)
	}
	sort.Slice(players, func(i, j int) bool { return players[i].Name < players[j].Name })
	return players
}

func loadFreeAgents(t testing.TB) []d.Player {
	t.Helper()
	fas := l.LoadFreeAgents(freeAgentsPath())
	if len(fas) == 0 {
		t.Fatalf("mock free agents at %s are empty or unreadable", freeAgentsPath())
	}
	return fas
}

// sortedTeams returns the tricodes in the week's schedule in a deterministic order.
func sortedTeams(week int) []string {
	teams := d.ScheduleMap.GetWeekSchedule(week).TeamSchedules
	names := make([]string, 0, len(teams))
	for name := range teams {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// teamPlayingOn returns a deterministic team tricode that plays on the given day of the
// week according to the loaded schedule, so test players can be built to be "active".
func teamPlayingOn(t testing.TB, week, day int) string {
	t.Helper()
	for _, name := range sortedTeams(week) {
		if d.ScheduleMap.IsPlaying(week, day, name) {
			return name
		}
	}
	t.Fatalf("no team plays on week %d day %d", week, day)
	return ""
}

// teamNotPlayingOn returns a deterministic team tricode that is idle on the given day.
func teamNotPlayingOn(t testing.TB, week, day int) string {
	t.Helper()
	for _, name := range sortedTeams(week) {
		if !d.ScheduleMap.IsPlaying(week, day, name) {
			return name
		}
	}
	t.Fatalf("every team plays on week %d day %d", week, day)
	return ""
}

// assertStreamersSlotted checks the invariant InsertStreamablePlayers must satisfy for a
// single day, independent of which season's schedule is loaded: a streamer whose team
// plays is rostered in one of his valid positions that the core lineup left open (or is
// benched only because none of them was free), and a streamer whose team is idle is benched.
func assertStreamersSlotted(t testing.TB, bt *team.BaseTeam, gene *p.Gene) {
	t.Helper()
	day := gene.Day

	for _, streamer := range bt.StreamablePlayers {
		pos := gene.GetPosOfPlayer(streamer)
		onBench := gene.Bench.IsOnBench(streamer)
		playing := d.ScheduleMap.IsPlaying(bt.Week, day, streamer.Team)

		switch {
		case !playing:
			if pos != "BE" || !onBench {
				t.Errorf("day %d: %s (%s) is idle and should be benched, got pos=%q bench=%v", day, streamer.Name, streamer.Team, pos, onBench)
			}
		case pos != "BE":
			if onBench {
				t.Errorf("day %d: %s is both rostered at %s and benched", day, streamer.Name, pos)
			}
			if !streamer.PlaysPosition(pos) {
				t.Errorf("day %d: %s rostered at %s, not one of %v", day, streamer.Name, pos, streamer.ValidPositions)
			}
			if !bt.UnusedPositions[day][pos] {
				t.Errorf("day %d: %s rostered at %s, but the core lineup already fills it", day, streamer.Name, pos)
			}
			if gene.FreePositions[pos] {
				t.Errorf("day %d: %s occupies %s but it is still marked free", day, streamer.Name, pos)
			}
		default: // playing but benched: only allowed when none of his positions is open
			if !onBench {
				t.Errorf("day %d: %s is neither rostered nor benched", day, streamer.Name)
			}
			for _, valid := range streamer.ValidPositions {
				if gene.FreePositions[valid] {
					t.Errorf("day %d: %s benched while his valid position %s is free", day, streamer.Name, valid)
				}
			}
		}
	}

	// Before AddBackNonStreamablePlayers, only streamers may appear in a gene's roster
	for pos, player := range gene.Roster {
		if player.Name == "" {
			continue
		}
		if !u.SliceContainsPlayer(bt.StreamablePlayers, &player) {
			t.Errorf("day %d: non-streamer %s at %s", day, player.Name, pos)
		}
	}
	if got, want := gene.GetNumStreamers(), len(bt.StreamablePlayers); got != want {
		t.Errorf("day %d: gene tracks %d streamers, want %d", day, got, want)
	}
}

// assertChromosomeConsistent checks the bookkeeping invariants every chromosome must hold
// after Populate, Crossover and Mutate: acquisitions match the recorded additions and
// drops, additions are unique and present in the gene, and no streamer is lost. (Crossover
// can pick up a parent's addition on a day the child has no open slot, benching him that
// day; that is a wasted acquisition the fitness score punishes, not a lost player.)
func assertChromosomeConsistent(t testing.TB, bt *team.BaseTeam, c *p.Chromosome, label string) {
	t.Helper()
	if got, want := len(c.Genes), d.ScheduleMap.GetGameSpan(bt.Week); got != want {
		t.Errorf("%s: %d genes, want %d", label, got, want)
	}

	total := 0
	for day, gene := range c.Genes {
		if gene.Day != day {
			t.Errorf("%s: gene %d has Day %d", label, day, gene.Day)
		}
		if len(gene.NewPlayers) != gene.Acquisitions {
			t.Errorf("%s day %d: %d new players but Acquisitions=%d", label, day, len(gene.NewPlayers), gene.Acquisitions)
		}
		if len(gene.NewPlayers) != len(gene.DroppedPlayers) {
			t.Errorf("%s day %d: %d additions but %d drops", label, day, len(gene.NewPlayers), len(gene.DroppedPlayers))
		}
		total += gene.Acquisitions

		seen := make(map[string]bool, len(gene.NewPlayers))
		for _, player := range gene.NewPlayers {
			if player.Name == "" {
				t.Errorf("%s day %d: addition with empty name", label, day)
			}
			if seen[player.Name] {
				t.Errorf("%s day %d: duplicate addition %s", label, day, player.Name)
			}
			seen[player.Name] = true
			if !gene.IsPlayerInGene(player) {
				t.Errorf("%s day %d: addition %s is neither rostered nor benched", label, day, player.Name)
			}
		}
		for _, player := range gene.DroppedPlayers {
			if player.Name == "" {
				t.Errorf("%s day %d: drop with empty name", label, day)
			}
		}
		if got, want := gene.GetNumStreamers(), len(bt.StreamablePlayers); got != want {
			t.Errorf("%s day %d: gene tracks %d streamers, want %d", label, day, got, want)
		}
	}
	if total != c.TotalAcquisitions {
		t.Errorf("%s: genes sum to %d acquisitions but TotalAcquisitions=%d", label, total, c.TotalAcquisitions)
	}
}

func printMemUsage() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Printf("Alloc = %v MiB", bToMb(m.Alloc))
	fmt.Printf("\tTotalAlloc = %v MiB", bToMb(m.TotalAlloc))
	fmt.Printf("\tSys = %v MiB", bToMb(m.Sys))
	fmt.Printf("\tNumGC = %v\n", m.NumGC)
}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}
