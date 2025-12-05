package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	h "v3/helpers"
)

func main() {

	fmt.Println("Server started on port 8080")

	// Handle request
	http.HandleFunc("/generate-lineup", func(w http.ResponseWriter, r *http.Request) {

		if r.Method == "OPTIONS" {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "POST")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			return
		}
	
		// Set CORS headers for actual request
		w.Header().Set("Access-Control-Allow-Origin", "*")
		
		var request h.Request
		err := json.NewDecoder(r.Body).Decode(&request)
		if err != nil {
			fmt.Println(err)
			http.Error(w, "Failed to decode request body", http.StatusBadRequest)
			return
		}

		// Print the decoded request for debugging purposes
		fmt.Printf("Received request: %+v\n", request)

		// Respond with a JSON-encoded message
		json_data, err := json.Marshal(GenerateLineup(request))
		if err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, err = w.Write(json_data)
		if err != nil {
			http.Error(w, "Failed to write response", http.StatusInternalServerError)
			return
		}
	})

	// Start server
	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic(err)
	}
}

func GenerateLineup(request h.Request) h.Response {
	// Initialize the schedule for the specific week requested
	schedule, err := h.LoadWeekSchedule("./static/schedule2025-2026.json", request.Week)
	if err != nil {
		fmt.Printf("Error loading schedule for week %d: %v\n", request.Week, err)
		return h.Response{
			Lineup:     []h.Roster{},
			Improvement: 0,
			Timestamp:  "",
			Week:       request.Week,
			Threshold:  request.Threshold,
		}
	}

	setup_state := h.InitSetupState(&schedule, request.RosterData, request.FreeAgentData, request.Threshold)

	// TODO: Implement lineup generation logic using weekSchedule
	// For now, return an empty response
	return h.Response{
		Lineup:     []h.Roster{},
		Improvement: 0,
		Timestamp:  "",
		Week:       request.Week,
		Threshold:  request.Threshold,
	}
}