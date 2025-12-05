package helpers

import "fmt"


type Lineup struct {
	roster 		map[string]Player
	bench 		[]Player
	additions []Player
	removals 	[]Player
	score 		float64
}

// NewLineup creates a new Lineup with initialized fields
func NewLineup(unused_positions map[string]bool) Lineup {
	lineup := Lineup{
		roster:    make(map[string]Player),
		bench:     make([]Player, 0),
		additions: make([]Player, 0),
		removals:  make([]Player, 0),
		score:     0.0,
	}
	for position := range unused_positions {
		lineup.roster[position] = Player{}
	}
	return lineup
}

func (l *Lineup) Print() {
	position_order := []string{"PG", "SG", "SF", "PF", "C", "G", "F", "UT1", "UT2", "UT3", "BE1", "BE2", "BE3"}
	for _, position := range position_order {
		if player, ok := l.roster[position]; ok {
			fmt.Println(position, player.Name, player.AvgPoints)
		} else {
			fmt.Println(position, "--------")
		}
	}
	fmt.Println("Bench:", l.bench)
	fmt.Println("Score:", l.score)
	fmt.Println()
}

func (l *Lineup) GetRoster() map[string]Player {
	return l.roster
}

func (l *Lineup) GetBench() []Player {
	return l.bench
}

func (l *Lineup) SlotStreamer(streamer Player) bool {

	// Priority order of most restrictive positions to funnel streamers into flexible positions
	position_order := []string{"PG", "SG", "SF", "PF", "G", "F", "C", "UT1", "UT2", "UT3", "BE1", "BE2", "BE3"}

	found_position := false
	for _, position := range position_order {
		// If we position is 
		if player, ok := l.roster[position]; ok && streamer.PlaysPosition(position) {
			l.roster[position] = streamer
			found_position = true
			l.score += streamer.AvgPoints
			break
		}
	}

	return found_position
}

type State struct {
	day        	      int
	score 		        int
	acq_left    	    int
	lineups      	    []Lineup
	free_agents    	  []Player
	current_streamers []Player
	dropped_players 	[]DroppedPlayer
}


func (s *State) Print() {

	fmt.Println("Start Day:", s.day)
	fmt.Println("Current Streamers:", s.current_streamers)
	fmt.Println("--------------------------------")
	for i := range len(s.lineups) {
		fmt.Println("Day:", i)
		s.lineups[i].Print()
	}
	fmt.Println()
}


func InitState(schedule *WeekSchedule, ssm *SetupStateMetadata, free_agents []Player) *State {
	state := &State{
		day: 0,
		score: 0,
		acq_left: schedule.GetGameSpan(),
		lineups: make([]Lineup, schedule.GetGameSpan()),
		free_agents: make([]Player, 0),
		current_streamers: make([]Player, 0),
		dropped_players: make([]DroppedPlayer, 0),
	}
	
	// Initialize each lineup with proper structure
	for i := range state.lineups {
		state.lineups[i] = NewLineup(ssm.unused_positions[i])
	}
	
	// Set the current streamers and free agents
	state.current_streamers = ssm.GetStreamablePlayers()
	state.free_agents = free_agents
	// Slot the streamers into the lineup
	state.SlotStreamers(schedule, false) // Don't decrement acq_left since these are players that are already on the roster

	// Score the lineup
	state.ScoreLineup()

	// This now serves as the initial (root) state for the beam search algorithm
	return state
}


func (s *State) SlotStreamers(schedule *WeekSchedule, decrement_acq_left bool) {
	// Note: streamers are already sorted by average points

	for _, streamer := range s.current_streamers {
		for _, day := range schedule.GetTeamSchedule(streamer.Team) {
			if day >= s.day {
				if ok := s.lineups[day].SlotStreamer(streamer); !ok {
					s.lineups[day].bench = append(s.lineups[day].bench, streamer)
				}
			}
		}
		// Regardles of whether the streamer was slotted, decrement the acq_left
		if decrement_acq_left {
			s.acq_left--
		}
	}
}

func (s *State) ScoreLineup() {
	total_score := 0.0
	for _, lineup := range s.lineups {
		total_score += lineup.score
	}
	s.score = int(total_score)
}

// Getter methods for testing
func (s *State) GetDay() int {
	return s.day
}

func (s *State) GetScore() int {
	return s.score
}

func (s *State) GetAcqLeft() int {
	return s.acq_left
}

func (s *State) GetLineups() []Lineup {
	return s.lineups
}

func (s *State) GetFreeAgents() []Player {
	return s.free_agents
}

func (s *State) GetCurrentStreamers() []Player {
	return s.current_streamers
}

func (s *State) GetDroppedPlayers() []DroppedPlayer {
	return s.dropped_players
}