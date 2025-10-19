package helpers

// Struct for how to contruct Players using the returned player data
type Player struct {
	Name           string   `json:"name"`
	AvgPoints      float64  `json:"avg_points"`
	Team           string   `json:"team"`
	ValidPositions []string `json:"valid_positions"`
	Injured        bool     `json:"injured"`
}

func (p Player) PlaysPosition(position string) bool {
	for _, valid_position := range p.ValidPositions {
		if valid_position == position {
			return true
		}
	}
	return false
}

// Struct for organizing data on a player who has been dropped
type DroppedPlayer struct {
	Player 	  Player
	Countdown int
}