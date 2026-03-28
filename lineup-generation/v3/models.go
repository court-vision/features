package v3


type Schedule struct {
	Periods []Period
}

type Period struct {
	Positions map[string]Item
}

type Item struct {
	Name string
	Schedule []map[string]bool
}