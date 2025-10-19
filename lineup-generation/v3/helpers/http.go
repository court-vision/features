package helpers

// Incoming request body to begin the lineup generation process
type Request struct {
	RosterData []Player     `json:"roster_data"`
	FreeAgentData []Player  `json:"free_agent_data"`
	Threshold float64   		`json:"threshold"`
	Week int    				    `json:"week"`
}

type Response struct {
	Lineup []Roster
	Improvement int
	Timestamp string
	Week int
	Threshold float64
}

type Roster struct {
	Day 	  	int
	Additions []Player
	Removals  []Player
	Roster	  map[string]Player
}