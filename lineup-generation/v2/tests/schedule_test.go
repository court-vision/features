package tests

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	d "v2/data"
	"v2/testutil"
)

func TestSchedule(t *testing.T) {
	loadSchedule(t)

	weeks := d.ScheduleMap.WeekCount()
	if weeks == 0 {
		t.Fatal("no weeks loaded")
	}

	// Weeks must be numbered 1..N with dates, a sane span, and every team's days inside the span
	for week := 1; week <= weeks; week++ {
		if !d.ScheduleMap.HasWeek(week) {
			t.Errorf("week %d missing: weeks should be numbered 1..%d", week, weeks)
			continue
		}
		ws := d.ScheduleMap.GetWeekSchedule(week)
		if ws.StartDate == "" || ws.EndDate == "" {
			t.Errorf("week %d has no start/end date", week)
		}
		if ws.GameSpan < 1 || ws.GameSpan > 14 {
			t.Errorf("week %d gameSpan %d out of range", week, ws.GameSpan)
		}
		if ws.GameSpan != d.ScheduleMap.GetGameSpan(week) {
			t.Errorf("week %d: GetGameSpan disagrees with the week schedule", week)
		}
		if len(ws.TeamSchedules) == 0 {
			t.Errorf("week %d has no team schedules", week)
		}
		for team, days := range ws.TeamSchedules {
			for dayStr := range days {
				day, err := strconv.Atoi(dayStr)
				if err != nil || day < 0 || day >= ws.GameSpan {
					t.Errorf("week %d team %s: day %q outside 0..%d", week, team, dayStr, ws.GameSpan-1)
					continue
				}
				if !d.ScheduleMap.IsPlaying(week, day, team) {
					t.Errorf("week %d team %s: IsPlaying(day %d) is false", week, team, day)
				}
			}
		}
	}
}

func TestWeekCountAndHasWeek(t *testing.T) {
	loadSchedule(t)

	weeks := d.ScheduleMap.WeekCount()
	if weeks != len(d.ScheduleMap.Schedule) {
		t.Errorf("WeekCount() = %d, want %d", weeks, len(d.ScheduleMap.Schedule))
	}
	if weeks < 20 {
		t.Errorf("expected a full season of at least 20 weeks, got %d", weeks)
	}

	cases := map[int]bool{-1: false, 0: false, 1: true, weeks: true, weeks + 1: false, 99: false}
	for week, want := range cases {
		if got := d.ScheduleMap.HasWeek(week); got != want {
			t.Errorf("HasWeek(%d) = %v, want %v", week, got, want)
		}
	}

	// An unknown week has a zero span, which is why callers must check HasWeek first
	if span := d.ScheduleMap.GetGameSpan(99); span != 0 {
		t.Errorf("GetGameSpan(99) = %d, want 0", span)
	}

	var empty d.SeasonSchedule
	if empty.WeekCount() != 0 || empty.HasWeek(1) {
		t.Errorf("empty schedule reports weeks: count=%d has1=%v", empty.WeekCount(), empty.HasWeek(1))
	}
}

func TestLoadScheduleErrors(t *testing.T) {
	loadSchedule(t)
	before := d.ScheduleMap.WeekCount()

	dir := t.TempDir()
	missing := filepath.Join(dir, "does-not-exist.json")
	cases := []struct{ name, path, want string }{
		{"missing file", missing, "read"},
		{"malformed json", writeTempFile(t, dir, "bad.json", "{not json"), "parse"},
		{"no weeks", writeTempFile(t, dir, "empty.json", `{"schedule": {}}`), "no weeks"},
		{"wrong shape", writeTempFile(t, dir, "shape.json", `{"weeks": [1, 2, 3]}`), "no weeks"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := d.LoadSchedule(tc.path)
			if err == nil {
				t.Fatalf("LoadSchedule(%s) returned nil error", tc.path)
			}
			if !strings.Contains(err.Error(), tc.want) || !strings.Contains(err.Error(), tc.path) {
				t.Errorf("error %q should mention %q and the path", err, tc.want)
			}
		})
	}

	if err := d.LoadSchedule(missing); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("missing file error should wrap os.ErrNotExist, got %v", err)
	}

	// A failed load must leave the previously loaded schedule intact
	if d.ScheduleMap.WeekCount() != before {
		t.Errorf("failed LoadSchedule clobbered the schedule: %d weeks, had %d", d.ScheduleMap.WeekCount(), before)
	}
}

func TestLoadScheduleReload(t *testing.T) {
	loadSchedule(t)
	path, err := testutil.SchedulePath()
	if err != nil {
		t.Fatal(err)
	}
	if err := d.LoadSchedule(path); err != nil {
		t.Fatalf("LoadSchedule(%s): %v", path, err)
	}
	if d.ScheduleMap.WeekCount() == 0 || !d.ScheduleMap.HasWeek(1) {
		t.Errorf("reload produced an unusable schedule: %d weeks", d.ScheduleMap.WeekCount())
	}
	// InitSchedule is a no-op once loaded, even for a bad path
	if err := d.InitSchedule(filepath.Join(t.TempDir(), "nope.json")); err != nil {
		t.Errorf("InitSchedule after a successful load should be a no-op, got %v", err)
	}
}

func writeTempFile(t testing.TB, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}
