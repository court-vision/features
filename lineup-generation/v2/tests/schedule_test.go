package tests

import (
	"testing"
	d "v2/data"
)

func TestSchedule(t *testing.T) {
	// Init the schedule
	d.InitSchedule("/Users/jameskendrick/Code/Projects/cv/features/lineup-generation/v2/static/schedule25-26.json")

	// Check that the schedule has been initialized
	for week := 1; week <= 20; week++ {
		week_schedule := d.ScheduleMap.GetWeekSchedule(week)
		if week_schedule.StartDate == "" {
			t.Errorf("Week %v schedule not initialized", week)
		}
	}

}