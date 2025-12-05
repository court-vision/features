package team

import (
	"sort"
	d "v2/data"
	l "v2/resources"
)

type BaseTeam struct {
	RosterMap         map[string]d.Player
	FreeAgents        []d.Player
	OptimalSlotting   map[int]map[string]d.Player
	UnusedPositions   map[int]map[string]bool
	StreamablePlayers []d.Player
	Score             int
	Week              int
}

func InitBaseTeam(rosterData []d.Player, freeAgentData []d.Player, week int, threshold float64) *BaseTeam {

	bt := &BaseTeam{}
	bt.RosterMap = d.PlayersToMap(rosterData)
	bt.FreeAgents = freeAgentData
	bt.OptimizeSlotting(week, threshold)
	bt.FindUnusedPositions()
	bt.CalculateOptimalScore()
	bt.Week = week

	return bt
}

func InitBaseTeamMock(week int, threshold float64) *BaseTeam {

	bt := &BaseTeam{}
	bt.RosterMap = l.LoadRosterMap("/Users/jameskendrick/Code/Projects/cv/features/lineup-generation/v2/resources/mock_roster.json")
	bt.FreeAgents = l.LoadFreeAgents("/Users/jameskendrick/Code/Projects/cv/features/lineup-generation/v2/resources/mock_freeagents.json")
	bt.OptimizeSlotting(week, threshold)
	bt.FindUnusedPositions()
	bt.CalculateOptimalScore()
	bt.Week = week

	return bt
}


// Finds available slots and players to experiment with on a roster when considering undroppable players and restrictive positions
func (t *BaseTeam) OptimizeSlotting(week int, threshold float64) {

	// Convert RosterMap to slices and abstract out IR spot. For the first day, pass all players to get_available_slots
	var streamable_players []d.Player
	var sorted_good_players []d.Player
	for _, player := range t.RosterMap {

		if player.Injured {
			continue
		}

		if player.AvgPoints > threshold {
			sorted_good_players = append(sorted_good_players, player)
		} else {
			streamable_players = append(streamable_players, player)
		}
	}

	// Sort good players by average points
	sort.Slice(sorted_good_players, func(i, j int) bool {
		return sorted_good_players[i].AvgPoints > sorted_good_players[j].AvgPoints
	})

	return_table := make(map[int]map[string]d.Player)

	// Fill return table and put extra IR players on bench
	for i := range d.ScheduleMap.GetGameSpan(week) {
		return_table[i] = t.GetAvailableSlots(sorted_good_players, i, week)
	}

	// Sort the streamable players by average points
	sort.Slice(streamable_players, func(i, j int) bool {
		return streamable_players[i].AvgPoints > streamable_players[j].AvgPoints
	})
	t.StreamablePlayers = streamable_players
	t.OptimalSlotting = return_table
}

// Struct for keeping track of state across recursive function calls to allow for early exit
type FitPlayersContext struct {
	BestLineup map[string]d.Player
	TopScore   int
	MaxScore   int
	EarlyExit  bool
}

// Function to get available slots for a given day
func (t *BaseTeam) GetAvailableSlots(players []d.Player, day int, week int) map[string]d.Player {

	// Priority order of most restrictive positions to funnel streamers into flexible positions
	position_order := []string{"PG", "SG", "SF", "PF", "G", "F", "C", "UT1", "UT2", "UT3", "BE1", "BE2", "BE3"} // For players playing
	
	var playing []d.Player

	for _, player := range players {

		// Checks if the player is playing on the given day
		if d.ScheduleMap.IsPlaying(week, day, player.Team){
			playing = append(playing, player)
		}
	}

	// Find most restrictive positions for players playing
	optimal_slotting := func (playing []d.Player) map[string]d.Player {

		sort.Slice(playing, func(i, j int) bool {
			return len(playing[i].ValidPositions) < len(playing[j].ValidPositions)
		})

		// Create struct to keep track of state across recursive function calls
		max_score := t.CalculateMaxScore(playing)
		p_context := &FitPlayersContext{
			BestLineup: make(map[string]d.Player), 
			TopScore: 0, 
			MaxScore: max_score, 
			EarlyExit: false,
		}
	
		// Recursive function call
		t.FitPlayers(playing, make(map[string]d.Player), position_order, p_context, 0)
	
		// Create response map and fill with best lineup or empty strings for unused positions except for bench spots
		response := make(map[string]d.Player)
		filter := map[string]bool{"BE1": true, "BE2": true, "BE3": true}
		for _, pos := range position_order {

			if value, ok := p_context.BestLineup[pos]; ok {
				response[pos] = value
				continue
			}
			if _, ok := filter[pos]; !ok {
				response[pos] = d.Player{}
			}
		}

		return response
	
	}(playing)

	return optimal_slotting

}

// Recursive backtracking function to find most restrictive positions for players
func (t *BaseTeam) FitPlayers(players []d.Player, cur_lineup map[string]d.Player, position_order []string, ctx *FitPlayersContext, index int) {

	// If we have found a lineup that has the max score, we can send returns to all other recursive calls
	if ctx.EarlyExit {
		return
	}
	
	// If all players have been given positions, check if the current lineup is better than the best lineup
	if len(players) == 0 {
		score := t.ScoreRoster(cur_lineup)
		// fmt.Println("Score:", score, "Max score:", ctx.MaxScore)
		if score > ctx.TopScore {
			ctx.TopScore = score
			ctx.BestLineup = make(map[string]d.Player)
			for key, value := range cur_lineup {
				ctx.BestLineup[key] = value
			}
		}
		if score == ctx.MaxScore {
			ctx.EarlyExit = true
		}
		return
	}

	// If we have not gone through all players, try to fit the rest of the players in the lineup
	position := position_order[index]
	found_player := false
	for _, player := range players {
		if player.PlaysPosition(position) {
			found_player = true
			cur_lineup[position] = player

			// Remove player from players slice
			var remaining_players []d.Player

			for _, p := range players {
				if p.Name != player.Name {
					remaining_players = append(remaining_players, p)
				}
			}

			t.FitPlayers(remaining_players, cur_lineup, position_order, ctx, index + 1) // Recurse

			delete(cur_lineup, position) // Backtrack
		}
	}

	// If we did not find a player for the position, advance to the next position
	if !found_player {
		t.FitPlayers(players, cur_lineup, position_order, ctx, index + 1) // Recurse
	}
}

// Function to score a roster based on restricitveness of positions
func (t *BaseTeam) ScoreRoster(roster map[string]d.Player) int {

	// Scoring system
	score_map := make(map[string]int)

	scoring_groups := [][]string{{"PG", "SG", "SF", "PF"}, {"G", "F"}, {"C"}, {"UT1", "UT2", "UT3"}, {"BE1", "BE2", "BE3"}}
	for score, group := range scoring_groups {
		for _, position := range group {
			score_map[position] = 5 - score
		}
	}

	// Score roster
	score := 0
	for pos := range roster {
		score += score_map[pos]
	}

	return score
}

// Function to calculate the max restrictiveness score for a given set of players
func (t *BaseTeam) CalculateMaxScore(players []d.Player) int {

	size := len(players)

	// Max score calulation corresponding with scoring_groups in score_roster
	switch {
		case size <= 4:
			return size * 5
		case size <= 6:
			return 20 + ((size - 4) * 4)
		case size <= 7:
			return 28 + ((size - 6) * 3)
		case size <= 10:
			return 31 + ((size - 7) * 2)
		default:
			return 37 + (size - 10)
	}
}

// Function to get the unused positions from the optimal slotting for good players playing for the week
func (t *BaseTeam) FindUnusedPositions() {

	// Order that the slice should be in
	order := []string{"PG", "SG", "SF", "PF", "C", "G", "F", "UT1", "UT2", "UT3"}

	// Create map to keep track of unused positions
	unused_positions := make(map[int]map[string]bool)

	// Loop through each optimal slotting and add unused positions to map
	for day, lineup := range t.OptimalSlotting {

		// Initialize map for day if it doesn't exist
		if unused_positions[day] == nil {
			unused_positions[day] = make(map[string]bool)
		}
		
		for _, pos := range order {
			
			// If the position is empty, add it to the unused positions
			if player := lineup[pos]; player.Name == "" {
				unused_positions[day][pos] = true
			}
		}
	}
	
	t.UnusedPositions = unused_positions
}

// Function to calculate the score of the optimal players for the week
func (t *BaseTeam) CalculateOptimalScore() {
	total_score := 0.0
	for _, lineup := range t.OptimalSlotting {
		for _, player := range lineup {
			total_score += player.AvgPoints
		}
	}
	t.Score = int(total_score)
}