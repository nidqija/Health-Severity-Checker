package analyzer

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// chatReply builds a fake /api/chat response with the given content string.
func chatReply(content string) []byte {
	b, _ := json.Marshal(map[string]interface{}{
		"message": map[string]string{"role": "assistant", "content": content},
	})
	return b
}

// newScoreServer builds a fake Ollama /api/chat server returning the given score and advice.
func newScoreServer(t *testing.T, score int, advice string) *httptest.Server {
	t.Helper()
	inner, _ := json.Marshal(map[string]interface{}{"score": score, "advice": advice})
	reply := chatReply(string(inner))
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(reply)
	}))
}

// withServer temporarily overrides currentOllamaURL to the test server's URL.
func withServer(srv *httptest.Server) func() {
	orig := currentOllamaURL
	currentOllamaURL = srv.URL + "/api/chat"
	return func() { currentOllamaURL = orig }
}

// TestSeverityResponseRoundTrip verifies Property 4: SeverityResponse serialization round-trip.
// Feature: severity-checker, Property 4: SeverityResponse serialization round-trip
func TestSeverityResponseRoundTrip(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		score := rapid.IntRange(1, 10).Draw(rt, "score")
		advice := rapid.StringMatching(`[A-Za-z0-9 .,!?]{1,200}`).Draw(rt, "advice")

		original := SeverityResponse{Score: score, Advice: advice}
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
// Feature: severity-checker, Property 5: Out-of-range score returns 422
func TestOutOfRangeScoreReturns422(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		score := rapid.OneOf(
			rapid.IntRange(-1000, 0),
			rapid.IntRange(11, 1000),
		).Draw(rt, "out_of_range_score")
		advice := rapid.StringMatching(`[A-Za-z0-9 .,!?]{1,100}`).Draw(rt, "advice")

		inner, _ := json.Marshal(map[string]interface{}{"score": score, "advice": advice})
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write(chatReply(string(inner)))
		}))
		defer srv.Close()
		defer withServer(srv)()

		_, err := Analyze(t.Context(), "test input", nil)
		if err == nil {
			rt.Fatalf("expected error for out-of-range score %d, got nil", score)
		}
		if !errors.Is(err, ErrScoreOutOfRange) {
			rt.Fatalf("expected ErrScoreOutOfRange for score %d, got: %v", score, err)
		}
	})
}

// TestUnparseableLLMResponseReturns422 verifies Property 6: Unparseable LLM response returns 422.
// Feature: severity-checker, Property 6: Unparseable LLM response returns 422
func TestUnparseableLLMResponseReturns422(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		category := rapid.IntRange(0, 1).Draw(rt, "category")
		var innerStr string
		switch category {
		case 0:
			innerStr = rapid.StringMatching(`[A-Za-z][A-Za-z0-9 .,!?]{0,79}`).Draw(rt, "plain_text")
		case 1:
			key := rapid.StringMatching(`[a-z]{1,10}`).Draw(rt, "key")
			val := rapid.StringMatching(`[a-z]{1,10}`).Draw(rt, "val")
			b, _ := json.Marshal(map[string]string{key: val})
			innerStr = string(b)
		}

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write(chatReply(innerStr))
		}))
		defer srv.Close()
		defer withServer(srv)()

		_, err := Analyze(t.Context(), "test input", nil)
		if err == nil {
			rt.Fatalf("expected error for unparseable response %q, got nil", innerStr)
		}
		if !errors.Is(err, ErrParseFailed) && !errors.Is(err, ErrScoreOutOfRange) {
			rt.Fatalf("expected ErrParseFailed or ErrScoreOutOfRange for %q, got: %v", innerStr, err)
		}
	})
}

// TestExtraFieldsInLLMResponseAreIgnored verifies Property 7: Extra fields in LLM response are ignored.
// Feature: severity-checker, Property 7: Extra fields in LLM response are ignored
func TestExtraFieldsInLLMResponseAreIgnored(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		score := rapid.IntRange(1, 10).Draw(rt, "score")
		advice := rapid.StringMatching(`[A-Za-z0-9 .,!?]{1,200}`).Draw(rt, "advice")
		numExtra := rapid.IntRange(1, 5).Draw(rt, "num_extra")
		inner := map[string]interface{}{"score": score, "advice": advice}
		for i := 0; i < numExtra; i++ {
			key := rapid.StringMatching(`[a-z]{1,10}`).Draw(rt, fmt.Sprintf("extra_key_%d", i))
			if key == "score" || key == "advice" {
				key += "_extra"
			}
			val := rapid.StringMatching(`[a-z0-9]{1,20}`).Draw(rt, fmt.Sprintf("extra_val_%d", i))
			inner[key] = val
		}
		innerJSON, _ := json.Marshal(inner)

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write(chatReply(string(innerJSON)))
		}))
		defer srv.Close()
		defer withServer(srv)()

		result, err := Analyze(t.Context(), "test input", nil)
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

// --- Unit tests ---

func TestScoreBoundary_1_Valid(t *testing.T) {
	srv := newScoreServer(t, 1, "minimal severity")
	defer srv.Close()
	defer withServer(srv)()
	result, err := Analyze(t.Context(), "some text", nil)
	if err != nil {
		t.Fatalf("expected no error for score=1, got: %v", err)
	}
	if result.Score != 1 {
		t.Fatalf("expected score=1, got %d", result.Score)
	}
}

func TestScoreBoundary_10_Valid(t *testing.T) {
	srv := newScoreServer(t, 10, "maximum severity")
	defer srv.Close()
	defer withServer(srv)()
	result, err := Analyze(t.Context(), "some text", nil)
	if err != nil {
		t.Fatalf("expected no error for score=10, got: %v", err)
	}
	if result.Score != 10 {
		t.Fatalf("expected score=10, got %d", result.Score)
	}
}

func TestScoreBoundary_0_Invalid(t *testing.T) {
	srv := newScoreServer(t, 0, "below range")
	defer srv.Close()
	defer withServer(srv)()
	_, err := Analyze(t.Context(), "some text", nil)
	if err == nil {
		t.Fatal("expected ErrScoreOutOfRange for score=0, got nil")
	}
	if !errors.Is(err, ErrScoreOutOfRange) {
		t.Fatalf("expected ErrScoreOutOfRange, got: %v", err)
	}
}

func TestScoreBoundary_11_Invalid(t *testing.T) {
	srv := newScoreServer(t, 11, "above range")
	defer srv.Close()
	defer withServer(srv)()
	_, err := Analyze(t.Context(), "some text", nil)
	if err == nil {
		t.Fatal("expected ErrScoreOutOfRange for score=11, got nil")
	}
	if !errors.Is(err, ErrScoreOutOfRange) {
		t.Fatalf("expected ErrScoreOutOfRange, got: %v", err)
	}
}

// TestPromptContainsModelAndText verifies the outgoing chat request contains the model and input text.
func TestPromptContainsModelAndText(t *testing.T) {
	inputText := "unique-input-text-for-prompt-test"
	var capturedBody []byte

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody, _ = io.ReadAll(r.Body)
		inner, _ := json.Marshal(map[string]interface{}{"score": 5, "advice": "ok"})
		w.Header().Set("Content-Type", "application/json")
		w.Write(chatReply(string(inner)))
	}))
	defer srv.Close()
	defer withServer(srv)()

	if _, err := Analyze(t.Context(), inputText, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var req chatRequest
	if err := json.Unmarshal(capturedBody, &req); err != nil {
		t.Fatalf("could not unmarshal captured request body: %v", err)
	}
	if req.Model != "qwen3-vl:8b" {
		t.Errorf("expected model %q, got %q", "qwen3-vl:8b", req.Model)
	}
	// Input text should appear in one of the user messages
	found := false
	for _, msg := range req.Messages {
		if strings.Contains(msg.Content, inputText) {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("input text %q not found in any message", inputText)
	}
}
