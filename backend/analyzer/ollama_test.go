package analyzer

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"pgregory.net/rapid"
)

// TestSeverityResponseRoundTrip verifies Property 4: SeverityResponse serialization round-trip.
// Validates: Requirements 3.3
//
// Feature: severity-checker, Property 4: SeverityResponse serialization round-trip
func TestSeverityResponseRoundTrip(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		score := rapid.IntRange(1, 10).Draw(rt, "score")
		advice := rapid.StringMatching(`[A-Za-z0-9 .,!?]{1,200}`).Draw(rt, "advice")

		original := SeverityResponse{
			Score:  score,
			Advice: advice,
		}

		data, err := json.Marshal(original)
		if err != nil {
			rt.Fatalf("marshal failed: %v", err)
		}

		var decoded SeverityResponse
		if err := json.Unmarshal(data, &decoded); err != nil {
			rt.Fatalf("unmarshal failed: %v", err)
		}

		if decoded.Score != original.Score {
			rt.Fatalf("score mismatch: got %d, want %d", decoded.Score, original.Score)
		}
		if decoded.Advice != original.Advice {
			rt.Fatalf("advice mismatch: got %q, want %q", decoded.Advice, original.Advice)
		}
	})
}

// TestOutOfRangeScoreReturns422 verifies Property 5: Out-of-range score returns 422.
// Validates: Requirements 3.2
//
// Feature: severity-checker, Property 5: Out-of-range score returns 422
func TestOutOfRangeScoreReturns422(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		// Generate a score outside [1,10]: either < 1 or > 10
		score := rapid.OneOf(
			rapid.IntRange(-1000, 0),
			rapid.IntRange(11, 1000),
		).Draw(rt, "out_of_range_score")

		advice := rapid.StringMatching(`[A-Za-z0-9 .,!?]{1,100}`).Draw(rt, "advice")

		// Build a fake Ollama response JSON with the out-of-range score
		innerJSON, _ := json.Marshal(map[string]interface{}{
			"score":  score,
			"advice": advice,
		})
		outerJSON, _ := json.Marshal(map[string]interface{}{
			"response": string(innerJSON),
		})

		// Spin up a test HTTP server that returns the fake Ollama response
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write(outerJSON)
		}))
		defer srv.Close()

		// Temporarily override the Ollama URL used by Analyze
		origURL := currentOllamaURL
		currentOllamaURL = srv.URL + "/api/generate"
		defer func() { currentOllamaURL = origURL }()

		_, err := Analyze(t.Context(), "test input")
		if err == nil {
			rt.Fatalf("expected error for out-of-range score %d, got nil", score)
		}
		if !errors.Is(err, ErrScoreOutOfRange) {
			rt.Fatalf("expected ErrScoreOutOfRange for score %d, got: %v", score, err)
		}
	})
}

// TestUnparseableLLMResponseReturns422 verifies Property 6: Unparseable LLM response returns 422.
// Validates: Requirements 2.6
//
// Feature: severity-checker, Property 6: Unparseable LLM response returns 422
func TestUnparseableLLMResponseReturns422(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		// Generate an inner "response" string that cannot be unmarshaled into a valid SeverityResponse.
		// Two categories of bad input:
		//   0 – raw non-JSON string (json.Unmarshal will fail → ErrParseFailed)
		//   1 – valid JSON but missing the "score" key (score defaults to 0, out of [1,10] → ErrScoreOutOfRange)
		category := rapid.IntRange(0, 1).Draw(rt, "category")

		var innerStr string
		switch category {
		case 0:
			// Plain text that is not valid JSON
			innerStr = rapid.StringMatching(`[A-Za-z][A-Za-z0-9 .,!?]{0,79}`).Draw(rt, "plain_text")
		case 1:
			// Valid JSON object with no "score" field; Go will unmarshal score as 0 (out of range)
			key := rapid.StringMatching(`[a-z]{1,10}`).Draw(rt, "key")
			val := rapid.StringMatching(`[a-z]{1,10}`).Draw(rt, "val")
			b, _ := json.Marshal(map[string]string{key: val})
			innerStr = string(b)
		}

		outerJSON, _ := json.Marshal(map[string]interface{}{
			"response": innerStr,
		})

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write(outerJSON)
		}))
		defer srv.Close()

		origURL := currentOllamaURL
		currentOllamaURL = srv.URL + "/api/generate"
		defer func() { currentOllamaURL = origURL }()

		_, err := Analyze(t.Context(), "test input")
		if err == nil {
			rt.Fatalf("expected error for unparseable response %q, got nil", innerStr)
		}
		// Both ErrParseFailed and ErrScoreOutOfRange map to HTTP 422 in the handler.
		if !errors.Is(err, ErrParseFailed) && !errors.Is(err, ErrScoreOutOfRange) {
			rt.Fatalf("expected ErrParseFailed or ErrScoreOutOfRange for %q, got: %v", innerStr, err)
		}
	})
}

// TestExtraFieldsInLLMResponseAreIgnored verifies Property 7: Extra fields in LLM response are ignored.
// Validates: Requirements 3.4
//
// Feature: severity-checker, Property 7: Extra fields in LLM response are ignored
func TestExtraFieldsInLLMResponseAreIgnored(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		score := rapid.IntRange(1, 10).Draw(rt, "score")
		advice := rapid.StringMatching(`[A-Za-z0-9 .,!?]{1,200}`).Draw(rt, "advice")

		// Generate 1–5 extra field names (distinct from "score" and "advice")
		numExtra := rapid.IntRange(1, 5).Draw(rt, "num_extra")
		inner := map[string]interface{}{
			"score":  score,
			"advice": advice,
		}
		for i := 0; i < numExtra; i++ {
			key := rapid.StringMatching(`[a-z]{1,10}`).Draw(rt, fmt.Sprintf("extra_key_%d", i))
			// Avoid colliding with the required fields
			if key == "score" || key == "advice" {
				key = key + "_extra"
			}
			val := rapid.StringMatching(`[a-z0-9]{1,20}`).Draw(rt, fmt.Sprintf("extra_val_%d", i))
			inner[key] = val
		}

		innerJSON, _ := json.Marshal(inner)
		outerJSON, _ := json.Marshal(map[string]interface{}{
			"response": string(innerJSON),
		})

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write(outerJSON)
		}))
		defer srv.Close()

		origURL := currentOllamaURL
		currentOllamaURL = srv.URL + "/api/generate"
		defer func() { currentOllamaURL = origURL }()

		result, err := Analyze(t.Context(), "test input")
		if err != nil {
			rt.Fatalf("expected no error with extra fields, got: %v", err)
		}
		if result.Score != score {
			rt.Fatalf("score mismatch: got %d, want %d", result.Score, score)
		}
		if result.Advice != advice {
			rt.Fatalf("advice mismatch: got %q, want %q", result.Advice, advice)
		}
	})
}

// --- Unit tests for Analyzer edge cases (Task 2.6) ---
// Validates: Requirements 2.2, 2.3, 3.2

// helper: build a fake Ollama server that returns the given score and advice.
func newScoreServer(t *testing.T, score int, advice string) *httptest.Server {
	t.Helper()
	inner, _ := json.Marshal(map[string]interface{}{"score": score, "advice": advice})
	outer, _ := json.Marshal(map[string]interface{}{"response": string(inner)})
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(outer)
	}))
}

// TestScoreBoundary_1_Valid verifies that score=1 is accepted (lower boundary).
func TestScoreBoundary_1_Valid(t *testing.T) {
	srv := newScoreServer(t, 1, "minimal severity")
	defer srv.Close()

	orig := currentOllamaURL
	currentOllamaURL = srv.URL + "/api/generate"
	defer func() { currentOllamaURL = orig }()

	result, err := Analyze(t.Context(), "some text")
	if err != nil {
		t.Fatalf("expected no error for score=1, got: %v", err)
	}
	if result.Score != 1 {
		t.Fatalf("expected score=1, got %d", result.Score)
	}
}

// TestScoreBoundary_10_Valid verifies that score=10 is accepted (upper boundary).
func TestScoreBoundary_10_Valid(t *testing.T) {
	srv := newScoreServer(t, 10, "maximum severity")
	defer srv.Close()

	orig := currentOllamaURL
	currentOllamaURL = srv.URL + "/api/generate"
	defer func() { currentOllamaURL = orig }()

	result, err := Analyze(t.Context(), "some text")
	if err != nil {
		t.Fatalf("expected no error for score=10, got: %v", err)
	}
	if result.Score != 10 {
		t.Fatalf("expected score=10, got %d", result.Score)
	}
}

// TestScoreBoundary_0_Invalid verifies that score=0 returns ErrScoreOutOfRange.
func TestScoreBoundary_0_Invalid(t *testing.T) {
	srv := newScoreServer(t, 0, "below range")
	defer srv.Close()

	orig := currentOllamaURL
	currentOllamaURL = srv.URL + "/api/generate"
	defer func() { currentOllamaURL = orig }()

	_, err := Analyze(t.Context(), "some text")
	if err == nil {
		t.Fatal("expected ErrScoreOutOfRange for score=0, got nil")
	}
	if !errors.Is(err, ErrScoreOutOfRange) {
		t.Fatalf("expected ErrScoreOutOfRange, got: %v", err)
	}
}

// TestScoreBoundary_11_Invalid verifies that score=11 returns ErrScoreOutOfRange.
func TestScoreBoundary_11_Invalid(t *testing.T) {
	srv := newScoreServer(t, 11, "above range")
	defer srv.Close()

	orig := currentOllamaURL
	currentOllamaURL = srv.URL + "/api/generate"
	defer func() { currentOllamaURL = orig }()

	_, err := Analyze(t.Context(), "some text")
	if err == nil {
		t.Fatal("expected ErrScoreOutOfRange for score=11, got nil")
	}
	if !errors.Is(err, ErrScoreOutOfRange) {
		t.Fatalf("expected ErrScoreOutOfRange, got: %v", err)
	}
}

// TestPromptContainsModelAndText verifies that the outgoing request to Ollama
// includes the model "gemma3:1b" and the input text in the request body.
func TestPromptContainsModelAndText(t *testing.T) {
	inputText := "unique-input-text-for-prompt-test"

	var capturedBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody, _ = io.ReadAll(r.Body)
		// Return a valid response so Analyze doesn't error out.
		inner, _ := json.Marshal(map[string]interface{}{"score": 5, "advice": "ok"})
		outer, _ := json.Marshal(map[string]interface{}{"response": string(inner)})
		w.Header().Set("Content-Type", "application/json")
		w.Write(outer)
	}))
	defer srv.Close()

	orig := currentOllamaURL
	currentOllamaURL = srv.URL + "/api/generate"
	defer func() { currentOllamaURL = orig }()

	if _, err := Analyze(t.Context(), inputText); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var req OllamaRequest
	if err := json.Unmarshal(capturedBody, &req); err != nil {
		t.Fatalf("could not unmarshal captured request body: %v", err)
	}

	if req.Model != "gemma3:1b" {
		t.Errorf("expected model %q, got %q", "gemma3:1b", req.Model)
	}
	if !containsString(req.Prompt, inputText) {
		t.Errorf("prompt does not contain input text %q; prompt: %q", inputText, req.Prompt)
	}
}

// containsString is a simple helper to check substring presence.
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		func() bool {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
			return false
		}())
}
