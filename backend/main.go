package main

import (
	"encoding/json"
	"log"
	"net/http"
)

// AnalyzeRequest is the JSON body accepted by POST /analyze.
type AnalyzeRequest struct {
	Text string `json:"text"`
}

// SeverityResponse is the JSON body returned by POST /analyze.
type SeverityResponse struct {
	Score  int    `json:"score"`  // 1–10 inclusive
	Advice string `json:"advice"` // non-empty guidance string
}

func analyzeHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: implement in task 3
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"error": "not implemented"})
}

func main() {
	http.HandleFunc("/analyze", analyzeHandler)
	log.Println("Backend listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
