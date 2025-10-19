package helpers

import (
	"sort"
)


type SetupState struct {
	roster []Player
	streamable_players []Player
	optimal_slotting map[int]map[string]Player
	unused_positions map[int]map[string]bool
	score int
}

func InitSetupState(schedule *WeekSchedule, roster []Player, threshold float64) *SetupState {

	ss := &SetupState{}
	ss.roster = roster
	ss.streamable_players = make([]Player, 0)
	ss.optimal_slotting = make(map[int]map[string]Player)
	ss.unused_positions = make(map[int]map[string]bool)
	ss.score = 0

	ss.OptimizeSlotting(schedule, threshold)
	
	return ss
}

// Finds available slots and players to experiment with on a roster when considering undroppable players and restrictive positions
func (ss *SetupState) OptimizeSlotting(schedule *WeekSchedule, threshold float64) {

	// Convert RosterMap to slices and abstract out IR spot. For the first day, pass all players to get_available_slots
	var streamable_players []Player
	var non_streamable_players []Player
	for _, player := range ss.roster {

		if player.Injured {
			continue
		}

		if player.AvgPoints > threshold {
			non_streamable_players = append(non_streamable_players, player)
		} else {
			streamable_players = append(streamable_players, player)
		}
	}

	// Sort good players by average points
	sort.Slice(non_streamable_players, func(i, j int) bool {
		return non_streamable_players[i].AvgPoints > non_streamable_players[j].AvgPoints
	})

	return_table := make(map[int]map[string]Player)

	// Fill return table and put extra IR players on bench
	for i := 0; i < schedule.GetGameSpan(); i++ {
		return_table[i] = ss.GetAvailableSlots(schedule, non_streamable_players, i)
	}

	// Sort the streamable players by average points
	sort.Slice(streamable_players, func(i, j int) bool {
		return streamable_players[i].AvgPoints > streamable_players[j].AvgPoints
	})
	ss.streamable_players = streamable_players
	ss.optimal_slotting = return_table
}

// Struct for keeping track of state across recursive function calls to allow for early exit
type FitPlayersContext struct {
	BestLineup map[string]Player
	TopScore   int
	MaxScore   int
	EarlyExit  bool
}

// Function to get available slots for a given day
func (ss *SetupState) GetAvailableSlots(schedule *WeekSchedule, players []Player, day int) map[string]Player {

	// Priority order of most restrictive positions to funnel streamers into flexible positions
	position_order := []string{"PG", "SG", "SF", "PF", "G", "F", "C", "UT1", "UT2", "UT3", "BE1", "BE2", "BE3"}
	
	var playing []Player

	for _, player := range players {

		// Checks if the player is playing on the given day
		if schedule.IsPlaying(day, player.Team){
			playing = append(playing, player)
		}
	}

	// Find most restrictive positions for players playing
	optimal_slotting := func (playing []Player) map[string]Player {

		sort.Slice(playing, func(i, j int) bool {
			return len(playing[i].ValidPositions) < len(playing[j].ValidPositions)
		})

		// Create struct to keep track of state across recursive function calls
		p_context := &FitPlayersContext{
			BestLineup: make(map[string]Player), 
			TopScore: 0,
			MaxScore: ss.CalculateMaxScore(playing),
			EarlyExit: false,
		}
	
		// Recursive function call
		ss.FitPlayers(playing, make(map[string]Player), position_order, p_context, 0)
	
		// Create response map and fill with best lineup or empty strings for unused positions except for bench spots
		response := make(map[string]Player)
		filter := map[string]bool{"BE1": true, "BE2": true, "BE3": true}
		for _, pos := range position_order {

			if value, ok := p_context.BestLineup[pos]; ok {
				response[pos] = value
				continue
			}
			if _, ok := filter[pos]; !ok {
				response[pos] = Player{}
			}
		}

		return response
	}(playing)

	return optimal_slotting

}

// Recursive backtracking function to find most restrictive positions for players
func (ss *SetupState) FitPlayers(players []Player, cur_lineup map[string]Player, position_order []string, ctx *FitPlayersContext, index int) {

	// If we have found a lineup that has the max score, we can send returns to all other recursive calls
	if ctx.EarlyExit {
		return
	}
	
	// If all players have been given positions, check if the current lineup is better than the best lineup
	if len(players) == 0 {
		score := ss.ScoreRoster(cur_lineup)
		// fmt.Println("Score:", score, "Max score:", ctx.MaxScore)
		if score > ctx.TopScore {
			ctx.TopScore = score
			ctx.BestLineup = make(map[string]Player)
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
			var remaining_players []Player
			for _, p := range players {
				if p.Name != player.Name {
					remaining_players = append(remaining_players, p)
				}
			}

			ss.FitPlayers(remaining_players, cur_lineup, position_order, ctx, index + 1) // Recurse

			delete(cur_lineup, position) // Backtrack
		}
	}

	// If we did not find a player for the position, advance to the next position
	if !found_player {
		ss.FitPlayers(players, cur_lineup, position_order, ctx, index + 1) // Recurse
	}
}

// Function to score a roster based on restricitveness of positions
func (ss *SetupState) ScoreRoster(roster map[string]Player) int {

	// Scoring system - hardcoded for performance
	score_map := map[string]int{
		"PG":  5, "SG":  5, "SF":  5, "PF":  5,  // Most restrictive positions
		"G":   4, "F":   4,
		"C":   3,
		"UT1": 2, "UT2": 2, "UT3": 2,
		"BE1": 1, "BE2": 1, "BE3": 1,            // Least restrictive positions
	}

	// Score roster
	score := 0
	for pos := range roster {
		score += score_map[pos]
	}

	return score
}

// Function to calculate the max restrictiveness score for a given set of players
func (ss *SetupState) CalculateMaxScore(players []Player) int {

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
func (ss *SetupState) FindUnusedPositions() {

	// Order that the slice should be in
	order := []string{"PG", "SG", "SF", "PF", "C", "G", "F", "UT1", "UT2", "UT3"}

	// Create map to keep track of unused positions
	unused_positions := make(map[int]map[string]bool)

	// Loop through each optimal slotting and add unused positions to map
	for day, lineup := range ss.optimal_slotting {

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
	
	ss.unused_positions = unused_positions
}

// Function to calculate the score of the optimal players for the week
func (ss *SetupState) CalculateOptimalScore() {
	total_score := 0.0
	for _, lineup := range ss.optimal_slotting {
		for _, player := range lineup {
			total_score += player.AvgPoints
		}
	}
	ss.score = int(total_score)
}