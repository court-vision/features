package tests

// import (
// 	"fmt"
// 	"math/rand"
// 	"time"
// 	"sync"
// 	"testing"
// 	"v2/team"
// 	d "v2/data"
// 	p "v2/population"
// )

// func TestOptimizeStreaming(t *testing.T) {
// 	start := time.Now()
// 	d.InitSchedule("/Users/jameskendrick/Code/cv/features/lineup-generation/v2/static/schedule.json")

// 	// bt := team.InitBaseTeamMock("16", 34.0)
// 	// week := "9"
// 	// bt := team.InitBaseTeam(424233486, "", "", "James's Scary Team", 2024, 100, week, 31.0)

// 	// // Create new populations
// 	// ev1 := p.InitPopulation(bt, 25)
// 	// ev2 := p.InitPopulation(bt, 25)
	
// 	// // Evolve the populations concurrently
// 	// var wg sync.WaitGroup
// 	// wg.Add(2)
// 	// go func() {
// 	// 	defer wg.Done()
// 	// 	for i := 0; i < 10; i++ {
// 	// 		ev1.Evolve(bt)
// 	// 	}
// 	// }()
// 	// go func() {
// 	// 	defer wg.Done()
// 	// 	for i := 0; i < 10; i++ {
// 	// 		ev2.Evolve(bt)
// 	// 	}
// 	// }()
// 	// wg.Wait()
	
// 	// // Combine the populations
// 	// ev1.Population = append(ev1.Population, ev2.Population...)
// 	// ev1.NumChromosomes = len(ev1.Population)
	
// 	// // Evolve the combined population
// 	// for i := 0; i < 10; i++ {
// 	// 	ev1.Evolve(bt)
// 	// }

// 	// if len(ev1.Population) != 50 {
// 	// 	t.Errorf("Incorrect number of chromosomes")
// 	// }

// 	// // Get the initial fitness score
// 	// base_chromosome := p.InitChromosome(bt)
// 	// for _, gene := range base_chromosome.Genes {
// 	// 	gene.InsertStreamablePlayers(bt)
// 	// }
// 	// base_chromosome.ScoreFitness()

// 	// ev1.SortByFitness()

// 	// // Make sure NewPlayer count and DropPlayer count are correct
// 	// for _, chromosome := range ev1.Population {
// 	// 	for _, gene := range chromosome.Genes {
// 	// 		if len(gene.NewPlayers) != len(gene.DroppedPlayers) {
// 	// 			t.Errorf("NewPlayer count does not match DropPlayer count")
// 	// 		}
// 	// 	}
// 	// }

// 	// // Print the best chromosome
// 	// best_chromosome := ev1.Population[ev1.NumChromosomes-1]

// 	// // Go through the chromosome to make sure Additions and DroppedPlayers are correct
// 	// for _, gene := range best_chromosome.Genes {
// 	// 	for _, player := range gene.NewPlayers {
// 	// 		if player.Name == "" {
// 	// 			t.Errorf("Player name is empty")
// 	// 		}
// 	// 	}
// 	// 	for _, player := range gene.DroppedPlayers {
// 	// 		if player.Name == "" {
// 	// 			t.Errorf("Player name is empty")
// 	// 		}
// 	// 	}
// 	// }


// 	// fmt.Println(bt.Score + best_chromosome.FitnessScore, "vs", bt.Score + base_chromosome.FitnessScore, "diff", best_chromosome.FitnessScore - base_chromosome.FitnessScore)
// 	// best_chromosome.AddBackNonStreamablePlayers(bt)
// 	// best_chromosome.Print()
// 	// elapsed := time.Since(start)
// 	// fmt.Println("Time to run InitPopulation: ", elapsed)

// 	// Create new populations
// 	ev1 := p.InitPopulation(bt, 20)
// 	ev2 := p.InitPopulation(bt, 20)

// 	// Evolve the populations concurrently
// 	var wg sync.WaitGroup
// 	wg.Add(2)
// 	go func() {
// 		defer wg.Done()
// 		for i := 0; i < 10; i++ {
// 			ev1.Evolve(bt)
// 		}
// 	}()
// 	go func() {
// 		defer wg.Done()
// 		for i := 0; i < 10; i++ {
// 			ev2.Evolve(bt)
// 		}
// 	}()
// 	wg.Wait()
	
// 	// Combine the populations
// 	ev1.Population = append(ev1.Population, ev2.Population...)
// 	ev1.NumChromosomes = len(ev1.Population)
// 	fmt.Println("Combined population size: ", ev1.NumChromosomes)
	
// 	// Evolve the combined population
// 	for i := 0; i < 10; i++ {
// 		ev1.Evolve(bt)
// 	}

// 	ev1.SortByFitness()
// 	best_chromosome_index := ev1.NumChromosomes - 1
// 	max_acquisitions := d.ScheduleMap.GetGameSpan(week) + 1
// 	for ev1.Population[best_chromosome_index].TotalAcquisitions > max_acquisitions {
// 		fmt.Println("Total acquisitions (over):", ev1.Population[best_chromosome_index].TotalAcquisitions)
// 		best_chromosome_index--
// 	}
// 	best_chromosome := ev1.Population[best_chromosome_index]
// 	best_chromosome.AddBackNonStreamablePlayers(bt)


// 	// Get the initial fitness score
// 	base_chromosome := p.InitChromosome(bt)
// 	for _, gene := range base_chromosome.Genes {
// 		gene.InsertStreamablePlayers(bt)
// 	}
// 	base_chromosome.ScoreFitness()

// 	// // Print the best chromosome
// 	fmt.Println(bt.Score + best_chromosome.FitnessScore, "vs", bt.Score + base_chromosome.FitnessScore, "diff", best_chromosome.FitnessScore - base_chromosome.FitnessScore)
// 	best_chromosome.Print()
// 	elapsed := time.Since(start)
// 	fmt.Println("Time to run algorithm: ", elapsed)

// 	printMemUsage()
// }

// func TestEvolve(t *testing.T) {
// 	d.InitSchedule("/Users/jameskendrick/Code/cv/features/lineup-generation/v2/static/schedule.json")

// 	bt := team.InitBaseTeamMock("19", 32.0)

// 	// Create the EvolutionManager
// 	ev := p.InitPopulation(bt, 50)

// 	// Evolve the population
// 	for i := 0; i < 100; i++ {
// 		ev.Evolve(bt)

// 		// Make sure there are no duplicate players in each gene's NewPlayers
// 		for _, chromosome := range ev.Population {
// 			for _, gene := range chromosome.Genes {
// 				for i, player := range gene.NewPlayers {
// 					for j, other_player := range gene.NewPlayers {
// 						if i != j && player.Name == other_player.Name {
// 							t.Errorf("Duplicate new player")
// 						}
// 					}
// 				}
// 			}
// 		}
// 	}
// }


// func TestCrossover(t *testing.T) {
// 	d.InitSchedule("/Users/jameskendrick/Code/cv/features/lineup-generation/v2/static/schedule.json")

// 	bt := team.InitBaseTeamMock("19", 32.0)

// 	errors := 0
// 	max_aquisitions := 0
// 	for i := 0; i < 150; i++ {

// 		// Create the EvolutionManager
// 		ev := &p.EvolutionManager{Population: make([]*p.Chromosome, 2), NumChromosomes: 2}

// 		// Get the two chromosomes to crossover
// 		c1 := p.InitChromosome(bt)
// 		c2 := p.InitChromosome(bt)
// 		c1.Populate(bt, rand.New(rand.NewSource(time.Now().UnixNano())))
// 		c2.Populate(bt, rand.New(rand.NewSource(time.Now().UnixNano())))
// 		c1.ScoreFitness()
// 		c2.ScoreFitness()
// 		ev.Population[0] = c1
// 		ev.Population[1] = c2

// 		// Crossover the chromosomes
// 		child := ev.Crossover(bt, c1, c2, rand.New(rand.NewSource(time.Now().UnixNano())))

// 		// Make sure NewPlayer count corresponds with gene and total acquisitions
// 		total_acquisitions := 0
// 		for _, gene := range child.Genes {
// 			if len(gene.NewPlayers) != gene.Acquisitions {
// 				t.Errorf("Acquisition count does not match new player count")
// 			}
// 			total_acquisitions += gene.Acquisitions
// 		}
// 		if total_acquisitions != child.TotalAcquisitions {
// 			t.Errorf("Total acquisitions do not match gene acquisitions")
// 		}

// 		// Make sure NewPlayer count and DropPlayer count are correct
// 		for _, gene := range child.Genes {
// 			if len(gene.NewPlayers) != len(gene.DroppedPlayers) {
// 				t.Errorf("NewPlayer count does not match DropPlayer count")
// 			}
// 		}

// 		// Make sure the number of streamers is correct
// 		for _, gene := range child.Genes {
// 			if gene.GetNumStreamers() != len(bt.StreamablePlayers) {
// 				t.Errorf("Streamer count is incorrect")
// 			}
// 		}

// 		// Make sure each addition is in the roster
// 		for _, gene := range child.Genes {
// 			for _, player := range gene.NewPlayers {
// 				if !gene.IsPlayerInRoster(player) {
// 					t.Errorf("Player not in roster")
// 				}
// 			}
// 		}

// 		// Make sure the length of the dropped players is the same as the number of acquisitions
// 		for _, gene := range child.Genes {
// 			if len(gene.DroppedPlayers) != gene.Acquisitions {
// 				errors++
// 				t.Errorf("Dropped player count does not match acquisitions")
// 			}
// 		}

// 		if child.TotalAcquisitions > max_aquisitions {
// 			max_aquisitions = child.TotalAcquisitions
// 		}
// 	}

// 	fmt.Println("Crossover errors:", errors, "Max acquisitions:", max_aquisitions)
// }

// func TestMutate(t *testing.T) {
// 	d.InitSchedule("/Users/jameskendrick/Code/cv/features/lineup-generation/v2/static/schedule.json")

// 	bt := team.InitBaseTeamMock("19", 32.0)

// 	// errors := 0
// 	for i := 0; i < 100; i++ {

// 		// Get a chromosome to mutate
// 		c1 := p.InitChromosome(bt)
// 		c1.Populate(bt, rand.New(rand.NewSource(time.Now().UnixNano())))
// 		c1.ScoreFitness()

// 		// Mutate the chromosomes
// 		dropped_player, added_player, start, end := c1.Mutate(bt, 1.00, rand.New(rand.NewSource(time.Now().UnixNano())))
		
// 		// Make sure the number of streamers is correct
// 		for _, gene := range c1.Genes {
// 			if gene.GetNumStreamers() != len(bt.StreamablePlayers) {
// 				t.Errorf("Streamer count is incorrect")
// 			}
// 		}

// 		// Make sure NewPlayer count and DropPlayer count are correct
// 		for _, gene := range c1.Genes {
// 			if len(gene.NewPlayers) != len(gene.DroppedPlayers) {
// 				t.Errorf("NewPlayer count does not match DropPlayer count")
// 			}
// 		}

// 		// Make sure the dropped_player was correctly handled
// 		if dropped_player.Name != "" {
// 			// He should NOT be in dropped players because the mutation is a replacement

// 			// Next check that he is not anywhere in the roster within the interval
// 			for i := start; i < end; i++ {
// 				if c1.Genes[i].IsPlayerInRoster(dropped_player) {
// 					fmt.Println("Start:", start, "End:", end)
// 					fmt.Println("Day:", i)
// 					fmt.Println("Dropped player:", dropped_player)
// 					fmt.Println("Added player:", added_player)
// 					c1.Print()
// 					t.Errorf("Dropped player found in roster")
// 				}
// 			}
			
// 		}

// 		// Make sure the added_player was correctly handled
// 		if added_player.Name != "" {
// 			// First check that he is in the new players for the start day
// 			found := false
// 			for _, player := range c1.Genes[start].NewPlayers {
// 				if player.Name == added_player.Name {
// 					found = true
// 				}
// 			}
// 			if !found {
// 				t.Errorf("Added player not found in new players")
// 			}

// 			// Next check that he is in the roster for the start day
// 			if !c1.Genes[start].IsPlayerInRoster(added_player) {
// 				t.Errorf("Added player not found in roster")
// 			}
// 		}
// 	}
// }