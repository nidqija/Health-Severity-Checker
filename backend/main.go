package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"severity-checker/analyzer"
)

// AnalyzeRequest is the JSON body accepted by POST /analyze.
type AnalyzeRequest struct {
	Text string `json:"text"`
}

func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next(w, r)
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func analyzeHandler(w http.ResponseWriter, r *http.Request) {
	var req AnalyzeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Text == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "text field is required"})
		return
	}

	result, err := analyzer.Analyze(r.Context(), req.Text)
	if err != nil {
		log.Printf("analyzer error: %v", err)

		if errors.Is(err, analyzer.ErrOllamaUnreachable) {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "analysis service unavailable"})
			return
		}
		if errors.Is(err, analyzer.ErrParseFailed) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "could not parse LLM response"})
			return
		}
		if errors.Is(err, analyzer.ErrScoreOutOfRange) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func main() {
	http.HandleFunc("/analyze", corsMiddleware(analyzeHandler))
	log.Println("Backend listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
