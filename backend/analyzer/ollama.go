package analyzer

import (
	"context"
	"errors"
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

// Analyze sends text to Ollama and returns a SeverityResponse.
// TODO: implement in task 2.
func Analyze(ctx context.Context, text string) (*SeverityResponse, error) {
	return nil, errors.New("not implemented")
}
