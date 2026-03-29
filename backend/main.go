package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"severity-checker/analyzer"
)

// Clinic represents a single clinic result returned to the frontend.
type Clinic struct {
	Name     string  `json:"name"`
	Address  string  `json:"address"`
	Rating   float64 `json:"rating"`
	OpenNow  bool    `json:"open_now"`
	PlaceID  string  `json:"place_id"`
}

// placesAPIURL is the Google Places Nearby Search endpoint; overridable in tests.
var placesAPIURL = "https://maps.googleapis.com/maps/api/place/nearbysearch/json"

// AnalyzeRequest is the JSON body accepted by POST /analyze.
type AnalyzeRequest struct {
	Text string `json:"text"`
}

func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
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

// klangValleyLat/Lng is the fallback centre (Kuala Lumpur city centre).
const klangValleyLat = "3.1390"
const klangValleyLng = "101.6869"

// searchClinics queries the Places API for a given location string and radius.
func searchClinics(location, radius, apiKey string) ([]Clinic, error) {
	keywords := []string{"clinic", "hospital", "klinik", "medical center"}
	seen := map[string]bool{}
	var clinics []Clinic

	for _, kw := range keywords {
		if len(clinics) >= 5 {
			break
		}
		params := url.Values{}
		params.Set("location", location)
		params.Set("radius", radius)
		params.Set("keyword", kw)
		params.Set("key", apiKey)

		resp, err := http.Get(placesAPIURL + "?" + params.Encode())
		if err != nil {
			log.Printf("places API error for keyword %q: %v", kw, err)
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		var result struct {
			Results []struct {
				Name         string  `json:"name"`
				Vicinity     string  `json:"vicinity"`
				Rating       float64 `json:"rating"`
				PlaceID      string  `json:"place_id"`
				OpeningHours *struct {
					OpenNow bool `json:"open_now"`
				} `json:"opening_hours"`
			} `json:"results"`
		}
		if err := json.Unmarshal(body, &result); err != nil {
			continue
		}
		for _, r := range result.Results {
			if seen[r.PlaceID] {
				continue
			}
			seen[r.PlaceID] = true
			openNow := false
			if r.OpeningHours != nil {
				openNow = r.OpeningHours.OpenNow
			}
			clinics = append(clinics, Clinic{
				Name:    r.Name,
				Address: r.Vicinity,
				Rating:  r.Rating,
				OpenNow: openNow,
				PlaceID: r.PlaceID,
			})
			if len(clinics) >= 5 {
				break
			}
		}
	}
	return clinics, nil
}

func clinicsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	latStr := r.URL.Query().Get("lat")
	lngStr := r.URL.Query().Get("lng")
	if latStr == "" || lngStr == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "lat and lng query parameters are required"})
		return
	}
	if _, err := strconv.ParseFloat(latStr, 64); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid lat"})
		return
	}
	if _, err := strconv.ParseFloat(lngStr, 64); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid lng"})
		return
	}

	apiKey := os.Getenv("GOOGLE_PLACES_API_KEY")
	if apiKey == "" {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "places API key not configured"})
		return
	}

	userLocation := fmt.Sprintf("%s,%s", latStr, lngStr)

	// Try user location at 10km, then 25km, then fall back to Klang Valley.
	attempts := []struct{ location, radius string }{
		{userLocation, "10000"},
		{userLocation, "25000"},
		{fmt.Sprintf("%s,%s", klangValleyLat, klangValleyLng), "20000"},
	}

	var clinics []Clinic
	for _, attempt := range attempts {
		results, err := searchClinics(attempt.location, attempt.radius, apiKey)
		if err != nil {
			continue
		}
		if len(results) > 0 {
			clinics = results
			break
		}
	}

	if clinics == nil {
		clinics = []Clinic{}
	}
	writeJSON(w, http.StatusOK, clinics)
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}
	http.HandleFunc("/analyze", corsMiddleware(analyzeHandler))
	http.HandleFunc("/clinics", corsMiddleware(clinicsHandler))
	log.Println("Backend listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
