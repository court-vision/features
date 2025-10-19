package models


// Struct for how to contruct Players using the returned player data
type Player struct {
	Name           string   `json:"name"`
	AvgPoints      float64  `json:"avg_points"`
	Team           string   `json:"team"`
	ValidPositions []string `json:"valid_positions"`
	Injured        bool     `json:"injured"`
}

// Struct for organizing data on a player who has been dropped
type DroppedPlayer struct {
	Player 	  Player
	Countdown int
}

type Roster struct {
	Day 	  	int
	Additions []Player
	Removals  []Player
	Roster	  map[string]Player
}