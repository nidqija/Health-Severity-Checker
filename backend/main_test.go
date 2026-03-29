package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"pgregory.net/rapid"
	"severity-checker/analyzer"
)

// TestValidOllamaResponseProduces200 verifies Property 3: Valid Ollama response produces 200 with parsed SeverityResponse.
// Validates: Requirements 2.4, 3.1
//
// Feature: severity-checker, Property 3: Valid Ollama response produces 200
func TestValidOllamaResponseProduces200(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		score := rapid.IntRange(1, 10).Draw(rt, "score")
		advice := rapid.StringMatching(`[A-Za-z0-9 .,!?]{1,200}`).Draw(rt, "advice")

		// Build the fake Ollama response: OllamaResponse wrapping a SeverityResponse JSON string.
		innerJSON, _ := json.Marshal(map[string]interface{}{
			"score":  score,
			"advice": advice,
		})
		outerJSON, _ := json.Marshal(map[string]interface{}{
			"response": string(innerJSON),
		})

		// Spin up a fake Ollama server.
		fakeSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write(outerJSON)
		}))
		defer fakeSrv.Close()

		// Override the Ollama URL used by the analyzer.
		analyzer.SetOllamaURL(fakeSrv.URL + "/api/generate")

		// Build the POST /analyze request body.
		reqBody, _ := json.Marshal(map[string]string{"text": "some input text"})
		req := httptest.NewRequest(http.MethodPost, "/analyze", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")

		// Call the handler directly via a ResponseRecorder.
		rr := httptest.NewRecorder()
		analyzeHandler(rr, req)

		// Verify HTTP 200.
		if rr.Code != http.StatusOK {
			rt.Fatalf("expected status 200, got %d (body: %s)", rr.Code, rr.Body.String())
		}

		// Verify the response body matches the expected score and advice.
		var got analyzer.SeverityResponse
		if err := json.NewDecoder(rr.Body).Decode(&got); err != nil {
			rt.Fatalf("could not decode response body: %v", err)
		}
		if got.Score != score {
			rt.Fatalf("score mismatch: got %d, want %d", got.Score, score)
		}
		if got.Advice != advice {
			rt.Fatalf("advice mismatch: got %q, want %q", got.Advice, advice)
		}
	})
}

// TestCORSHeadersPresentOnAllResponses verifies Property 8: CORS headers present on all responses.
// Validates: Requirements 2.7
//
// Feature: severity-checker, Property 8: CORS headers present on all responses
func TestCORSHeadersPresentOnAllResponses(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		// Pick a random scenario: 0 = valid request with fake Ollama, 1 = empty text (400 path)
		scenario := rapid.IntRange(0, 1).Draw(rt, "scenario")

		var req *http.Request

		if scenario == 0 {
			// Valid request: spin up a fake Ollama server returning a valid response.
			score := rapid.IntRange(1, 10).Draw(rt, "score")
			advice := rapid.StringMatching(`[A-Za-z0-9 .,!?]{1,100}`).Draw(rt, "advice")

			innerJSON, _ := json.Marshal(map[string]interface{}{
				"score":  score,
				"advice": advice,
			})
			outerJSON, _ := json.Marshal(map[string]interface{}{
				"response": string(innerJSON),
			})

			fakeSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Write(outerJSON)
			}))
			defer fakeSrv.Close()

			analyzer.SetOllamaURL(fakeSrv.URL + "/api/generate")

			reqBody, _ := json.Marshal(map[string]string{"text": "some input text"})
			req = httptest.NewRequest(http.MethodPost, "/analyze", bytes.NewReader(reqBody))
			req.Header.Set("Content-Type", "application/json")
		} else {
			// Error path: empty text → 400
			reqBody, _ := json.Marshal(map[string]string{"text": ""})
			req = httptest.NewRequest(http.MethodPost, "/analyze", bytes.NewReader(reqBody))
			req.Header.Set("Content-Type", "application/json")
		}

		rr := httptest.NewRecorder()
		corsMiddleware(analyzeHandler)(rr, req)

		// Assert CORS headers are present on every response regardless of status.
		if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "*" {
			rt.Fatalf("expected Access-Control-Allow-Origin: *, got %q", got)
		}
		if got := rr.Header().Get("Access-Control-Allow-Methods"); got == "" {
			rt.Fatalf("expected Access-Control-Allow-Methods to be set, got empty")
		}
		if got := rr.Header().Get("Access-Control-Allow-Headers"); got == "" {
			rt.Fatalf("expected Access-Control-Allow-Headers to be set, got empty")
		}
	})
}

// TestMissingTextReturns400 verifies that POST /analyze with a missing or empty text field returns HTTP 400.
// Validates: Requirements 2.1
func TestMissingTextReturns400(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"empty object", `{}`},
		{"empty text field", `{"text":""}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/analyze", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			corsMiddleware(analyzeHandler)(rr, req)

			if rr.Code != http.StatusBadRequest {
				t.Fatalf("expected status 400, got %d (body: %s)", rr.Code, rr.Body.String())
			}
		})
	}
}

// TestOllamaUnreachableReturns502 verifies that POST /analyze returns HTTP 502 when Ollama is unreachable.
// Validates: Requirements 2.5
func TestOllamaUnreachableReturns502(t *testing.T) {
	// Point the analyzer at a port that is not listening so the connection is refused.
	analyzer.SetOllamaURL("http://localhost:1/api/generate")

	reqBody, _ := json.Marshal(map[string]string{"text": "some valid input text"})
	req := httptest.NewRequest(http.MethodPost, "/analyze", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	corsMiddleware(analyzeHandler)(rr, req)

	if rr.Code != http.StatusBadGateway {
		t.Fatalf("expected status 502, got %d (body: %s)", rr.Code, rr.Body.String())
	}
}
