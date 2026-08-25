package tests

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	d "v2/data"
	p "v2/population"
	"v2/team"
	u "v2/utils"
)

func TestInitChromosome(t *testing.T) {
	loadSchedule(t)

	week := 1
	bt := team.InitBaseTeamMock(week, 2)

	c := p.InitChromosome(bt)

	if got, want := len(c.Genes), d.ScheduleMap.GetGameSpan(week); got != want {
		t.Errorf("Incorrect number of genes: %d, want %d", got, want)
	}
	for i, gene := range c.Genes {
		if gene == nil || gene.Day != i {
			t.Errorf("gene %d is %+v", i, gene)
		}
	}
	if c.Week != week {
		t.Errorf("chromosome week %d, want %d", c.Week, week)
	}
	if len(c.CurStreamers) != len(bt.StreamablePlayers) {
		t.Errorf("%d current streamers, want %d", len(c.CurStreamers), len(bt.StreamablePlayers))
	}
}

func TestChromosomeInsertStreamablePlayers(t *testing.T) {
	loadSchedule(t)

	bt := team.InitBaseTeamMock(1, 2)

	c := p.InitChromosome(bt)

	// Insert streamable players into the genes
	for _, gene := range c.Genes {
		gene.InsertStreamablePlayers(bt)
	}

	// Validate that the streamers were inserted into the right spots on every day
	for _, gene := range c.Genes {
		assertStreamersSlotted(t, bt, gene)
	}
}

func TestInsertFreeAgent(t *testing.T) {
	loadSchedule(t)

	week := 1

	// Make the scenario deterministic for any season's schedule: both streamers sit out
	// day 0, so the free agent (who plays day 0) replaces the worst benched streamer
	// from day 0 through the end of the week.
	roster := loadRosterMap(t)
	idle := teamNotPlayingOn(t, week, 0)
	for _, name := range []string{"Bradley Beal", "Vince Williams Jr."} {
		player, ok := roster[name]
		if !ok {
			t.Fatalf("%s missing from mock roster", name)
		}
		player.Team = idle
		roster[name] = player
	}
	bt := team.InitBaseTeam(rosterToSlice(roster), loadFreeAgents(t), week, 2)
	if len(bt.StreamablePlayers) != 2 {
		t.Fatalf("expected 2 streamers, got %d", len(bt.StreamablePlayers))
	}

	c := p.InitChromosome(bt)

	// Insert streamable players into the genes
	for _, gene := range c.Genes {
		gene.InsertStreamablePlayers(bt)
	}
	if c.Genes[0].Bench.GetLength() != 2 {
		t.Fatalf("both streamers should be benched on day 0, bench has %d", c.Genes[0].Bench.GetLength())
	}
	worst := c.Genes[0].Bench.Players[0]

	// Insert a free agent who plays on day 0 into the chromosome
	free_agent := d.Player{Name: "Random Free Agent1", AvgPoints: 10.0, Team: teamPlayingOn(t, week, 0), ValidPositions: []string{"C", "F", "UT1", "UT2", "UT3"}, Injured: false}
	wantPos := ""
	for _, pos := range free_agent.ValidPositions {
		if c.Genes[0].FreePositions[pos] {
			wantPos = pos
			break
		}
	}
	if !c.InsertFreeAgent(bt, 0, free_agent) {
		t.Fatalf("InsertFreeAgent failed")
	}

	c.Print()

	// Validate that the free agent was inserted for the rest of the week and that the worst streamer was dropped
	for day, gene := range c.Genes {
		if gene.IsPlayerInGene(worst) {
			t.Errorf("%s is still in the gene on day %d", worst.Name, day)
		}
		if !gene.IsPlayerInGene(free_agent) {
			t.Errorf("%s is missing from the gene on day %d", free_agent.Name, day)
		}
		if d.ScheduleMap.IsPlaying(week, day, free_agent.Team) {
			if pos := gene.GetPosOfPlayer(free_agent); pos != "BE" && (!free_agent.PlaysPosition(pos) || gene.FreePositions[pos]) {
				t.Errorf("day %d: %s rostered at %s (free=%v)", day, free_agent.Name, pos, gene.FreePositions[pos])
			}
		} else if !gene.Bench.IsOnBench(free_agent) {
			t.Errorf("day %d: %s is idle and should be benched", day, free_agent.Name)
		}
	}
	if wantPos != "" {
		if got := c.Genes[0].GetPosOfPlayer(free_agent); got != wantPos {
			t.Errorf("day 0: %s at %q, want %s", free_agent.Name, got, wantPos)
		}
	}

	// The free agent replaces the dropped streamer in the chromosome's current streamers
	if !u.SliceContainsPlayer(c.CurStreamers, &free_agent) || u.SliceContainsPlayer(c.CurStreamers, &worst) {
		t.Errorf("current streamers %v should contain %s and not %s", c.CurStreamers, free_agent.Name, worst.Name)
	}
}

func TestPopulateChromosome(t *testing.T) {
	loadSchedule(t)

	max_aquisitions := 0
	for i := 0; i < 100; i++ {

		bt := team.InitBaseTeamMock(2, 3)
		seed := time.Now().UnixNano() + int64(1)
		rng := rand.New(rand.NewSource(seed))

		c := p.InitChromosome(bt)

		c.Populate(bt, rng)

		assertChromosomeConsistent(t, bt, c, fmt.Sprintf("populate #%d", i))

		// Populate only picks free agents who fit an open slot, so every addition starts
		for day, gene := range c.Genes {
			for _, player := range gene.NewPlayers {
				if !gene.IsPlayerInRoster(player) {
					t.Errorf("populate #%d day %d: addition %s is not in the roster", i, day, player.Name)
				}
			}
		}

		if c.TotalAcquisitions > max_aquisitions {
			max_aquisitions = c.TotalAcquisitions
		}
	}

	fmt.Println("Max acquisitions:", max_aquisitions)
}

func TestChromosomeSlim(t *testing.T) {
	loadSchedule(t)

	week := 2
	bt := team.InitBaseTeamMock(week, 2)
	seed := time.Now().UnixNano()
	rng := rand.New(rand.NewSource(seed))

	c := p.InitChromosome(bt)

	c.Populate(bt, rng)
	c.AddBackNonStreamablePlayers(bt)

	slim_chromosome := c.Slim()
	if got, want := len(slim_chromosome), d.ScheduleMap.GetGameSpan(week); got != want {
		t.Fatalf("slim chromosome has %d days, want %d", got, want)
	}
	for day, slim := range slim_chromosome {
		if slim.Day != day {
			t.Errorf("slim gene %d has Day %d", day, slim.Day)
		}
		if len(slim.Additions) != len(c.Genes[day].NewPlayers) || len(slim.Removals) != len(c.Genes[day].DroppedPlayers) {
			t.Errorf("day %d: slim additions/removals %d/%d, want %d/%d", day, len(slim.Additions), len(slim.Removals), len(c.Genes[day].NewPlayers), len(c.Genes[day].DroppedPlayers))
		}
		// Core players are back in their optimal positions
		for pos, player := range bt.OptimalSlotting[day] {
			if player.Name == "" {
				continue
			}
			if slim.Roster[pos].Name != player.Name {
				t.Errorf("day %d: %s should hold %s, got %q", day, pos, player.Name, slim.Roster[pos].Name)
			}
		}
		for pos, player := range slim.Roster {
			if player.Name == "" {
				t.Errorf("day %d: empty player at %s", day, pos)
			}
		}
	}
	fmt.Println(slim_chromosome[0])
}
