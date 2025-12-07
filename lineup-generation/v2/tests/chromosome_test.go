package tests

import (
	"fmt"
	"math/rand"
	d "v2/data"
	p "v2/population"
	"v2/team"
	"testing"
	"time"
)

func TestInitChromosome(t *testing.T) {
	d.InitSchedule("/Users/jameskendrick/Code/cv/features/lineup-generation/v2/static/schedule.json")

	bt := team.InitBaseTeamMock(1, 32.0)

	c := p.InitChromosome(bt)

	if len(c.Genes) != 6 {
		t.Errorf("Incorrect number of genes")
	}

}

func TestChromosomeInsertStreamablePlayers(t *testing.T) {
	d.InitSchedule("/Users/jameskendrick/Code/cv/features/lineup-generation/v2/static/schedule.json")

	bt := team.InitBaseTeamMock(1, 32.0)

	c := p.InitChromosome(bt)

	// Insert streamable players into the genes
	for _, gene := range c.Genes {
		gene.InsertStreamablePlayers(bt)
	}

	// Validate that the streamers were inserted into the right spots
	if c.Genes[0].Roster["PG"].Name != "Bradley Beal" {
		t.Errorf("Bradley Beal not in the right spot for day 0")
	}
	if !c.Genes[1].Bench.IsOnBench("Vince Williams Jr.") {
		t.Errorf("Vince Williams Jr. not on the bench for day 1")
	}
	if c.Genes[2].Roster["PG"].Name != "Bradley Beal" {
		t.Errorf("Bradley Beal not in the right spot for day 2")
	}
	if c.Genes[3].Roster["UT2"].Name != "Vince Williams Jr." {
		t.Errorf("Vince Williams Jr. not in the right spot for day 3")
	}
	if c.Genes[4].Roster["G"].Name != "Bradley Beal" {
		t.Errorf("Bradley Beal not in the right spot for day 4")
	}
	if c.Genes[4].Roster["F"].Name != "Vince Williams Jr." {
		t.Errorf("Vince Williams Jr. not in the right spot for day 4")
	}

}


func TestInsertFreeAgent(t *testing.T) {
	d.InitSchedule("/Users/jameskendrick/Code/cv/features/lineup-generation/v2/static/schedule.json")

	bt := team.InitBaseTeamMock(1, 32.0)

	c := p.InitChromosome(bt)

	// Insert streamable players into the genes
	for _, gene := range c.Genes {
		gene.InsertStreamablePlayers(bt)
	}

	// Insert "random" free agent into the chromosome
	free_agent := d.Player{Name: "Random Free Agent1", AvgPoints: 10.0, Team: "PHX", ValidPositions: []string{"C", "F", "UT1", "UT2", "UT3"}, Injured: false}
	c.InsertFreeAgent(bt, 0, free_agent)

	c.Print()

	// Validate that the free agent was inserted into the right spots and that the worst streamer was dropped
	if c.Genes[0].Roster["C"].Name != "Random Free Agent1" { t.Errorf("Random Free Agent1 not in the right spot for day 0") }
	if c.Genes[0].FreePositions["C"] { t.Errorf("Free position (C) is incorrect for day 0") }
	if c.Genes[0].Bench.IsOnBench("Vince Williams Jr.") { t.Errorf("Vince Williams Jr. is on the bench for day 0") }
	
	if !c.Genes[1].Bench.IsOnBench(free_agent) { t.Errorf("Random Free Agent1 not on the bench for day 1") }
	if c.Genes[1].Bench.IsOnBench("Vince Williams Jr.") { t.Errorf("Vince Williams Jr. is on the bench for day 0") }

	if c.Genes[2].Roster["C"].Name != "Random Free Agent1" { t.Errorf("Random Free Agent1 not in the right spot for day 2") }
	if c.Genes[2].FreePositions["C"] { t.Errorf("Free position (C) is incorrect for day 2") }
	if c.Genes[2].Bench.IsOnBench("Vince Williams Jr.") { t.Errorf("Vince Williams Jr. is on the bench for day 0") }

	if !c.Genes[3].Bench.IsOnBench(free_agent) { t.Errorf("Random Free Agent1 not on the bench for day 3") }
	if c.Genes[3].Roster["UT2"].Name == "Vince Williams Jr." { t.Errorf("Vince Williams Jr. still in roster for day 3") }

	if c.Genes[4].Roster["F"].Name != "Random Free Agent1" { t.Errorf("Random Free Agent1 not in the right spot for day 4") }
	if c.Genes[4].FreePositions["F"] { t.Errorf("Free position (F) is incorrect for day 4") }
	if c.Genes[4].Bench.IsOnBench("Vince Williams Jr.") { t.Errorf("Vince Williams Jr. is on the bench for day 4") }

	if !c.Genes[5].Bench.IsOnBench(free_agent) { t.Errorf("Random Free Agent1 not on the bench for day 5") }
	if c.Genes[5].Bench.IsOnBench("Vince Williams Jr.") { t.Errorf("Vince Williams Jr. is on the bench for day 5") }

}

func TestPopulateChromosome(t *testing.T) {
	d.InitSchedule("/Users/jameskendrick/Code/cv/features/lineup-generation/v2/static/schedule.json")

	errors := 0
	max_aquisitions := 0
	for i := 0; i < 100; i++ {
			
		bt := team.InitBaseTeamMock(2, 34.0)
		seed := time.Now().UnixNano() + int64(1)
		rng := rand.New(rand.NewSource(seed))

		c := p.InitChromosome(bt)

		c.Populate(bt, rng)

		// Make sure NewPlayer count corresponds with gene and total acquisitions
		total_acquisitions := 0
		for _, gene := range c.Genes {
			if len(gene.NewPlayers) != gene.Acquisitions {
				fmt.Println(gene.NewPlayers, gene.Acquisitions)
				t.Errorf("Acquisition count does not match new player count")
			}
			total_acquisitions += gene.Acquisitions
		}
		if total_acquisitions != c.TotalAcquisitions {
			t.Errorf("Total acquisitions do not match gene acquisitions")
		}

		// Make sure NewPlayer count and DropPlayer count are correct
		for _, gene := range c.Genes {
			if len(gene.NewPlayers) != len(gene.DroppedPlayers) {
				t.Errorf("NewPlayer count does not match DropPlayer count")
			}
		}
		

		// Make sure the number of streamers is correct
		for _, gene := range c.Genes {
			if gene.GetNumStreamers() != len(bt.StreamablePlayers) {
				t.Errorf("Streamer count is incorrect")
			}
		}

		// Make sure each addition is in the roster
		for day, gene := range c.Genes {
			for _, player := range gene.NewPlayers {
				if !gene.IsPlayerInRoster(player) {
					errors++
					fmt.Println("Day:", day)
					c.Print()
					t.Errorf("Player not in roster")
				}
			}
		}

		// Make sure the length of the dropped players is the same as the number of acquisitions
		for _, gene := range c.Genes {
			if len(gene.DroppedPlayers) != gene.Acquisitions {
				errors++
				t.Errorf("Dropped player count does not match acquisitions")
			}
		}

		// Make sure Name fields are not empty
		for _, gene := range c.Genes {
			for _, player := range gene.NewPlayers {
				if player.Name == "" {
					errors++
					t.Errorf("New player name is empty")
				}
			}
			for _, player := range gene.DroppedPlayers {
				if player.Name == "" {
					errors++
					t.Errorf("Dropped player name is empty")
				}
			}
		}

		// Make sure there are no duplicates in NewPlayers
		for _, gene := range c.Genes {
			for i, player := range gene.NewPlayers {
				for j, other_player := range gene.NewPlayers {
					if i != j && player.Name == other_player.Name {
						errors++
						t.Errorf("Duplicate new player")
					}
				}
			}
		}

		if c.TotalAcquisitions > max_aquisitions {
			max_aquisitions = c.TotalAcquisitions
		}
	}
 
	fmt.Println(errors)
	fmt.Println("Max acquisitions:", max_aquisitions)
}

func TestChromosomeSlim(t *testing.T) {
	d.InitSchedule("/Users/jameskendrick/Code/cv/stopz/v2/static/schedule.json")

	bt := team.InitBaseTeamMock(2, 32.0)
	seed := time.Now().UnixNano()
	rng := rand.New(rand.NewSource(seed))

	c := p.InitChromosome(bt)

	c.Populate(bt, rng)
	c.AddBackNonStreamablePlayers(bt)

	slim_chromosome := c.Slim()
	fmt.Println(slim_chromosome[0])
}
	