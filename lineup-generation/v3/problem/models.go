package v3


type Problem struct {
	// Main inputs
	Schedule Schedule 
	Items []Item
	Constraints []Constraint
	ValueFunction ValueFunction

	// Cooldown
  RecoveryTime int
	OnCooldown map[string]int

	// Downstream effects
	HasDownstreamEffects bool

	// Prioritization
	PositionOrderMatters bool
}

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

type Constraint struct {
	Constraint func(Schedule) bool
}

type ValueFunction struct {
	Value func(Schedule) float64
}