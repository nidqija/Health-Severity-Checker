package analyzer

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
)

// chatMessage is a single message in the Ollama chat API format.
type chatMessage struct {
	Role    string   `json:"role"`
	Content string   `json:"content"`
	Images  []string `json:"images,omitempty"` // base64 strings, for user messages
}

// chatRequest is the payload sent to /api/chat.
type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
	Stream   bool          `json:"stream"`
	Format   string        `json:"format"`
}

// ollamaResp handles both /api/chat (message.content) and /api/generate (response) formats.
type ollamaResp struct {
	Response string `json:"response"` // /api/generate
	Message  struct {
		Content string `json:"content"`
	} `json:"message"` // /api/chat
}

// SeverityResponse is the parsed result returned to callers.
type SeverityResponse struct {
	Score  int    `json:"score"`  // 1–10 inclusive
	Advice string `json:"advice"` // non-empty guidance string
}

var ErrOllamaUnreachable = errors.New("ollama unreachable")
var ErrParseFailed = errors.New("could not parse LLM response")
var ErrScoreOutOfRange = errors.New("score out of range")

// currentOllamaURL points to /api/chat; overridable in tests.
var currentOllamaURL = "http://localhost:11434/api/chat"

func SetOllamaURL(url string) { currentOllamaURL = url }

const ollamaModel = "qwen3-vl:8b"

const systemPrompt = `You are a medical triage assistant. Assess the severity of the symptom description and respond ONLY with a JSON object.

Scoring guide:
- 1-2: No real symptoms, healthy, minor discomfort
- 3-4: Mild symptoms, manageable at home
- 5-6: Moderate symptoms, consider seeing a doctor soon
- 7-8: Serious symptoms, see a doctor today
- 9-10: Life-threatening emergency (chest pain, difficulty breathing, loss of consciousness, severe bleeding)

Respond ONLY with this exact JSON format, no other text:
{"score": <integer 1-10>, "advice": "<string>"}`

// Analyze sends text (and optional base64 images) to Ollama via /api/chat.
func Analyze(ctx context.Context, text string, images []string) (*SeverityResponse, error) {
	userContent := "Symptom description: " + text
	if text == "" && len(images) > 0 {
		userContent = "Please assess the severity of the condition shown in the image."
	}

	messages := []chatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userContent, Images: images},
	}

	reqBody := chatRequest{
		Model:    ollamaModel,
		Messages: messages,
		Stream:   false,
		Format:   "json",
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

	log.Printf("ollama full raw body: %.500s", string(respBytes))

	var parsed ollamaResp
	if err := json.Unmarshal(respBytes, &parsed); err != nil {
		log.Printf("ollama: unmarshal failed: %v", err)
		return nil, ErrParseFailed
	}

	// Prefer message.content (chat API), fall back to response (generate API).
	content := parsed.Message.Content
	if content == "" {
		content = parsed.Response
	}
	log.Printf("ollama content: %q", content)

	// Strip <think>...</think> blocks and markdown fences.
	cleaned := stripThinkingTags(content)
	log.Printf("ollama cleaned: %q", cleaned)

	jsonStr := extractFirstJSON(cleaned)
	log.Printf("ollama extracted JSON: %q", jsonStr)
	if jsonStr == "" {
		return nil, ErrParseFailed
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(jsonStr), &raw); err != nil {
		return nil, ErrParseFailed
	}

	var severity SeverityResponse

	if scoreRaw, ok := raw["score"]; ok {
		if err := json.Unmarshal(scoreRaw, &severity.Score); err != nil {
			// Try as quoted string "7"
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

var thinkTagRe = regexp.MustCompile(`(?s)<think>.*?</think>`)
var mdCodeBlockRe = regexp.MustCompile("(?s)```(?:json)?\\s*(.*?)\\s*```")

func stripThinkingTags(s string) string {
	s = thinkTagRe.ReplaceAllString(s, "")
	if m := mdCodeBlockRe.FindStringSubmatch(s); len(m) > 1 {
		s = m[1]
	}
	return strings.TrimSpace(s)
}

func extractFirstJSON(s string) string {
	start := strings.Index(s, "{")
	if start == -1 {
		return ""
	}
	depth, inString, escaped := 0, false, false
	for i := start; i < len(s); i++ {
		c := s[i]
		if escaped {
			escaped = false
			continue
		}
		if c == '\\' && inString {
			escaped = true
			continue
		}
		if c == '"' {
			inString = !inString
			continue
		}
		if inString {
			continue
		}
		switch c {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return s[start : i+1]
			}
		}
	}
	return ""
}
