package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	d "v2/data"
	l "v2/resources"
	"v2/testutil"
	u "v2/utils"
)

func loadTestSchedule(t testing.TB) string {
	t.Helper()
	path, err := testutil.SchedulePath()
	if err != nil {
		t.Fatal(err)
	}
	if err := d.InitSchedule(path); err != nil {
		t.Fatalf("load schedule %s: %v", path, err)
	}
	return path
}

// fixtureRequest builds a /generate-lineup body from the mock roster and free agents.
func fixtureRequest(t testing.TB, week, streamingSlots int) u.ReqBody {
	t.Helper()
	rosterMap := l.LoadRosterMap(testutil.RepoPath("resources", "mock_roster.json"))
	freeAgents := l.LoadFreeAgents(testutil.RepoPath("resources", "mock_freeagents.json"))
	if len(rosterMap) == 0 || len(freeAgents) == 0 {
		t.Fatalf("fixtures are empty: %d roster players, %d free agents", len(rosterMap), len(freeAgents))
	}
	roster := make([]d.Player, 0, len(rosterMap))
	for _, player := range rosterMap {
		roster = append(roster, player)
	}
	sort.Slice(roster, func(i, j int) bool { return roster[i].Name < roster[j].Name })
	return u.ReqBody{RosterData: roster, FreeAgentData: freeAgents, StreamingSlots: streamingSlots, Week: week}
}

func postJSON(t testing.TB, handler http.Handler, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func TestHealthz(t *testing.T) {
	path := loadTestSchedule(t)
	mux := newMux(path)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /healthz = %d: %s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q", ct)
	}
	var body healthzResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v: %s", err, rec.Body.String())
	}
	if body.Status != "ok" || body.ScheduleFile != path || body.Weeks != d.ScheduleMap.WeekCount() {
		t.Errorf("healthz = %+v, want status ok, file %s, weeks %d", body, path, d.ScheduleMap.WeekCount())
	}

	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/healthz", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST /healthz = %d, want 405", rec.Code)
	}
}

func TestGenerateLineupUnknownWeek(t *testing.T) {
	mux := newMux(loadTestSchedule(t))
	weeks := d.ScheduleMap.WeekCount()

	for _, week := range []int{0, 99, weeks + 1} {
		rec := postJSON(t, mux, "/generate-lineup", fixtureRequest(t, week, 3))
		if rec.Code != http.StatusBadRequest {
			t.Errorf("week %d: status %d, want 400: %s", week, rec.Code, rec.Body.String())
			continue
		}
		if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
			t.Errorf("week %d: Content-Type = %q", week, ct)
		}
		if origin := rec.Header().Get("Access-Control-Allow-Origin"); origin != "*" {
			t.Errorf("week %d: CORS header %q", week, origin)
		}
		var body unknownWeekResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("week %d: decode: %v: %s", week, err, rec.Body.String())
		}
		if body.Error != "unknown week" || body.Week != week || body.Weeks != weeks {
			t.Errorf("week %d: body %+v", week, body)
		}
	}
}

func TestGenerateLineupBadBody(t *testing.T) {
	mux := newMux(loadTestSchedule(t))

	req := httptest.NewRequest(http.MethodPost, "/generate-lineup", strings.NewReader("not json"))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status %d, want 400", rec.Code)
	}
}

func TestGenerateLineupPreflight(t *testing.T) {
	mux := newMux(loadTestSchedule(t))

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodOptions, "/generate-lineup", nil))
	if rec.Code != http.StatusOK {
		t.Errorf("OPTIONS status %d", rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "*" || rec.Header().Get("Access-Control-Allow-Methods") != "POST" {
		t.Errorf("preflight headers %v", rec.Header())
	}
}

func TestGenerateLineupWeekOne(t *testing.T) {
	if testing.Short() {
		t.Skip("runs the full genetic algorithm")
	}
	mux := newMux(loadTestSchedule(t))
	week, streamingSlots := 1, 3

	rec := postJSON(t, mux, "/generate-lineup", fixtureRequest(t, week, streamingSlots))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	var resp u.Response
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	span := d.ScheduleMap.GetGameSpan(week)
	if len(resp.Lineup) != span {
		t.Fatalf("plan has %d days, want the week's game span %d", len(resp.Lineup), span)
	}
	if resp.Week != week || resp.StreamingSlots != streamingSlots || resp.Timestamp == "" {
		t.Errorf("response echo %+v", resp)
	}

	acquisitions := 0
	for i, day := range resp.Lineup {
		if day.Day != i {
			t.Errorf("lineup[%d].Day = %d", i, day.Day)
		}
		if len(day.Roster) == 0 {
			t.Errorf("day %d has an empty roster", i)
		}
		for pos, player := range day.Roster {
			if player.Name == "" || player.Team == "" {
				t.Errorf("day %d: empty player at %s", i, pos)
			}
		}
		if len(day.Additions) != len(day.Removals) {
			t.Errorf("day %d: %d additions but %d removals", i, len(day.Additions), len(day.Removals))
		}
		acquisitions += len(day.Additions)
	}
	if acquisitions > span {
		t.Errorf("plan makes %d acquisitions, more than the %d allowed for the week", acquisitions, span)
	}
}
