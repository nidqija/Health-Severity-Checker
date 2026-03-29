package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/jung-kurt/gofpdf"
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
	Text   string   `json:"text"`
	Images []string `json:"images"` // base64-encoded images (optional)
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
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || (req.Text == "" && len(req.Images) == 0) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "text or images field is required"})
		return
	}

	result, err := analyzer.Analyze(r.Context(), req.Text, req.Images)
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

// PDFRequest is the payload accepted by POST /generate-pdf.
type PDFRequest struct {
	SymptomText   string `json:"symptom_text"`
	Severity      int    `json:"severity"`
	AIAdvice      string `json:"ai_advice"`
	ClinicName    string `json:"clinic_name"`
	ClinicAddress string `json:"clinic_address"`
	ImageData     string `json:"image_data"` // optional base64 (with or without data URI prefix)
}

func generatePDFHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req PDFRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetMargins(20, 20, 20)

	// ── Header ──────────────────────────────────────────────────────────────
	pdf.SetFont("Helvetica", "B", 20)
	pdf.SetTextColor(79, 70, 229) // indigo
	pdf.CellFormat(0, 10, "MediQuick Triage Report", "", 1, "C", false, 0, "")

	pdf.SetDrawColor(79, 70, 229)
	pdf.SetLineWidth(0.5)
	pdf.Line(20, pdf.GetY()+2, 190, pdf.GetY()+2)
	pdf.Ln(6)

	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(107, 114, 128)
	pdf.CellFormat(0, 6, "Generated: "+time.Now().Format("02 Jan 2006, 15:04 MST"), "", 1, "C", false, 0, "")
	pdf.Ln(4)

	// ── Symptom text ─────────────────────────────────────────────────────────
	pdf.SetFont("Helvetica", "B", 11)
	pdf.SetTextColor(17, 24, 39)
	pdf.CellFormat(0, 7, "Patient Reported Symptoms", "", 1, "L", false, 0, "")

	pdf.SetFont("Helvetica", "", 10)
	pdf.SetTextColor(55, 65, 81)
	symptomText := req.SymptomText
	if symptomText == "" {
		symptomText = "(No text provided — image only submission)"
	}
	pdf.MultiCell(0, 6, symptomText, "1", "L", false)
	pdf.Ln(4)

	// ── Optional image ───────────────────────────────────────────────────────
	if req.ImageData != "" {
		raw := req.ImageData
		if idx := strings.Index(raw, ","); idx != -1 {
			raw = raw[idx+1:]
		}
		imgBytes, err := base64.StdEncoding.DecodeString(raw)
		if err == nil && len(imgBytes) > 0 {
			// Detect image type from first bytes
			imgType := "jpeg"
			if len(imgBytes) > 3 && imgBytes[0] == 0x89 && imgBytes[1] == 0x50 {
				imgType = "png"
			}
			opt := gofpdf.ImageOptions{ImageType: imgType, ReadDpi: true}
			pdf.RegisterImageOptionsReader("symptom_img", opt, bytes.NewReader(imgBytes))
			// Center thumbnail max 150mm wide
			imgW := 150.0
			pdf.SetX((210 - imgW) / 2)
			pdf.ImageOptions("symptom_img", (210-imgW)/2, pdf.GetY(), imgW, 0, true, opt, 0, "")
			pdf.Ln(4)
		}
	}

	// ── AI Assessment box ────────────────────────────────────────────────────
	pdf.SetFont("Helvetica", "B", 11)
	pdf.SetTextColor(17, 24, 39)
	pdf.CellFormat(0, 7, "AI Assessment", "", 1, "L", false, 0, "")

	// Severity score badge
	scoreColor := [3]int{21, 128, 61} // green
	if req.Severity >= 8 {
		scoreColor = [3]int{185, 28, 28} // red
	} else if req.Severity >= 5 {
		scoreColor = [3]int{161, 98, 7} // amber
	}
	pdf.SetFillColor(249, 250, 251)
	pdf.SetDrawColor(229, 231, 235)
	pdf.SetLineWidth(0.3)
	pdf.RoundedRect(20, pdf.GetY(), 170, 28, 3, "1234", "FD")

	pdf.SetFont("Helvetica", "B", 13)
	pdf.SetTextColor(scoreColor[0], scoreColor[1], scoreColor[2])
	pdf.SetXY(25, pdf.GetY()+4)
	pdf.CellFormat(0, 7, fmt.Sprintf("Severity Score: %d / 10", req.Severity), "", 1, "L", false, 0, "")

	pdf.SetFont("Helvetica", "", 10)
	pdf.SetTextColor(55, 65, 81)
	pdf.SetX(25)
	pdf.MultiCell(160, 5, "Recommended Action: "+req.AIAdvice, "", "L", false)
	pdf.Ln(4)

	// ── Destination clinic ───────────────────────────────────────────────────
	pdf.SetFont("Helvetica", "B", 11)
	pdf.SetTextColor(17, 24, 39)
	pdf.CellFormat(0, 7, "Selected Clinic", "", 1, "L", false, 0, "")

	pdf.SetFont("Helvetica", "B", 10)
	pdf.SetTextColor(55, 65, 81)
	pdf.CellFormat(0, 6, req.ClinicName, "", 1, "L", false, 0, "")

	pdf.SetFont("Helvetica", "", 10)
	pdf.CellFormat(0, 6, req.ClinicAddress, "", 1, "L", false, 0, "")

	// ── Footer ───────────────────────────────────────────────────────────────
	pdf.SetY(-20)
	pdf.SetFont("Helvetica", "I", 8)
	pdf.SetTextColor(156, 163, 175)
	pdf.CellFormat(0, 6, "This report is AI-generated and does not constitute medical advice. Always consult a qualified healthcare professional.", "", 1, "C", false, 0, "")

	// Stream PDF to response
	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to generate PDF"})
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `attachment; filename="MediQuick-Triage-Report.pdf"`)
	w.Header().Set("Content-Length", strconv.Itoa(buf.Len()))
	w.WriteHeader(http.StatusOK)
	w.Write(buf.Bytes())
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}
	http.HandleFunc("/analyze", corsMiddleware(analyzeHandler))
	http.HandleFunc("/clinics", corsMiddleware(clinicsHandler))
	http.HandleFunc("/generate-pdf", corsMiddleware(generatePDFHandler))
	log.Println("Backend listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
