package tests

import (
	"fmt"
	d "v2/data"
	l "v2/resources"
	"v2/team"
	"testing"
)

func TestBTInitWithData(t *testing.T) {
	d.InitSchedule("/Users/jameskendrick/Code/Projects/cv/features/lineup-generation/v2/static/schedule25-26.json")

	// Test the InitBaseTeam function with mock data
	week := 1
	threshold := 30.0
	roster := l.LoadFreeAgents("/Users/jameskendrick/Code/Projects/cv/features/lineup-generation/v2/resources/mock_roster.json")
	freeAgents := l.LoadFreeAgents("/Users/jameskendrick/Code/Projects/cv/features/lineup-generation/v2/resources/mock_freeagents.json")
	bt := team.InitBaseTeam(roster, freeAgents, week, threshold)

	// Validate fields
	BTFieldValidator(bt, t, "Anthony Edwards", "SG", 7, "MIN", threshold, "RosterMap")
	BTFieldValidator(bt, t, "Naz Reid", "PF", 6, "MIN", threshold, "FreeAgents")
	if bt.Week != week {
		t.Errorf("Week is incorrect")
	}
}

func TestBTInitMock(t *testing.T) {
	d.InitSchedule("/Users/jameskendrick/Code/Projects/cv/features/lineup-generation/v2/static/schedule25-26.json")

	// Test the InitBaseTeamMock function
	week := 1
	threshold := 32.0
	bt := team.InitBaseTeamMock(week, threshold)

	// Validate fields
	BTFieldValidator(bt, t, "Anthony Edwards", "SG", 7, "MIN", threshold, "RosterMap")
	BTFieldValidator(bt, t, "Naz Reid", "PF", 6, "MIN", threshold, "FreeAgents")
	if bt.Week != week {
		t.Errorf("Week is incorrect")
	}

	// Print the optimal slotting and the streamable players
	fmt.Println("Optimal Slotting")
	order := []string{"PG", "SG", "SF", "PF", "C", "G", "F", "UT1", "UT2", "UT3"}
	for i, day := range bt.OptimalSlotting {
		fmt.Println("Day", i)
		for _, pos := range order {
			if player, ok := day[pos]; ok && player.GetName() != "" {
				fmt.Println(pos, player.GetName())
			} else {
				fmt.Println(pos, "Empty")
			}
		}
		fmt.Println()
	}
	fmt.Println("Streamable Players")
	for _, player := range bt.StreamablePlayers {
		fmt.Println(player.GetName(), player.GetAvgPoints(), player.GetTeam(), player.GetValidPositions())
		for day := range d.ScheduleMap.GetWeekSchedule(week).TeamSchedules[player.GetTeam()] {
			fmt.Println("Playing day", day)
		}
	}

}

func TestBTPlayersToMap(t *testing.T) {
	// Test the PlayersToMap function
	roster := l.LoadFreeAgents("/Users/jameskendrick/Code/Projects/cv/features/lineup-generation/v2/resources/mock_roster.json")
	roster_map := d.PlayersToMap(roster)

	bt := &team.BaseTeam{
		RosterMap: roster_map,
	}

	// Validate fields
	BTFieldValidator(bt, t, "Anthony Edwards", "SG", 7, "MIN", 0.0, "RosterMap")
}

func TestBTOptimizeSlottingAndStreamablePlayers(t *testing.T) {
	d.InitSchedule("/Users/jameskendrick/Code/Projects/cv/features/lineup-generation/v2/static/schedule25-26.json")

	// Test the OptimizeSlotting function
	week := 1
	threshold := 30.0
	roster_map := l.LoadRosterMap("/Users/jameskendrick/Code/Projects/cv/features/lineup-generation/v2/resources/mock_roster.json")
	free_agents := l.LoadFreeAgents("/Users/jameskendrick/Code/Projects/cv/features/lineup-generation/v2/resources/mock_freeagents.json")
	bt := &team.BaseTeam{
		RosterMap: roster_map,
		FreeAgents: free_agents,
	}
	bt.OptimizeSlotting(week, threshold)

	// Validate field
	BTFieldValidator(bt, t, "Anthony Edwards", "SG", 7, "MIN", threshold, "OptimalSlotting")

}

func TestBTFindUnusedPositions(t *testing.T) {
	d.InitSchedule("/Users/jameskendrick/Code/Projects/cv/features/lineup-generation/v2/static/schedule25-26.json")

	// Test the FindUnusedPositions function
	week := 1
	threshold := 30.0
	roster_map := l.LoadRosterMap("/Users/jameskendrick/Code/Projects/cv/features/lineup-generation/v2/resources/mock_roster.json")
	free_agents := l.LoadFreeAgents("/Users/jameskendrick/Code/Projects/cv/features/lineup-generation/v2/resources/mock_freeagents.json")
	bt := &team.BaseTeam{
		RosterMap: roster_map,
		FreeAgents: free_agents,
	}
	bt.OptimizeSlotting(week, threshold)
	bt.FindUnusedPositions()

	// Validate field
	BTFieldValidator(bt, t, "", "", 0, "", threshold, "UnusedPositions")

	// Print unused positions
	pos_order := []string{"PG", "SG", "SF", "PF", "C", "G", "F", "UT1", "UT2", "UT3"}
	for i := 0; i < 6; i++ {
		day := bt.UnusedPositions[i]
		fmt.Println("Day", i)
		for _, pos := range pos_order {
			if val, ok := day[pos]; ok && val {
				fmt.Println(pos, "Unused")
			} else {
				fmt.Println(pos, "used by", bt.OptimalSlotting[i][pos].GetName())
			}
		}
	}
}

func BTFieldValidator(bt *team.BaseTeam, t *testing.T, name string, position string, num_positions int, team string, threshold float64, field string) {
	found_player := false
	switch field {
	case "RosterMap":
		// RosterMap
		c := bt.RosterMap
		fmt.Println("Testing", field)
		if len(c) == 0 {
			t.Errorf(field, "is empty");
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
	case "FreeAgents":
		// FreeAgents
		fmt.Println("Testing", field)
		c := bt.FreeAgents
		if len(c) == 0 {
			t.Errorf(field, "is empty")
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
	case "OptimalSlotting":
		// OptimalSlotting
		c := bt.OptimalSlotting
		fmt.Println("Testing", field)
		if len(c) < 5 {
			t.Errorf(field, "is not filled")
		}
		flexible_positions := []string{"UT1", "UT2", "UT3", "G", "F"}
		restrictive_positions := []string{"PG", "SG", "SF", "PF", "C"}
		restrictive_positions_map := make(map[string]bool)
		for _, pos := range restrictive_positions {
			restrictive_positions_map[pos] = true
		}

		// Make sure that players are slotted into the most restrictive positions
		for _, day := range c {
			// Make sure players in non-restrictive positions can't be slotted into more restrictive positions
			for _, pos := range flexible_positions {
				if player, ok := day[pos]; ok {
					// If the player is in a non-restrictive position make sure there is a player in their more restrictive ValidPositions
					for _, valid_pos := range player.GetValidPositions() {
						if _, ok := restrictive_positions_map[valid_pos]; ok {
							if player, ok := day[valid_pos]; ok && player.GetName() == "" {
								t.Errorf("Player is slotted into less restrictive position")
							}
						} else {
							// ValidPositions are ordered in descending order of restrictiveness so next positions will be less restrictive
							break
						}
					}
				}
			}
		}
	case "UnusedPositions":
		c := bt.UnusedPositions
		fmt.Println("Testing", field)
		if len(c) < 5 {
			t.Errorf(field, "is empty")
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
		fmt.Println("Testing", field)
		if len(c) == 0 {
			t.Errorf(field, "is empty")
		}
		for _, player := range c {
			if player.GetAvgPoints() > threshold {
				t.Errorf("Player average points is less than threshold")
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