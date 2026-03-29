# Design Document: Severity Checker

## Overview

The Severity Checker is a full-stack feature with a Vue.js single-page application (SPA) frontend and a Go HTTP backend. The user submits free-form text; the backend forwards it to a locally running Ollama instance (`gemma3:1b`) and returns a structured severity assessment. If the score is 8 or higher, the frontend surfaces a prominent "Book Appointment" call-to-action.

The system is intentionally minimal: no database, no authentication, no persistent state. All analysis is stateless and synchronous from the user's perspective.

---

## Architecture

```mermaid
sequenceDiagram
    participant User
    participant Vue (Frontend)
    participant Go (Backend /analyze)
    participant Ollama (gemma3:1b)

    User->>Vue (Frontend): Types text, clicks "Check Severity"
    Vue (Frontend)->>Go (Backend /analyze): POST /analyze { "text": "..." }
    Go (Backend /analyze)->>Ollama (gemma3:1b): POST /api/generate { model, prompt }
    Ollama (gemma3:1b)-->>Go (Backend /analyze): { "response": "{\"score\":7,\"advice\":\"...\"}" }
    Go (Backend /analyze)-->>Vue (Frontend): 200 { "score": 7, "advice": "..." }
    Vue (Frontend)-->>User: Displays score + advice (no CTA)
```

```mermaid
graph TD
    subgraph Frontend (Vue.js)
        A[TextInput Component] --> B[SeverityChecker View]
        B --> C[ResultDisplay Component]
        C --> D[BookAppointmentButton Component]
    end
    subgraph Backend (Go)
        E[HTTP Server - main.go] --> F[analyzeHandler]
        F --> G[Analyzer - ollama.go]
    end
    B -- POST /analyze --> F
    G -- POST http://localhost:11434/api/generate --> H[Ollama]
```

---

## Components and Interfaces

### Frontend Components

**SeverityChecker (View)**
- Root component that owns all state: `inputText`, `loading`, `result`, `error`
- Orchestrates child components and calls the backend

**TextInput**
- Props: `modelValue: string`, `disabled: boolean`
- Emits: `update:modelValue`
- Renders a `<textarea>` with minimum height 120px

**ResultDisplay**
- Props: `score: number`, `advice: string`
- Renders the score and advice text

**BookAppointmentButton**
- Props: `visible: boolean`
- Renders a red button (min-width 200px, min-height 48px) only when `visible` is true

### Backend

**POST /analyze**

Request body:
```json
{ "text": "string" }
```

Success response (200):
```json
{ "score": 7, "advice": "Monitor symptoms and rest." }
```

Error responses:
- `400` — missing or empty `text` field
- `422` — LLM response could not be parsed or score out of range
- `502` — Ollama unreachable

**Analyzer (internal Go package)**

```go
type SeverityResponse struct {
    Score  int    `json:"score"`
    Advice string `json:"advice"`
}

func Analyze(ctx context.Context, text string) (*SeverityResponse, error)
```

The `Analyze` function builds a prompt instructing the LLM to return only a JSON object with `score` and `advice`, calls the Ollama `/api/generate` endpoint, extracts the `response` field, and unmarshals it into `SeverityResponse`.

---

## Data Models

### AnalyzeRequest (Go)
```go
type AnalyzeRequest struct {
    Text string `json:"text"`
}
```

### SeverityResponse (Go / JSON contract)
```go
type SeverityResponse struct {
    Score  int    `json:"score"`  // 1–10 inclusive
    Advice string `json:"advice"` // non-empty guidance string
}
```

### OllamaRequest (Go, internal)
```go
type OllamaRequest struct {
    Model  string `json:"model"`
    Prompt string `json:"prompt"`
    Stream bool   `json:"stream"` // always false
    Format string `json:"format"` // "json"
}
```

### OllamaResponse (Go, internal)
```go
type OllamaResponse struct {
    Response string `json:"response"` // raw JSON string from LLM
}
```

### Frontend State (Vue reactive)
```ts
interface AppState {
  inputText: string
  loading: boolean
  result: { score: number; advice: string } | null
  error: string | null
}
```

---

## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system — essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.*

### Property 1: Non-empty input triggers POST request

*For any* non-empty string entered into the text input, clicking "Check Severity" should result in exactly one POST request being sent to `/analyze` with that string as the `text` field in the request body.

**Validates: Requirements 1.4**

---

### Property 2: Outgoing Ollama request contains model and input text

*For any* valid `/analyze` request with a given `text` value, the request forwarded to Ollama should use the `gemma3:1b` model and include the original `text` somewhere in the prompt.

**Validates: Requirements 2.2, 2.3**

---

### Property 3: Valid Ollama response produces 200 with parsed SeverityResponse

*For any* JSON string returned by Ollama that contains a valid `score` (integer 1–10) and non-empty `advice` string, the backend should return HTTP 200 with a JSON body whose `score` and `advice` fields match the parsed values exactly.

**Validates: Requirements 2.4, 3.1**

---

### Property 4: SeverityResponse serialization round-trip

*For any* valid `SeverityResponse` struct (score in [1,10], non-empty advice), marshaling to JSON and then unmarshaling should produce a struct with identical `score` and `advice` values.

**Validates: Requirements 3.3**

---

### Property 5: Out-of-range score returns 422

*For any* LLM response JSON where the `score` field is an integer less than 1 or greater than 10, the backend should return HTTP 422.

**Validates: Requirements 3.2**

---

### Property 6: Unparseable LLM response returns 422

*For any* string returned by Ollama that cannot be unmarshaled into a `SeverityResponse` with valid fields, the backend should return HTTP 422.

**Validates: Requirements 2.6**

---

### Property 7: Extra fields in LLM response are ignored

*For any* JSON object that contains valid `score` and `advice` fields plus one or more additional fields, parsing should succeed and return the correct `score` and `advice` values without error.

**Validates: Requirements 3.4**

---

### Property 8: CORS headers present on all responses

*For any* request to the `/analyze` endpoint (success or error), the response should include the appropriate CORS headers permitting the frontend origin.

**Validates: Requirements 2.7**

---

### Property 9: Book Appointment button visibility matches score threshold

*For any* `SeverityResponse`, the "Book Appointment" button should be rendered if and only if `score >= 8`. For scores in [8,10] it must be visible and red; for scores in [1,7] it must not be rendered.

**Validates: Requirements 5.1, 5.2**

---

### Property 10: Frontend displays score and advice for any valid response

*For any* valid `SeverityResponse` returned by the backend, the rendered frontend output should contain both the numeric score value and the advice string.

**Validates: Requirements 4.1**

---

### Property 11: Frontend displays error message for any error response

*For any* error response from the backend (4xx or 5xx), the frontend should display a non-empty, human-readable error message and not display a result.

**Validates: Requirements 4.2**

---

## Error Handling

### Backend

| Scenario | HTTP Status | Response Body |
|---|---|---|
| Empty or missing `text` field | 400 | `{"error": "text field is required"}` |
| Ollama unreachable / connection refused | 502 | `{"error": "analysis service unavailable"}` |
| LLM response not valid JSON | 422 | `{"error": "could not parse LLM response"}` |
| Score out of range [1,10] | 422 | `{"error": "score out of range: <value>"}` |
| Internal unexpected error | 500 | `{"error": "internal server error"}` |

The Go handler wraps all Analyzer errors and maps them to the appropriate HTTP status. Errors are logged server-side with full context; the client receives only a safe, descriptive message.

### Frontend

- On network failure (fetch throws): display "Could not reach the server. Please try again."
- On 502: display "The analysis service is currently unavailable."
- On 422: display "The analysis returned an unexpected result. Please try again."
- On any other non-200: display "An unexpected error occurred (HTTP {status})."
- While loading: clear any previous result and error to avoid stale display (Requirement 4.3).

---

## Testing Strategy

### Dual Testing Approach

Both unit tests and property-based tests are required. They are complementary:
- Unit tests cover specific examples, integration points, and edge cases.
- Property-based tests verify universal correctness across randomized inputs.

### Backend (Go)

**Property-based testing library**: `pgregory.net/rapid`

Each property test runs a minimum of 100 iterations.

Property tests:

| Test | Design Property | Tag |
|---|---|---|
| For any valid SeverityResponse, marshal→unmarshal is identity | Property 4 | `Feature: severity-checker, Property 4: SeverityResponse serialization round-trip` |
| For any score outside [1,10], Analyze returns 422-class error | Property 5 | `Feature: severity-checker, Property 5: Out-of-range score returns 422` |
| For any unparseable LLM output, Analyze returns parse error | Property 6 | `Feature: severity-checker, Property 6: Unparseable LLM response returns 422` |
| For any JSON with extra fields + valid score/advice, parse succeeds | Property 7 | `Feature: severity-checker, Property 7: Extra fields in LLM response are ignored` |
| For any valid Ollama response, handler returns 200 with matching body | Property 3 | `Feature: severity-checker, Property 3: Valid Ollama response produces 200` |
| For any request, response includes CORS headers | Property 8 | `Feature: severity-checker, Property 8: CORS headers present on all responses` |

Unit tests (specific examples and edge cases):
- `POST /analyze` with missing `text` → 400
- `POST /analyze` when Ollama is unreachable → 502
- Prompt construction includes model name `gemma3:1b` and input text
- Score boundary values: score=1 (valid), score=10 (valid), score=0 (422), score=11 (422)

### Frontend (Vue.js)

**Property-based testing library**: `fast-check` with `vitest`

Each property test runs a minimum of 100 iterations.

Property tests:

| Test | Design Property | Tag |
|---|---|---|
| For any non-empty string, submit triggers POST with that text | Property 1 | `Feature: severity-checker, Property 1: Non-empty input triggers POST request` |
| For any score >= 8, BookAppointmentButton is rendered | Property 9 | `Feature: severity-checker, Property 9: Book Appointment button visibility matches score threshold` |
| For any score < 8, BookAppointmentButton is not rendered | Property 9 | `Feature: severity-checker, Property 9: Book Appointment button visibility matches score threshold` |
| For any valid SeverityResponse, score and advice appear in output | Property 10 | `Feature: severity-checker, Property 10: Frontend displays score and advice for any valid response` |
| For any error response, a non-empty error message is shown | Property 11 | `Feature: severity-checker, Property 11: Frontend displays error message for any error response` |

Unit tests (specific examples and edge cases):
- Empty input → validation message shown, no POST sent (Requirement 1.3)
- While loading → button is disabled and loading indicator is present (Requirement 1.5)
- New request starts → previous result is cleared (Requirement 4.3)
- Book Appointment button has min-width 200px and min-height 48px (Requirement 5.3)
- Text area has min-height 120px (Requirement 1.1)
