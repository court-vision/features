package population

import (
	"fmt"
	"math"
	"sort"
	"math/rand"
	d "v2/data"
	t "v2/team"
	u "v2/utils"
)

// Struct for chromosome for genetic algorithm
type Chromosome struct {
	Genes             []*Gene
	FitnessScore      int
	TotalAcquisitions int
	CumProbTracker    float64
	DroppedPlayers    map[string]d.DroppedPlayer
	CurStreamers      []d.Player
	Week              int
}

// Function to create a new chromosome
func InitChromosome(bt *t.BaseTeam) *Chromosome {
	
	// Create a new chromosome
	chromosome := &Chromosome{Genes: make([]*Gene, d.ScheduleMap.GetGameSpan(bt.Week)), 
		FitnessScore: 0, 
		TotalAcquisitions: 0, 
		CumProbTracker: 0.0, 
		DroppedPlayers: make(map[string]d.DroppedPlayer),
		CurStreamers: make([]d.Player, len(bt.StreamablePlayers)),
		Week: bt.Week,
	}

	// Make the initial streamers the current streamers
	copy(chromosome.CurStreamers, bt.StreamablePlayers)

	// Create a gene for each day in the week
	for i := range d.ScheduleMap.GetGameSpan(bt.Week) {
		gene := InitGene(bt, i)
		chromosome.Genes[i] = gene
	}

	return chromosome
}

// Function to insert random free agents into the chromosome
func (c *Chromosome) Populate(bt *t.BaseTeam, rng *rand.Rand) {

	// Insert streamable players into the genes
	for _, gene := range c.Genes {
		gene.InsertStreamablePlayers(bt)
	}

	// Insert random free agents into the genes
	for day, gene := range c.Genes {
		acq_count := min(len(bt.UnusedPositions[day]), (rng.Intn(5) / 2) + rng.Intn(2))

		// On the first day, make sure you can't drop initial streamers who are playing
		if non_playing_streamers_count := gene.Bench.GetLength(); day == 0 && acq_count > non_playing_streamers_count {
			acq_count = non_playing_streamers_count
		}

		// If the roster is full, don't make acquisitions
		num_open_posiitons := u.CountOpenPositions(gene.FreePositions)
		if num_open_posiitons == 0 {
			acq_count = 0
		}

		// If acq_count is more than the number of streamable players, take the number of streamable players
		if acq_count > len(bt.StreamablePlayers) {
			acq_count = len(bt.StreamablePlayers)
		}

		// Create a map of the current (old) streamers
		old_streamers := make(map[string]d.Player)
		for _, player := range c.CurStreamers {
			old_streamers[player.Name] = player
		}

		// Make acquisitions
		for range acq_count {
			free_agent := gene.FindRandomFreeAgent(bt, c, rng, d.Player{}); if free_agent.Name == "" {
				break
			}
			
			c.InsertFreeAgent(bt, day, free_agent)

		}

		// Go through the old streamers and find the ones that were dropped
		for _, old_player := range old_streamers {
			if !u.SliceContainsPlayer(c.CurStreamers, &old_player) {
				c.DroppedPlayers[old_player.Name] = d.DroppedPlayer{Player: old_player, Countdown: 3}
				gene.DroppedPlayers = append(gene.DroppedPlayers, old_player)
			}
		}

		// Go through the new players and find the ones that were added
		for _, new_player := range c.CurStreamers {
			if _, ok := old_streamers[new_player.Name]; !ok {
				gene.NewPlayers = append(gene.NewPlayers, new_player)
				gene.Acquisitions++
				c.TotalAcquisitions++
			}
		}


		// Decrement the countdown for dropped players
		c.DecrementDroppedPlayers()
	}
}

// Function to insert a free agent into the chromosome
func (c *Chromosome) InsertFreeAgent(bt *t.BaseTeam, day int, free_agent d.Player) bool {
	gene := c.Genes[day]

	// If it is the first day or there are streamers on the bench, drop the worst bench player and find the best positions for the new player
	if day == 0 || gene.Bench.GetLength() > 0 {

		dropped_player, ok := gene.DropWorstBenchPlayer(); if !ok {
			return false
		}

		c.RemoveStreamer(day, free_agent, dropped_player)
		c.SlotPlayer(bt, day, len(c.Genes),  free_agent)
	} else {
		// If there are no streamers on the bench (i.e. the roster is full), drop the worst playing streamer that the free agent can replace and find the best position for the new player

		// Find the worst current streamer that the free agent can replace
		player_to_drop := c.FindStreamerToDrop(day, free_agent); if player_to_drop == nil {
			fmt.Println("Error finding streamer to drop")
			return false
		}

		// Drop the worst streamer and add the free agent
		c.RemoveStreamer(day, free_agent, *player_to_drop)
		c.SlotPlayer(bt, day, len(c.Genes), free_agent)
	}

	return true
}

// Function to find the worst streamer to drop
func (c *Chromosome) FindStreamerToDrop(day int, player_to_add d.Player) *d.Player {
	sort.Slice(c.CurStreamers, func(i, j int) bool {
		return c.CurStreamers[i].AvgPoints < c.CurStreamers[j].AvgPoints
	})

	// If there are free posisitions that the incoming player can fill, just return the worst player
	for _, pos := range player_to_add.ValidPositions {
		if val, ok := c.Genes[day].FreePositions[pos]; ok && val {
			return &c.CurStreamers[0]
		}
	}

	// Otherwise, find the worst streamer that the incoming player can replace
	for _, streamer := range c.CurStreamers {

		// Get the streamers position for the day
		pos := c.Genes[day].GetPosOfPlayer(streamer)

		// Check if the incoming free agent can replace the streamer
		if pos != "" && pos != "BE" {
			for _, valid_pos := range player_to_add.ValidPositions {
				if valid_pos == pos {
					return &streamer
				}
			}
		}
	}

	return nil
}

// Function to remove a streamer from the entire chromosome
func (c *Chromosome) RemoveStreamer(day int, player_to_add d.Player, player_to_drop d.Player) {
	for _, gene := range c.Genes[day:] {
		gene.RemoveStreamer(player_to_drop)
	}

	// Remove the streamer from the current streamers by replacing based on index
	for i, player := range c.CurStreamers {
		if player.Name == player_to_drop.Name {
			c.CurStreamers[i] = player_to_add
			break
		}
	}
}

// Function to find slots for a free agent over the course of the week
func (c *Chromosome) SlotPlayer(bt *t.BaseTeam, start int, end int, free_agent d.Player) {
	for _, gene := range c.Genes[start:end] {
		gene.SlotPlayer(bt, free_agent)
	}
}


// Function to decrement the countdown for dropped players
func (c *Chromosome) DecrementDroppedPlayers() {
	for _, dropped_player := range c.DroppedPlayers {
		if dropped_player.Countdown > 0 {
			dropped_player.Countdown--
		} else {
			delete(c.DroppedPlayers, dropped_player.Player.Name)
		}
	}
}


// Function to mutate a chromosome
func (c *Chromosome) Mutate(bt *t.BaseTeam, prob float64, rng *rand.Rand) (d.Player, d.Player, int, int) {

	// Get random number to determine if the chromosome will mutate
	rand_num := rng.Float64(); if rand_num > prob {
		return d.Player{}, d.Player{}, 0, 0
	}

	player_to_drop, pos, start, end := c.FindRandomPlayerToDrop(rng); if player_to_drop.Name == "" || start == -1 {
		return d.Player{}, d.Player{}, 0, 0
	}
	// Add the dropped player to the dropped players map for the start day
	c.DroppedPlayers[player_to_drop.Name] = d.DroppedPlayer{Player: player_to_drop, Countdown: 3}

	// Free the position of the player to drop
	if pos != "BE" {
		c.Genes[start].FreePositions[pos] = true
	}
	player_to_add := c.Genes[start].FindRandomFreeAgent(bt, c, rng, player_to_drop); if player_to_add.Name == "" {
		return d.Player{}, d.Player{}, 0, 0
	}


	// Drop the player to drop and add the player to add
	for i := start; i < end; i++ {

		// For each day, decrement the countdown for the dropped player
		if dropped_player, ok := c.DroppedPlayers[player_to_drop.Name]; ok {
			if dropped_player.Countdown > 0 {
				dropped_player.Countdown--
			} else {
				delete(c.DroppedPlayers, player_to_drop.Name)
			}
		}

		// Create a copy of the current streamers
		old_streamers := make(map[string]d.Player)
		for _, player := range c.CurStreamers {
			old_streamers[player.Name] = player
		}

		c.Genes[i].RemoveStreamer(player_to_drop)
		c.Genes[i].SlotPlayer(bt, player_to_add)

	}

	// If the player to add got in to the gene on the start day, put him in the NewPlayers list in the place of the player to drop
	if c.Genes[start].IsPlayerInGene(player_to_add) {
		for i, player := range c.Genes[start].NewPlayers {
			if player.Name == player_to_drop.Name {
				c.Genes[start].NewPlayers[i] = player_to_add
				break
			}
		}

		// We don't need to increment acquisitions because we should have dropped a player as part of the mutation transaction
	}

	// If the new player is still in the gene at the end of the week, add him to CurStreamers
	if c.Genes[len(c.Genes)-1].IsPlayerInGene(player_to_add) {
		for i, player := range c.CurStreamers {
			if player.Name == player_to_drop.Name {
				c.CurStreamers[i] = player_to_add
				break
			}
		}
	}

	return player_to_drop, player_to_add, start, end
}

// Function to find a random player to drop
func (c *Chromosome) FindRandomPlayerToDrop(rng *rand.Rand) (d.Player, string, int, int) {

	start := 0
	trials := 0
	test_start := rng.Intn(len(c.Genes))
	for start == 0 && trials < len(c.Genes) {
		if c.Genes[test_start].Acquisitions > 0 {
			start = test_start
			break
		} else {
			test_start = rng.Intn(len(c.Genes))
			trials++
		}
	}
	if start == 0 {
		return d.Player{}, "", -1, -1
	}

	player_to_drop := c.Genes[start].NewPlayers[rng.Intn(len(c.Genes[start].NewPlayers))]

	// Find the day that the player to drop is no longer in the gene
	end := len(c.Genes)
	for i := start; i < len(c.Genes); i++ {
		if !c.Genes[i].IsPlayerInGene(player_to_drop) {
			end = i
			break
		}
	}

	return player_to_drop, c.Genes[start].GetPosOfPlayer(player_to_drop), start, end

}

// Function to add back the non-streamable players to the chromosome for returning
func (c *Chromosome) AddBackNonStreamablePlayers(bt *t.BaseTeam) {
	for day, gene := range c.Genes {
		for pos, player := range bt.OptimalSlotting[day] {
			if player.Name != "" {
				gene.Roster[pos] = player
			}
		}
	}
}


// -------------------------- UTILS --------------------------

// Function to print the chromosome
func (c *Chromosome) Print() {
	order := []string{"PG", "SG", "SF", "PF", "C", "G", "F", "UT1", "UT2", "UT3"}


	fmt.Println("Total Acquisitions:", c.TotalAcquisitions)
	for i := 0; i < len(c.Genes); i++ {
		gene := c.Genes[i]
		fmt.Println("Day", i)
		fmt.Println("New Players", gene.NewPlayers)
		for _, pos := range order {
			if val, ok := gene.FreePositions[pos]; ok && val {
				fmt.Println(pos, "Unused")
			} else if player, ok := gene.Roster[pos]; ok && player.Name != "" {
				fmt.Println(pos, gene.Roster[pos].Name, gene.Roster[pos].AvgPoints)
			} else {
				fmt.Println(pos, "--------")
			}
		}
		fmt.Println("Bench")
		for _, player := range gene.Bench.Players {
			fmt.Println(player.Name)
		}
		fmt.Println()
	}
		
}

// Function to score the fitness of the chromosome
func (c *Chromosome) ScoreFitness() {

	fitness_score := 0.0
	penalty_factor := 1.0

	if c.TotalAcquisitions > d.ScheduleMap.GetGameSpan(c.Week) + 1 {
		penalty_factor = 1.0 / math.Pow(1.3, float64(c.TotalAcquisitions - (d.ScheduleMap.GetGameSpan(c.Week) + 1)))
	}
	for _, gene := range c.Genes {
		for _, player := range gene.Roster {
			fitness_score += player.AvgPoints
		}
	}

	c.FitnessScore = int(fitness_score * penalty_factor)
}

// Function to return a slimmed down, defreferenced version of the chromosome
func (c *Chromosome) Slim() []u.SlimGene {
	slim_chromosome := make([]u.SlimGene, len(c.Genes))
	for i, gene := range c.Genes {
		slim_chromosome[i] = gene.Slim()
	}
	return slim_chromosome
}
