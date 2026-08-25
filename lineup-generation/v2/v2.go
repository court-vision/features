package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	d "v2/data"
	p "v2/population"
	t "v2/team"
	u "v2/utils"
)

// defaultScheduleFile is used when SCHEDULE_FILE is unset. The file must be regenerated
// (and this default bumped) every season; see README.md.
const defaultScheduleFile = "./static/schedule26-27.json"

const listenAddr = ":8080"

func main() {
	scheduleFile := os.Getenv("SCHEDULE_FILE")
	if scheduleFile == "" {
		scheduleFile = defaultScheduleFile
	}

	// Load the schedule exactly once and fail fast: a missing or malformed file would
	// otherwise make every week look like a zero-day span and every plan come back empty.
	if err := d.LoadSchedule(scheduleFile); err != nil {
		log.Fatalf("failed to load schedule: %v", err)
	}
	log.Printf("Loaded schedule %s (%d weeks)", scheduleFile, d.ScheduleMap.WeekCount())

	log.Printf("Server started on %s", listenAddr)
	if err := http.ListenAndServe(listenAddr, newMux(scheduleFile)); err != nil {
		log.Fatal(err)
	}
}

// newMux wires the HTTP routes. scheduleFile is only echoed back by /healthz.
func newMux(scheduleFile string) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", healthzHandler(scheduleFile))
	mux.HandleFunc("/generate-lineup", generateLineupHandler)
	return mux
}

type healthzResponse struct {
	Status       string `json:"status"`
	ScheduleFile string `json:"schedule_file"`
	Weeks        int    `json:"weeks"`
}

// healthzHandler reports liveness plus which schedule is loaded, so a deploy with a
// missing or stale schedule file is visible from the health check.
func healthzHandler(scheduleFile string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, http.StatusOK, healthzResponse{
			Status:       "ok",
			ScheduleFile: scheduleFile,
			Weeks:        d.ScheduleMap.WeekCount(),
		})
	}
}

type unknownWeekResponse struct {
	Error string `json:"error"`
	Week  int    `json:"week"`
	Weeks int    `json:"weeks"`
}

func generateLineupHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method == "OPTIONS" {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		return
	}

	// Set CORS headers for actual request
	w.Header().Set("Access-Control-Allow-Origin", "*")

	var request u.ReqBody
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Failed to decode request body", http.StatusBadRequest)
		return
	}

	// Print the decoded request for debugging purposes
	fmt.Printf("Received request: Week %d, StreamingSlots %d\n", request.Week, request.StreamingSlots)

	// Reject weeks the loaded schedule doesn't cover; GetGameSpan would return 0 for them
	// and the optimizer would silently produce an empty plan.
	if !d.ScheduleMap.HasWeek(request.Week) {
		writeJSON(w, http.StatusBadRequest, unknownWeekResponse{
			Error: "unknown week",
			Week:  request.Week,
			Weeks: d.ScheduleMap.WeekCount(),
		})
		return
	}

	// Respond with a JSON-encoded message
	writeJSON(w, http.StatusOK, OptimizeStreaming(request))
}

// writeJSON serialises v and writes it with the given status; on encoding failure it
// responds 500 without having written any of the body.
func writeJSON(w http.ResponseWriter, status int, v any) {
	body, err := json.Marshal(v)
	if err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, err := w.Write(body); err != nil {
		fmt.Println("Failed to write response:", err)
	}
}

func OptimizeStreaming(req u.ReqBody) u.Response {
	start := time.Now()

	// Extract request data
	week := req.Week
	streamingSlots := req.StreamingSlots

	// Initialize the BaseTeam object with player data from the request
	bt := t.InitBaseTeam(req.RosterData, req.FreeAgentData, week, streamingSlots)

	// Create new populations
	ev1 := p.InitPopulation(bt, 20)
	ev2 := p.InitPopulation(bt, 20)

	// Evolve the populations concurrently
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for range 10 {
			ev1.Evolve(bt)
		}
	}()
	go func() {
		defer wg.Done()
		for range 10 {
			ev2.Evolve(bt)
		}
	}()
	wg.Wait()

	// Combine the populations
	ev1.Population = append(ev1.Population, ev2.Population...)
	ev1.NumChromosomes = len(ev1.Population)
	fmt.Println("Combined population size: ", ev1.NumChromosomes)

	// Evolve the combined population
	for range 10 {
		ev1.Evolve(bt)
	}

	ev1.SortByFitness()
	best_chromosome_index := ev1.NumChromosomes - 1
	for ev1.Population[best_chromosome_index].TotalAcquisitions > d.ScheduleMap.GetGameSpan(week) {
		best_chromosome_index--
	}
	best_chromosome := ev1.Population[best_chromosome_index]
	best_chromosome.AddBackNonStreamablePlayers(bt)

	// Get the initial fitness score
	base_chromosome := p.InitChromosome(bt)
	for _, gene := range base_chromosome.Genes {
		gene.InsertStreamablePlayers(bt)
	}
	base_chromosome.ScoreFitness()

	// Print the best chromosome
	fmt.Println(bt.Score+best_chromosome.FitnessScore, "vs", bt.Score+base_chromosome.FitnessScore, "diff", best_chromosome.FitnessScore-base_chromosome.FitnessScore)
	best_chromosome.Print()
	elapsed := time.Since(start)
	fmt.Println("Time to run algorithm: ", elapsed)

	current_time := time.Now()
	layout := "1/2/2006 3:04PM"

	return u.Response{Lineup: best_chromosome.Slim(), Improvement: best_chromosome.FitnessScore - base_chromosome.FitnessScore, Timestamp: current_time.Format(layout), Week: week, StreamingSlots: streamingSlots}

}
