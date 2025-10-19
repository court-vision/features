package tests

import (
	"os"
	"path/filepath"
	"testing"

	h "v3/helpers"
)

// TestInitWeekSchedule tests the new single week loading functionality
func TestLoadWeekSchedule(t *testing.T) {
	// Create a temporary test schedule file
	testScheduleData := `{
		"1": {
			"startDate": "10/21/2025",
			"endDate": "10/26/2025",
			"gameSpan": 6,
			"games": {
				"OKC": [0, 2, 4],
				"HOU": [0, 3],
				"LAL": [0, 3, 5]
			}
		},
		"2": {
			"startDate": "10/27/2025",
			"endDate": "11/02/2025",
			"gameSpan": 7,
			"games": {
				"DET": [0, 2, 5],
				"CLE": [0, 2, 4, 6]
			}
		}
	}`

	// Create temporary file
	tmpFile, err := os.CreateTemp("", "test_schedule_*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write test data to file
	if _, err := tmpFile.WriteString(testScheduleData); err != nil {
		t.Fatalf("Failed to write test data: %v", err)
	}
	tmpFile.Close()

	// Test loading week 1
	week1_schedule, err := h.LoadWeekSchedule(tmpFile.Name(), 1)
	if err != nil {
		t.Fatalf("Failed to load week 1 schedule: %v", err)
	}

	// Verify that CurrentWeekData was loaded correctly
	if week1_schedule.StartDate != "10/21/2025" {
		t.Errorf("Expected startDate '10/21/2025', got '%s'", week1_schedule.StartDate)
	}
	if week1_schedule.EndDate != "10/26/2025" {
		t.Errorf("Expected endDate '10/26/2025', got '%s'", week1_schedule.EndDate)
	}
	if week1_schedule.GameSpan != 6 {
		t.Errorf("Expected gameSpan 6, got %d", week1_schedule.GameSpan)
	}

	// Verify team schedules for week 1
	okcGames, exists := week1_schedule.TeamSchedules["OKC"]
	if !exists {
		t.Fatal("OKC team schedule not found in week 1")
	}
	expectedOKCGames := []int{0, 2, 4}
	if !compareIntSlices(okcGames, expectedOKCGames) {
		t.Errorf("Expected OKC games %v, got %v", expectedOKCGames, okcGames)
	}

	// Test loading week 2
	week2_schedule, err := h.LoadWeekSchedule(tmpFile.Name(), 2)
	if err != nil {
		t.Fatalf("Failed to load week 2 schedule: %v", err)
	}

	// Verify week 2 data
	if week2_schedule.StartDate != "10/27/2025" {
		t.Errorf("Expected startDate '10/27/2025', got '%s'", week2_schedule.StartDate)
	}
	if week2_schedule.GameSpan != 7 {
		t.Errorf("Expected gameSpan 7, got %d", week2_schedule.GameSpan)
	}

	// Verify team schedules for week 2
	detGames, exists := week2_schedule.TeamSchedules["DET"]
	if !exists {
		t.Fatal("DET team schedule not found in week 2")
	}
	expectedDETGames := []int{0, 2, 5}
	if !compareIntSlices(detGames, expectedDETGames) {
		t.Errorf("Expected DET games %v, got %v", expectedDETGames, detGames)
	}
}

// TestInitWeekScheduleWithInvalidWeek tests error handling for invalid week numbers
func TestInitWeekScheduleWithInvalidWeek(t *testing.T) {
	// Create a temporary test schedule file
	testScheduleData := `{
		"1": {
			"startDate": "10/21/2025",
			"endDate": "10/26/2025",
			"gameSpan": 6,
			"games": {
				"OKC": [0, 2, 4]
			}
		}
	}`

	tmpFile, err := os.CreateTemp("", "test_schedule_*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(testScheduleData); err != nil {
		t.Fatalf("Failed to write test data: %v", err)
	}
	tmpFile.Close()

	// Reset CurrentWeekData to ensure clean state
	week_schedule, err := h.LoadWeekSchedule(tmpFile.Name(), 99)

	// Test with non-existent week
	if week_schedule.StartDate != "" {
		t.Error("Week schedule should be empty when week doesn't exist")
	}
	if err == nil {
		t.Error("Expected error for non-existent week, got nil")
	}

	// CurrentWeekData should remain empty
	if week_schedule.StartDate != "" {
		t.Error("CurrentWeekData should be empty when week doesn't exist")
	}
}

// TestInitWeekScheduleWithInvalidFile tests error handling for invalid file path
func TestInitWeekScheduleWithInvalidFile(t *testing.T) {
	// Reset CurrentWeekData to ensure clean state
	week_schedule, err := h.LoadWeekSchedule("non_existent_file.json", 1)

	// Test with non-existent file
	if week_schedule.StartDate != "" {
		t.Error("Week schedule should be empty when file doesn't exist")
	}
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}

	// CurrentWeekData should remain empty
	if week_schedule.StartDate != "" {
		t.Error("CurrentWeekData should be empty when file doesn't exist")
	}
}

// TestInitWeekScheduleWithInvalidJSON tests error handling for invalid JSON
func TestInitWeekScheduleWithInvalidJSON(t *testing.T) {
	week_schedule, err := h.LoadWeekSchedule("./static/invalid_json_file.json", 1)

	// Test loading invalid JSON
	if week_schedule.StartDate != "" {
		t.Error("Week schedule should be empty when JSON is invalid")
	}
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}

	// CurrentWeekData should remain empty
	if week_schedule.StartDate != "" {
		t.Error("CurrentWeekData should be empty when JSON is invalid")
	}
}

// TestInitWeekScheduleWithRealData tests with actual NBA schedule data
func TestInitWeekScheduleWithRealData(t *testing.T) {
	// Path to the actual schedule file
	schedulePath := filepath.Join("..", "static", "schedule2025-2026.json")
	
	// Check if the file exists
	if _, err := os.Stat(schedulePath); os.IsNotExist(err) {
		t.Skipf("Schedule file not found at %s, skipping integration test", schedulePath)
		return
	}

	// Test loading week 1
	week1_schedule, err := h.LoadWeekSchedule(schedulePath, 1)
	if err != nil {
		t.Fatalf("Failed to load week 1 schedule: %v", err)
	}

	if week1_schedule.StartDate != "10/21/2025" {
		t.Errorf("Expected startDate '10/21/2025', got '%s'", week1_schedule.StartDate)
	}
	if week1_schedule.EndDate != "10/26/2025" {
		t.Errorf("Expected endDate '10/26/2025', got '%s'", week1_schedule.EndDate)
	}
	if week1_schedule.GameSpan != 6 {
		t.Errorf("Expected gameSpan 6, got %d", week1_schedule.GameSpan)
	}

	// Test that we have team schedules
	if len(week1_schedule.TeamSchedules) == 0 {
		t.Error("Week 1 should have team schedules")
	}

	// Test specific teams that should be in week 1
	expectedTeams := []string{"OKC", "HOU", "LAL", "GSW", "NYK"}
	for _, team := range expectedTeams {
		if _, exists := week1_schedule.TeamSchedules[team]; !exists {
			t.Errorf("Team %s not found in week 1 schedule", team)
		}
	}

	// Test loading week 20 (should be a special week with limited games)
	week20_schedule, err := h.LoadWeekSchedule(schedulePath, 20)
	if err != nil {
		t.Fatalf("Failed to load week 20 schedule: %v", err)
	}

	if week20_schedule.StartDate != "03/09/2026" {
		t.Errorf("Expected startDate '03/09/2026', got '%s'", week20_schedule.StartDate)
	}
	if week20_schedule.EndDate != "03/15/2026" {
		t.Errorf("Expected endDate '03/15/2026', got '%s'", week20_schedule.EndDate)
	}
	if week20_schedule.GameSpan != 7 {
		t.Errorf("Expected gameSpan 7, got %d", week20_schedule.GameSpan)
	}
}

// BenchmarkInitWeekSchedule benchmarks the single week loading performance
func BenchmarkInitWeekSchedule(b *testing.B) {
	schedulePath := filepath.Join("..", "static", "schedule2025-2026.json")
	
	// Check if the file exists
	if _, err := os.Stat(schedulePath); os.IsNotExist(err) {
		b.Skipf("Schedule file not found at %s, skipping benchmark", schedulePath)
		return
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Reset CurrentWeekData for each iteration
		_, err := h.LoadWeekSchedule(schedulePath, 1)
		if err != nil {
			b.Fatalf("Failed to load week 1 schedule: %v", err)
		}
	}
}

// Helper function to compare two integer slices
func compareIntSlices(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}