package data

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
)

// Struct for JSON schedule file that is used to get days a player is playing
type WeekSchedule struct {
	StartDate     string                     `json:"startDate"`
	EndDate       string                     `json:"endDate"`
	GameSpan      int                        `json:"gameSpan"`
	TeamSchedules map[string]map[string]bool `json:"games"`
}

// Struct to organize the season schedule
type SeasonSchedule struct {
	Schedule map[string]WeekSchedule `json:"schedule"`
}

// ScheduleMap is the process-wide season schedule. It is written once at startup by
// LoadSchedule/InitSchedule and read concurrently by the optimizer afterwards.
var ScheduleMap SeasonSchedule

// InitSchedule loads the schedule from path unless one has already been loaded, in
// which case it is a no-op. Tests use this so that every test can request the
// schedule without paying for (or racing on) repeated loads.
func InitSchedule(path string) error {
	if ScheduleMap.Schedule != nil {
		return nil
	}
	return LoadSchedule(path)
}

// LoadSchedule reads the JSON schedule at path into ScheduleMap, replacing whatever was
// loaded before. On any error ScheduleMap is left untouched and a wrapped error is
// returned so callers can fail fast (a missing or empty schedule would otherwise make
// GetGameSpan return 0 for every week and the optimizer silently return an empty plan).
func LoadSchedule(path string) error {
	jsonBytes, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("schedule: read: %w", err) // the os error already names the path
	}

	var parsed SeasonSchedule
	if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
		return fmt.Errorf("schedule: parse %s: %w", path, err)
	}
	if len(parsed.Schedule) == 0 {
		return fmt.Errorf("schedule: %s contains no weeks", path)
	}

	ScheduleMap = parsed
	return nil
}

// WeekCount returns the number of fantasy weeks in the loaded schedule (0 if none is loaded).
func (s *SeasonSchedule) WeekCount() int {
	return len(s.Schedule)
}

// HasWeek reports whether the loaded schedule contains the given fantasy week.
func (s *SeasonSchedule) HasWeek(week int) bool {
	_, ok := s.Schedule[strconv.Itoa(week)]
	return ok
}

// Function to get the schedule for a specific week
func (s *SeasonSchedule) GetWeekSchedule(week int) WeekSchedule {
	return s.Schedule[strconv.Itoa(week)]
}

// Function to get the game span for a specific week
func (s *SeasonSchedule) GetGameSpan(week int) int {
	return s.Schedule[strconv.Itoa(week)].GameSpan
}

func (s *SeasonSchedule) IsPlaying(week int, day int, team string) bool {
	weekStr := strconv.Itoa(week)
	if _, ok := s.Schedule[weekStr].TeamSchedules[team][strconv.Itoa(day)]; ok {
		return true
	} else {
		return false
	}
}

func (w *WeekSchedule) GetStartDate() string {
	return w.StartDate
}

func (w *WeekSchedule) GetEndDate() string {
	return w.EndDate
}

func (w *WeekSchedule) GetGameSpan() int {
	return w.GameSpan
}
