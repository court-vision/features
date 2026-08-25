package tests

import (
	"testing"

	d "v2/data"
	"v2/team"
)

func TestBTInitWithData(t *testing.T) {
	loadSchedule(t)

	// Test the InitBaseTeam function with mock data
	week := 1
	streamingSlots := 3
	bt := team.InitBaseTeam(loadRosterSlice(t), loadFreeAgents(t), week, streamingSlots)

	// Validate fields
	BTFieldValidator(bt, t, "Anthony Edwards", "SG", 7, "MIN", streamingSlots, "RosterMap")
	BTFieldValidator(bt, t, "Naz Reid", "PF", 6, "MIN", streamingSlots, "FreeAgents")
	BTFieldValidator(bt, t, "", "", 0, "", streamingSlots, "OptimalSlotting")
	BTFieldValidator(bt, t, "", "", 0, "", streamingSlots, "UnusedPositions")
	BTFieldValidator(bt, t, "Bradley Beal", "PG", 6, "PHX", streamingSlots, "StreamablePlayers")
	if bt.Week != week {
		t.Errorf("Week is incorrect")
	}
	if got, want := len(bt.OptimalSlotting), d.ScheduleMap.GetGameSpan(week); got != want {
		t.Errorf("OptimalSlotting covers %d days, want %d", got, want)
	}
	if bt.Score <= 0 {
		t.Errorf("optimal score should be positive, got %d", bt.Score)
	}
}

func TestBTInitMock(t *testing.T) {
	loadSchedule(t)

	// Test the InitBaseTeamMock function
	week := 1
	streamingSlots := 3
	bt := team.InitBaseTeamMock(week, streamingSlots)

	// Validate fields
	BTFieldValidator(bt, t, "Anthony Edwards", "SG", 7, "MIN", streamingSlots, "RosterMap")
	BTFieldValidator(bt, t, "Naz Reid", "PF", 6, "MIN", streamingSlots, "FreeAgents")
	BTFieldValidator(bt, t, "", "", 0, "", streamingSlots, "OptimalSlotting")
	BTFieldValidator(bt, t, "", "", 0, "", streamingSlots, "UnusedPositions")
	BTFieldValidator(bt, t, "Bradley Beal", "PG", 6, "PHX", streamingSlots, "StreamablePlayers")
	if bt.Week != week {
		t.Errorf("Week is incorrect")
	}

	// The mock must match what InitBaseTeam builds from the same fixtures
	fromData := team.InitBaseTeam(loadRosterSlice(t), loadFreeAgents(t), week, streamingSlots)
	if fromData.Score != bt.Score || len(fromData.StreamablePlayers) != len(bt.StreamablePlayers) {
		t.Errorf("mock base team (score %d, %d streamers) differs from data-built one (score %d, %d streamers)",
			bt.Score, len(bt.StreamablePlayers), fromData.Score, len(fromData.StreamablePlayers))
	}
}

func TestBTPlayersToMap(t *testing.T) {
	// Test the PlayersToMap function
	roster := loadRosterSlice(t)
	roster_map := d.PlayersToMap(roster)
	if len(roster_map) != len(roster) {
		t.Errorf("PlayersToMap produced %d entries from %d players", len(roster_map), len(roster))
	}

	bt := &team.BaseTeam{
		RosterMap: roster_map,
	}

	// Validate fields
	BTFieldValidator(bt, t, "Anthony Edwards", "SG", 7, "MIN", 0, "RosterMap")
}

func TestBTOptimizeSlottingAndStreamablePlayers(t *testing.T) {
	loadSchedule(t)

	// Test the OptimizeSlotting function
	week := 1
	streamingSlots := 3
	bt := &team.BaseTeam{
		RosterMap:  loadRosterMap(t),
		FreeAgents: loadFreeAgents(t),
	}
	bt.OptimizeSlotting(week, streamingSlots)

	// Validate field
	BTFieldValidator(bt, t, "Anthony Edwards", "SG", 7, "MIN", streamingSlots, "OptimalSlotting")
	BTFieldValidator(bt, t, "Bradley Beal", "PG", 6, "PHX", streamingSlots, "StreamablePlayers")
	if got, want := len(bt.OptimalSlotting), d.ScheduleMap.GetGameSpan(week); got != want {
		t.Errorf("OptimalSlotting covers %d days, want %d", got, want)
	}

	// Streamers are the lowest-scoring healthy players; injured players are never slotted or streamed
	for _, streamer := range bt.StreamablePlayers {
		if streamer.Injured {
			t.Errorf("injured player %s is streamable", streamer.Name)
		}
		for _, player := range bt.RosterMap {
			if !player.Injured && player.AvgPoints < streamer.AvgPoints && !containsPlayer(bt.StreamablePlayers, player.Name) {
				t.Errorf("%s (%.2f) is streamable but lower-scoring %s (%.2f) is not", streamer.Name, streamer.AvgPoints, player.Name, player.AvgPoints)
			}
		}
	}
	for day, lineup := range bt.OptimalSlotting {
		for pos, player := range lineup {
			if player.Injured {
				t.Errorf("day %d: injured %s slotted at %s", day, player.Name, pos)
			}
		}
	}
}

func TestBTFindUnusedPositions(t *testing.T) {
	loadSchedule(t)

	// Test the FindUnusedPositions function
	week := 1
	streamingSlots := 3
	bt := &team.BaseTeam{
		RosterMap:  loadRosterMap(t),
		FreeAgents: loadFreeAgents(t),
	}
	bt.OptimizeSlotting(week, streamingSlots)
	bt.FindUnusedPositions()

	// Validate field
	BTFieldValidator(bt, t, "", "", 0, "", streamingSlots, "UnusedPositions")
	if got, want := len(bt.UnusedPositions), d.ScheduleMap.GetGameSpan(week); got != want {
		t.Errorf("UnusedPositions covers %d days, want %d", got, want)
	}

	// Every starting position is either used by the optimal lineup or reported unused
	pos_order := []string{"PG", "SG", "SF", "PF", "C", "G", "F", "UT1", "UT2", "UT3"}
	for day := 0; day < d.ScheduleMap.GetGameSpan(week); day++ {
		for _, pos := range pos_order {
			used := bt.OptimalSlotting[day][pos].Name != ""
			if used == bt.UnusedPositions[day][pos] {
				t.Errorf("day %d %s: used=%v but unused=%v", day, pos, used, bt.UnusedPositions[day][pos])
			}
		}
	}
}

func containsPlayer(players []d.Player, name string) bool {
	for _, player := range players {
		if player.Name == name {
			return true
		}
	}
	return false
}

func BTFieldValidator(bt *team.BaseTeam, t *testing.T, name string, position string, num_positions int, team string, streamingSlots int, field string) {
	t.Helper()
	found_player := false
	switch field {
	case "RosterMap":
		// RosterMap
		c := bt.RosterMap
		if len(c) == 0 {
			t.Errorf("%s is empty", field)
		}
		for _, player := range c {
			if player.GetAvgPoints() == 0 {
				t.Errorf("Player average points is 0")
			}
			if player.GetName() == name {
				found_player = true
				if player.GetValidPositions()[0] != position || len(player.GetValidPositions()) != num_positions {
					t.Errorf("Player position is incorrect")
				}
				if player.GetTeam() != team {
					t.Errorf("Player team is incorrect")
				}
			}
			if found_player {
				break
			}
		}
		if !found_player {
			t.Errorf("%s not found in %s", name, field)
		}
	case "FreeAgents":
		// FreeAgents
		c := bt.FreeAgents
		if len(c) == 0 {
			t.Errorf("%s is empty", field)
		}
		for _, player := range c {
			if player.GetName() == name {
				found_player = true
				if player.GetValidPositions()[0] != position || len(player.GetValidPositions()) != num_positions {
					t.Errorf("Player position is incorrect")
				}
				if player.GetTeam() != team {
					t.Errorf("Player team is incorrect")
				}
			}
			if found_player {
				break
			}
		}
		if !found_player {
			t.Errorf("%s not found in %s", name, field)
		}
	case "OptimalSlotting":
		// OptimalSlotting
		c := bt.OptimalSlotting
		if len(c) == 0 {
			t.Errorf("%s is not filled", field)
		}
		// Restrictiveness as scored by BaseTeam.ScoreRoster: PG/SG/SF/PF > G/F > C > UT
		score := map[string]int{"PG": 5, "SG": 5, "SF": 5, "PF": 5, "G": 4, "F": 4, "C": 3, "UT1": 2, "UT2": 2, "UT3": 2}

		// Make sure that players are slotted into the most restrictive positions: an optimal
		// slotting never leaves a player in a slot when one of his more restrictive valid
		// slots is empty, because moving him there alone would raise the score
		for dayIdx, day := range c {
			for pos, player := range day {
				if player.GetName() == "" {
					continue
				}
				for _, valid_pos := range player.GetValidPositions() {
					if score[valid_pos] > score[pos] && day[valid_pos].GetName() == "" {
						t.Errorf("day %d: %s is slotted at %s while more restrictive %s is empty", dayIdx, player.GetName(), pos, valid_pos)
					}
				}
			}
		}
	case "UnusedPositions":
		c := bt.UnusedPositions
		if len(c) == 0 {
			t.Errorf("%s is empty", field)
		}

		// Make sure that none of the UnusedPositions are in the OptimalSlotting
		for i, day := range c {
			for pos := range day {
				if player, ok := bt.OptimalSlotting[i][pos]; ok && player.GetName() != "" {
					t.Errorf("Unused position is in OptimalSlotting")
				}
			}
		}
	case "StreamablePlayers":
		c := bt.StreamablePlayers
		if len(c) == 0 {
			t.Errorf("%s is empty", field)
		}
		if len(c) != streamingSlots {
			t.Errorf("Expected %d streamable players, got %d", streamingSlots, len(c))
		}
		for _, player := range c {
			if player.GetName() == name {
				found_player = true
				if player.GetValidPositions()[0] != position || len(player.GetValidPositions()) != num_positions {
					t.Errorf("Player position is incorrect")
				}
				if player.GetTeam() != team {
					t.Errorf("Player team is incorrect")
				}
			}
			if found_player {
				break
			}
		}
		if !found_player {
			t.Errorf("%s not found in %s", name, field)
		}

		// Make sure that none of the StreamablePlayers are in the OptimalSlotting
		for _, day := range bt.OptimalSlotting {
			for _, player := range day {
				if player.GetName() == "" {
					continue
				}
				for _, streamable_player := range c {
					if player.GetName() == streamable_player.GetName() {
						t.Errorf("Streamable player is in OptimalSlotting")
					}
				}
			}
		}
	}
}
