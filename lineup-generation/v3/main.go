package main

import (
	"fmt"
	"net/http"
	"encoding/json"

	u "v3/models"
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
		
		var request u.Request
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

func GenerateLineup(request u.Request) u.Response {
	return u.Response{}
}