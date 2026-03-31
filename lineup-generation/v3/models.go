package v3


type Schedule struct {
	Periods []Period
}

type Period struct {
	Positions map[string]Item
}

type Item struct {
	Name string
	Schedule [][]string // [period][position]
	Positions map[string]bool
	Value float64
}