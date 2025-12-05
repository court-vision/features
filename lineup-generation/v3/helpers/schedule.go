package helpers

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"slices"
)

// Struct to hold a single week's schedule
type WeekSchedule struct {
	StartDate     string           	   	  	 `json:"startDate"`
	EndDate       string           	      	 `json:"endDate"`
	GameSpan  	  int                     	 `json:"gameSpan"`
	TeamSchedules map[string][]int 					 `json:"games"`
}

// InitWeekSchedule loads only the specific week's schedule data
func LoadWeekSchedule(path string, week int) (WeekSchedule, error) {
	// Load JSON schedule file
	json_schedule, err := os.Open(path)
	if err != nil {
		fmt.Println("Error opening json schedule:", err)
		return WeekSchedule{}, err
	}
	defer json_schedule.Close()

	// Read the contents of the json_schedule file
	jsonBytes, err := io.ReadAll(json_schedule)
	if err != nil {
		fmt.Println("Error reading json schedule:", err)
		return WeekSchedule{}, err
	}

	// Parse the full schedule to get the specific week
	var fullSchedule map[string]WeekSchedule
	err = json.Unmarshal(jsonBytes, &fullSchedule)
	if err != nil {
		fmt.Println("Error parsing schedule:", err)
		return WeekSchedule{}, err
	}

	// Get the specific week
	weekKey := strconv.Itoa(week)
	weekData, exists := fullSchedule[weekKey]
	if !exists {
		return WeekSchedule{}, fmt.Errorf("week %d not found in schedule", week)
	}

	fmt.Printf("Loaded schedule for week %d\n", week)
	return weekData, nil
}

func (w *WeekSchedule) GetGameSpan() int {
	return w.GameSpan
}

func (w *WeekSchedule) GetTeamSchedule(team string) []int {
	return w.TeamSchedules[team]
}

func (w *WeekSchedule) IsPlaying(day int, team string) bool {
	if game_list, ok := w.TeamSchedules[team]; ok && slices.Contains(game_list, day) {
		return true
	} else {
		return false
	}
}
	