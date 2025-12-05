package helpers

import (
	"fmt"
	"sort"
)


type SetupStateMetadata struct {
	roster             []Player
	streamable_players []Player
	optimal_slotting   map[int]map[string]Player
	unused_positions   map[int]map[string]bool
}

func (ssm *SetupStateMetadata) Print() {
	position_order := []string{"PG", "SG", "SF", "PF", "G", "F", "C", "UT1", "UT2", "UT3", "BE1", "BE2", "BE3"}

	for i := range len(ssm.optimal_slotting) {
		lineup := ssm.optimal_slotting[i]
		fmt.Println("Day:", i)
		fmt.Println(len(lineup))
		fmt.Println(len(ssm.unused_positions[i]))
		for _, position := range position_order {
			if player, ok := lineup[position]; ok {
				fmt.Println(position, player.Name, player.AvgPoints)
			} else {
				if _, ok := ssm.unused_positions[i][position]; ok {
					fmt.Println(position, "Unused")
				} else {
					fmt.Println(position, "--------")
				}
			}
		}
		fmt.Println()
	}
}

func InitSetupState(schedule *WeekSchedule, roster []Player, free_agents []Player, threshold float64) *SetupStateMetadata {

	ssm := &SetupStateMetadata{
		roster: roster,
		streamable_players: make([]Player, 0),
		optimal_slotting: make(map[int]map[string]Player),
		unused_positions: make(map[int]map[string]bool),
	}
	ssm.OptimizeSlotting(schedule, threshold)

	return ssm
}

// Finds available slots and players to experiment with on a roster when considering undroppable players and restrictive positions
func (ssm *SetupStateMetadata) OptimizeSlotting(schedule *WeekSchedule, threshold float64) {

	// Convert RosterMap to slices and abstract out IR spot. For the first day, passm all players to get_available_slots
	var streamable_players []Player
	var non_streamable_players []Player
	for _, player := range ssm.roster {

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
		return_table[i] = ssm.GetAvailableSlots(schedule, non_streamable_players, i)
	}

	// Sort the streamable players by average points
	sort.Slice(streamable_players, func(i, j int) bool {
		return streamable_players[i].AvgPoints > streamable_players[j].AvgPoints
	})
	ssm.streamable_players = streamable_players
	ssm.optimal_slotting = return_table
	ssm.FindUnusedPositions()
}

// Struct for keeping track of state acrossm recursive function calls to allow for early exit
type FitPlayersContext struct {
	BestLineup map[string]Player
	TopScore   int
	MaxScore   int
	EarlyExit  bool
}

// Function to get available slots for a given day
func (ssm *SetupStateMetadata) GetAvailableSlots(schedule *WeekSchedule, players []Player, day int) map[string]Player {

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

		// Create struct to keep track of state acrossm recursive function calls
		p_context := &FitPlayersContext{
			BestLineup: make(map[string]Player), 
			TopScore: 0,
			MaxScore: ssm.CalculateMaxScore(playing),
			EarlyExit: false,
		}
	
		// Recursive function call
		ssm.FitPlayers(playing, make(map[string]Player), position_order, p_context, 0)
	
		// Create response map and fill with best lineup or empty strings for unused positions except for bench spots
		response := make(map[string]Player)
		for _, pos := range position_order {

			if value, ok := p_context.BestLineup[pos]; ok {
				response[pos] = value
				continue
			}
		}

		return response
	}(playing)

	return optimal_slotting

}

// Recursive backtracking function to find most restrictive positions for players
func (ssm *SetupStateMetadata) FitPlayers(players []Player, cur_lineup map[string]Player, position_order []string, ctx *FitPlayersContext, index int) {

	// If we have found a lineup that has the max score, we can send returns to all other recursive calls
	if ctx.EarlyExit {
		return
	}
	
	// If all players have been given positions, check if the current lineup is better than the best lineup
	if len(players) == 0 {
		score := ssm.ScoreRoster(cur_lineup)
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
	if index >= len(position_order) {
		return // No more positions to try
	}
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

			ssm.FitPlayers(remaining_players, cur_lineup, position_order, ctx, index + 1) // Recurse

			delete(cur_lineup, position) // Backtrack
		}
	}

	// If we did not find a player for the position, advance to the next position
	if !found_player {
		ssm.FitPlayers(players, cur_lineup, position_order, ctx, index + 1) // Recurse
	}
}

// Function to score a roster based on restricitvenessm of positions
func (ssm *SetupStateMetadata) ScoreRoster(roster map[string]Player) int {

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

// Function to calculate the max restrictivenessm score for a given set of players
func (ssm *SetupStateMetadata) CalculateMaxScore(players []Player) int {

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
func (ssm *SetupStateMetadata) FindUnusedPositions() {

	positions := []string{"PG", "SG", "SF", "PF", "C", "G", "F", "UT1", "UT2", "UT3"}

	// Create map to keep track of unused positions
	unused_positions := make(map[int]map[string]bool)

	// Loop through each optimal slotting and add unused positions to map
	for day, lineup := range ssm.optimal_slotting {
		if unused_positions[day] == nil {
			unused_positions[day] = make(map[string]bool)
		}
		for _, pos := range positions {
			// If the position is empty, add it to the unused positions
			if _, ok := lineup[pos]; !ok {
				unused_positions[day][pos] = true
			}
		}
	}
	
	ssm.unused_positions = unused_positions
}

// Getter methods for testing
func (ssm *SetupStateMetadata) GetRoster() []Player {
	return ssm.roster
}

func (ssm *SetupStateMetadata) GetStreamablePlayers() []Player {
	return ssm.streamable_players
}

func (ssm *SetupStateMetadata) GetOptimalSlotting() map[int]map[string]Player {
	return ssm.optimal_slotting
}

func (ssm *SetupStateMetadata) GetUnusedPositions() map[int]map[string]bool {
	return ssm.unused_positions
}