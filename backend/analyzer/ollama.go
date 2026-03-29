package analyzer

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// OllamaRequest is the payload sent to the Ollama /api/generate endpoint.
type OllamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"` // always false
	Format string `json:"format"` // "json"
}

// OllamaResponse is the response from the Ollama /api/generate endpoint.
type OllamaResponse struct {
	Response string `json:"response"` // raw JSON string from LLM
}

// SeverityResponse is the parsed result returned to callers.
type SeverityResponse struct {
	Score  int    `json:"score"`  // 1–10 inclusive
	Advice string `json:"advice"` // non-empty guidance string
}

// ErrOllamaUnreachable is returned when the Ollama service cannot be contacted.
var ErrOllamaUnreachable = errors.New("ollama unreachable")

// ErrParseFailed is returned when the LLM response cannot be parsed.
var ErrParseFailed = errors.New("could not parse LLM response")

// ErrScoreOutOfRange is returned when the parsed score is not in [1,10].
var ErrScoreOutOfRange = errors.New("score out of range")

// currentOllamaURL is the Ollama endpoint; overridable in tests.
var currentOllamaURL = "http://localhost:11434/api/generate"

// SetOllamaURL overrides the Ollama endpoint URL. Intended for use in tests.
func SetOllamaURL(url string) { currentOllamaURL = url }

// Analyze sends text to Ollama and returns a SeverityResponse.
func Analyze(ctx context.Context, text string) (*SeverityResponse, error) {
	prompt := fmt.Sprintf(
		"You are a medical triage assistant. Assess the severity of the following symptom description and respond ONLY with a JSON object.\n\n"+
			"Scoring guide:\n"+
			"- 1-2: No real symptoms, healthy, minor discomfort\n"+
			"- 3-4: Mild symptoms, manageable at home\n"+
			"- 5-6: Moderate symptoms, consider seeing a doctor soon\n"+
			"- 7-8: Serious symptoms, see a doctor today\n"+
			"- 9-10: Life-threatening emergency (e.g. chest pain, difficulty breathing, loss of consciousness, severe bleeding, suicidal ideation, dying)\n\n"+
			"If the person says they are dying, in extreme pain, having a heart attack, can't breathe, or describes any life-threatening situation, the score MUST be 9 or 10.\n\n"+
			"Respond ONLY with this exact JSON format, no other text:\n"+
			"{\"score\": <integer 1-10>, \"advice\": \"<string>\"}\n\n"+
			"Symptom description: %s", text,
	)

	reqBody := OllamaRequest{
		Model:  "gemma3:1b",
		Prompt: prompt,
		Stream: false,
		Format: "json",
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, ErrParseFailed
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, currentOllamaURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, ErrParseFailed
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, ErrOllamaUnreachable
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, ErrParseFailed
	}

	var ollamaResp OllamaResponse
	if err := json.Unmarshal(respBytes, &ollamaResp); err != nil {
		return nil, ErrParseFailed
	}

	// Use a raw map first to handle score as either number or string from the LLM.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(ollamaResp.Response), &raw); err != nil {
		return nil, ErrParseFailed
	}

	var severity SeverityResponse

	// Parse score — LLMs sometimes return it as a quoted string.
	if scoreRaw, ok := raw["score"]; ok {
		if err := json.Unmarshal(scoreRaw, &severity.Score); err != nil {
			// Try as string "7"
			var scoreStr string
			if err2 := json.Unmarshal(scoreRaw, &scoreStr); err2 != nil {
				return nil, ErrParseFailed
			}
			if _, err2 := fmt.Sscanf(scoreStr, "%d", &severity.Score); err2 != nil {
				return nil, ErrParseFailed
			}
		}
	} else {
		return nil, ErrParseFailed
	}

	// Parse advice.
	if adviceRaw, ok := raw["advice"]; ok {
		if err := json.Unmarshal(adviceRaw, &severity.Advice); err != nil {
			return nil, ErrParseFailed
		}
	} else {
		return nil, ErrParseFailed
	}

	if severity.Score < 1 || severity.Score > 10 {
		return nil, fmt.Errorf("%w: %d", ErrScoreOutOfRange, severity.Score)
	}

	return &severity, nil
}
