package tests

import (
	"fmt"
	"testing"

	d "v2/data"
	p "v2/population"
	"v2/team"
)

func TestInitPopulation(t *testing.T) {
	loadSchedule(t)

	week := 2
	bt := team.InitBaseTeamMock(week, 3)

	ev := p.InitPopulation(bt, 10)
	if ev.NumChromosomes != 10 || len(ev.Population) != 10 {
		t.Fatalf("population has %d/%d chromosomes, want 10", len(ev.Population), ev.NumChromosomes)
	}
	for i, c := range ev.Population {
		if c == nil {
			t.Fatalf("chromosome %d is nil", i)
		}
		assertChromosomeConsistent(t, bt, c, fmt.Sprintf("chromosome %d", i))
	}
}

func TestEvolve(t *testing.T) {
	loadSchedule(t)

	week := 2
	bt := team.InitBaseTeamMock(week, 3)

	ev := p.InitPopulation(bt, 10)
	for generation := 0; generation < 5; generation++ {
		ev.Evolve(bt)

		if len(ev.Population) != 10 {
			t.Fatalf("generation %d: population shrank to %d", generation, len(ev.Population))
		}
		for i, c := range ev.Population {
			if c == nil {
				t.Fatalf("generation %d: chromosome %d is nil", generation, i)
			}
			assertChromosomeConsistent(t, bt, c, fmt.Sprintf("generation %d chromosome %d", generation, i))
		}
	}

	// SortByFitness orders ascending so the fittest chromosome is last
	ev.SortByFitness()
	for i := 1; i < len(ev.Population); i++ {
		if ev.Population[i-1].FitnessScore > ev.Population[i].FitnessScore {
			t.Errorf("population not sorted by fitness at %d: %d > %d", i, ev.Population[i-1].FitnessScore, ev.Population[i].FitnessScore)
		}
	}

	// Fitness never exceeds the unpenalised roster total
	best := ev.Population[len(ev.Population)-1]
	if best.FitnessScore < 0 {
		t.Errorf("negative fitness %d", best.FitnessScore)
	}
	if best.TotalAcquisitions <= d.ScheduleMap.GetGameSpan(week) {
		unpenalised := 0.0
		for _, gene := range best.Genes {
			for _, player := range gene.Roster {
				unpenalised += player.AvgPoints
			}
		}
		if best.FitnessScore != int(unpenalised) {
			t.Errorf("fitness %d != roster total %d with no acquisition penalty", best.FitnessScore, int(unpenalised))
		}
	}
}
