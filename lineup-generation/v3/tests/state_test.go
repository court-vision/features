package tests

import (
	"testing"

	h "v3/helpers"
)


// Mock data for testing
func createMockSchedule() *h.WeekSchedule {
	return &h.WeekSchedule{
		StartDate: "10/21/2025",
		EndDate:   "10/26/2025",
		GameSpan:  6,
		TeamSchedules: map[string][]int{
			"MIL": {1, 3, 5},    // Games on days 1, 3, 5
			"PHX": {1, 3, 5},    // Games on days 1, 3, 5
			"PHI": {1, 4},       // Games on days 1, 4
			"ORL": {1, 3, 4},    // Games on days 1, 3, 4
			"CHA": {1, 4, 5},    // Games on days 1, 4, 5
		},
	}
}

func createMockRoster() []h.Player {
	return []h.Player{
		// High-scoring players (above threshold) - should be non-streamable
		{
			Name:           "Giannis Antetokounmpo",
			AvgPoints:      65.0,
			Team:           "MIL",
			ValidPositions: []string{"PF", "C", "F"},
			Injured:        false,
		},
		{
			Name:           "Devin Booker",
			AvgPoints:      60.0,
			Team:           "PHX",
			ValidPositions: []string{"PG", "SG", "G"},
			Injured:        false,
		},
		{
			Name:           "Tyrese Maxey",
			AvgPoints:      55.0,
			Team:           "PHI",
			ValidPositions: []string{"PG", "SG", "G"},
			Injured:        false,
		},
		{
			Name:           "Paolo Banchero",
			AvgPoints:      50.0,
			Team:           "ORL",
			ValidPositions: []string{"SF", "PF", "F"},
			Injured:        false,
		},
		
		// Streamable players
		{
			Name:           "Paul George",
			AvgPoints:      25.0,
			Team:           "PHI",
			ValidPositions: []string{"SF", "PF", "F"},
			Injured:        false,
		},
		{
			Name:           "Brandon Miller",
			AvgPoints:      25.0,
			Team:           "CHA",
			ValidPositions: []string{"SF", "SG", "G"},
			Injured:        false,
		},
		
		// Injured player - should be excluded
		{
			Name:           "Injured Player",
			AvgPoints:      20.0,
			Team:           "CHA",
			ValidPositions: []string{"SF", "F"},
			Injured:        true,
		},
	}
}

func createMockFreeAgents() []h.Player {
	return []h.Player{
		{
			Name:           "Free Agent PG",
			AvgPoints:      15.2,
			Team:           "OKC",
			ValidPositions: []string{"PG", "G"},
			Injured:        false,
		},
		{
			Name:           "Free Agent C",
			AvgPoints:      11.8,
			Team:           "LAL",
			ValidPositions: []string{"C", "PF"},
			Injured:        false,
		},
		{
			Name:           "Free Agent SF",
			AvgPoints:      9.3,
			Team:           "HOU",
			ValidPositions: []string{"SF", "F"},
			Injured:        false,
		},
		{
			Name:           "Free Agent SG",
			AvgPoints:      6.7,
			Team:           "NYK",
			ValidPositions: []string{"SG", "G"},
			Injured:        false,
		},
	}
}

// TestInitSetupState tests the main InitSetupState function
func TestInitSetupStateMetadata(t *testing.T) {
	schedule := createMockSchedule()
	roster := createMockRoster()
	freeAgents := createMockFreeAgents()
	threshold := 30.0

	setup_state := h.InitSetupState(schedule, roster, freeAgents, threshold)
	setup_state.Print()
}

func TestInitState(t *testing.T) {
	schedule := createMockSchedule()
	roster := createMockRoster()
	freeAgents := createMockFreeAgents()
	threshold := 30.0

	setup_state := h.InitSetupState(schedule, roster, freeAgents, threshold)
	state := h.InitState(schedule, setup_state, freeAgents)
	state.Print()
}