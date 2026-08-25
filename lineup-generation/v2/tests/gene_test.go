package tests

import (
	"testing"

	d "v2/data"
	p "v2/population"
	"v2/team"
	u "v2/utils"
)

func TestGeneInit(t *testing.T) {
	loadSchedule(t)

	gene := p.InitGene(team.InitBaseTeamMock(1, 2), 0)
	if gene.Day != 0 {
		t.Errorf("Gene day is incorrect")
	}
	if len(gene.Roster) != 0 || len(gene.FreePositions) != 0 || gene.Acquisitions != 0 || gene.Bench.GetLength() != 0 {
		t.Errorf("new gene is not empty: %+v", gene)
	}
}

func TestGeneInsertStreamablePlayers(t *testing.T) {
	loadSchedule(t)

	week := 1
	bt := team.InitBaseTeamMock(week, 2)
	if len(bt.StreamablePlayers) != 2 {
		t.Fatalf("expected 2 streamers, got %d", len(bt.StreamablePlayers))
	}

	for day := 0; day < d.ScheduleMap.GetGameSpan(week); day++ {
		gene := p.InitGene(bt, day)
		gene.InsertStreamablePlayers(bt)

		// Streamers land in open valid positions when playing and on the bench when idle
		assertStreamersSlotted(t, bt, gene)

		// Free positions are exactly the open positions the streamers did not take
		rostered := 0
		for _, player := range gene.Roster {
			if player.Name != "" {
				rostered++
			}
		}
		if got, want := u.CountOpenPositions(gene.FreePositions), len(bt.UnusedPositions[day])-rostered; got != want {
			t.Errorf("day %d: %d free positions, want %d", day, got, want)
		}
	}
	printMemUsage()
}

func TestGeneSlotPlayerDropBench(t *testing.T) {
	loadSchedule(t)

	week := 1
	bt := team.InitBaseTeamMock(week, 3)

	// An empty bench has nobody to drop
	if _, ok := p.InitGene(bt, 0).DropWorstBenchPlayer(); ok {
		t.Errorf("DropWorstBenchPlayer succeeded on an empty bench")
	}

	// Find a day on which at least one streamer sits out, so there is someone on the bench
	var gene *p.Gene
	for day := 0; day < d.ScheduleMap.GetGameSpan(week); day++ {
		candidate := p.InitGene(bt, day)
		candidate.InsertStreamablePlayers(bt)
		if candidate.Bench.GetLength() > 0 {
			gene = candidate
			break
		}
	}
	if gene == nil {
		t.Skip("every streamer plays every day of week 1 in this schedule; nothing to bench")
	}
	day := gene.Day

	// The bench is kept in ascending order, so the worst player is first
	worst := gene.Bench.Players[0]
	for _, player := range gene.Bench.Players {
		if player.AvgPoints < worst.AvgPoints {
			t.Errorf("bench is not sorted ascending: %s (%.2f) before %s (%.2f)", worst.Name, worst.AvgPoints, player.Name, player.AvgPoints)
		}
	}

	// Test the DropWorstBenchPlayer function
	dropped, ok := gene.DropWorstBenchPlayer()
	if !ok || dropped.Name != worst.Name {
		t.Fatalf("DropWorstBenchPlayer = (%s, %v), want %s", dropped.Name, ok, worst.Name)
	}
	if gene.IsPlayerInGene(dropped) {
		t.Errorf("%s is still in the gene after being dropped", dropped.Name)
	}

	// Test the SlotPlayer function: an active guard lands in his first free valid position
	streamer1 := d.Player{
		Name:           "Test Player1",
		AvgPoints:      10.0,
		Team:           teamPlayingOn(t, week, day),
		ValidPositions: []string{"PG", "SG", "G", "UT1", "UT2", "UT3"},
		Injured:        false,
	}
	want := ""
	for _, pos := range streamer1.ValidPositions {
		if gene.FreePositions[pos] {
			want = pos
			break
		}
	}
	gene.SlotPlayer(bt, streamer1)
	if want == "" {
		if !gene.Bench.IsOnBench(streamer1) {
			t.Errorf("no valid position was free; %s should be benched", streamer1.Name)
		}
	} else {
		if got := gene.GetPosOfPlayer(streamer1); got != want {
			t.Errorf("%s slotted at %q, want %s", streamer1.Name, got, want)
		}
		if gene.FreePositions[want] {
			t.Errorf("%s is still marked free after slotting", want)
		}
	}

	// An idle player always goes to the bench, whatever positions are free
	streamer2 := d.Player{
		Name:           "Test Player2",
		AvgPoints:      10.0,
		Team:           worst.Team,
		ValidPositions: []string{"PG", "SG", "SF", "PF", "C", "G", "F", "UT1", "UT2", "UT3"},
		Injured:        false,
	}
	gene.SlotPlayer(bt, streamer2)
	if !gene.Bench.IsOnBench(streamer2) || gene.GetPosOfPlayer(streamer2) != "BE" {
		t.Errorf("idle %s should be benched, got pos %q", streamer2.Name, gene.GetPosOfPlayer(streamer2))
	}
}

func TestGeneSlotPlayerDropWorst(t *testing.T) {
	loadSchedule(t)

	week := 1
	bt := team.InitBaseTeamMock(week, 2)
	c := p.InitChromosome(bt)
	for _, gene := range c.Genes {
		gene.InsertStreamablePlayers(bt)
	}

	for day, gene := range c.Genes {
		incoming := d.Player{
			Name:           "Incoming Guard",
			AvgPoints:      10.0,
			Team:           teamPlayingOn(t, week, day),
			ValidPositions: []string{"PG", "SG", "G", "UT1", "UT2", "UT3"},
			Injured:        false,
		}
		hasFree := false
		for _, pos := range incoming.ValidPositions {
			if gene.FreePositions[pos] {
				hasFree = true
				break
			}
		}

		// Test the FindStreamerToDrop function
		toDrop := c.FindStreamerToDrop(day, incoming)
		if toDrop == nil {
			// Only legitimate when nothing is free and no streamer occupies a position the newcomer can play
			if hasFree {
				t.Errorf("day %d: no streamer to drop although a valid position is free", day)
			}
			for _, streamer := range c.CurStreamers {
				if pos := gene.GetPosOfPlayer(streamer); pos != "BE" && incoming.PlaysPosition(pos) {
					t.Errorf("day %d: no streamer to drop although %s holds replaceable position %s", day, streamer.Name, pos)
				}
			}
			continue
		}
		if !u.SliceContainsPlayer(c.CurStreamers, toDrop) {
			t.Errorf("day %d: %s is not a current streamer", day, toDrop.Name)
		}
		if hasFree {
			// With a free position available, the lowest-scoring streamer is dropped
			for _, streamer := range c.CurStreamers {
				if streamer.AvgPoints < toDrop.AvgPoints {
					t.Errorf("day %d: dropped %s (%.2f) but %s (%.2f) is worse", day, toDrop.Name, toDrop.AvgPoints, streamer.Name, streamer.AvgPoints)
				}
			}
		} else {
			// Otherwise the dropped streamer must hold a position the newcomer can fill
			if pos := gene.GetPosOfPlayer(*toDrop); pos == "BE" || !incoming.PlaysPosition(pos) {
				t.Errorf("day %d: dropped %s at %q, which %s cannot replace", day, toDrop.Name, pos, incoming.Name)
			}
		}

		// Test the SlotPlayer function after removing the streamer: the newcomer is always rostered
		dropped := *toDrop
		gene.RemoveStreamer(dropped)
		gene.SlotPlayer(bt, incoming)
		if gene.IsPlayerInGene(dropped) {
			t.Errorf("day %d: %s still in gene after removal", day, dropped.Name)
		}
		if pos := gene.GetPosOfPlayer(incoming); pos == "BE" || !incoming.PlaysPosition(pos) {
			t.Errorf("day %d: %s slotted at %q", day, incoming.Name, pos)
		}
	}
}
