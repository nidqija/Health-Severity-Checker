# Implementation Plan: Severity Checker

## Overview

Implement a full-stack severity checker with a Vue.js SPA frontend and a Go HTTP backend. The backend proxies text to a local Ollama instance and returns a structured severity assessment. The frontend renders the result and conditionally shows a "Book Appointment" CTA for high scores.

## Tasks

- [x] 1. Set up project structure and core types
  - Create Go module with `main.go`, `analyzer/ollama.go`, and data model types (`AnalyzeRequest`, `SeverityResponse`, `OllamaRequest`, `OllamaResponse`)
  - Scaffold Vue.js project with Vite, install `fast-check` and `vitest`
  - Define the `AppState` TypeScript interface in the frontend
  - _Requirements: 1.1, 2.1, 3.1_

- [x] 2. Implement the Go Analyzer
  - [x] 2.1 Implement `Analyze(ctx, text)` in `analyzer/ollama.go`
    - Build the Ollama prompt instructing the LLM to return `{"score": <int>, "advice": "<string>"}`
    - POST to `http://localhost:11434/api/generate` with `model: gemma3:1b`, `stream: false`, `format: "json"`
    - Extract `response` field from `OllamaResponse` and unmarshal into `SeverityResponse`
    - Return 502-class error if Ollama is unreachable; 422-class error if parse fails or score out of [1,10]
    - _Requirements: 2.2, 2.3, 2.4, 2.5, 2.6, 3.1, 3.2, 3.4_

  - [x] 2.2 Write property test for SeverityResponse serialization round-trip
    - **Property 4: SeverityResponse serialization round-trip**
    - **Validates: Requirements 3.3**

  - [x] 2.3 Write property test for out-of-range score returns 422-class error
    - **Property 5: Out-of-range score returns 422**
    - **Validates: Requirements 3.2**

  - [x] 2.4 Write property test for unparseable LLM response returns 422-class error
    - **Property 6: Unparseable LLM response returns 422**
    - **Validates: Requirements 2.6**

  - [x] 2.5 Write property test for extra fields in LLM response are ignored
    - **Property 7: Extra fields in LLM response are ignored**
    - **Validates: Requirements 3.4**

  - [x] 2.6 Write unit tests for Analyzer edge cases
    - Score boundary values: score=1 (valid), score=10 (valid), score=0 (422), score=11 (422)
    - Prompt construction includes `gemma3:1b` and the input text
    - _Requirements: 2.2, 2.3, 3.2_

- [x] 3. Implement the Go HTTP handler
  - [x] 3.1 Implement `analyzeHandler` in `main.go`
    - Decode `AnalyzeRequest`; return 400 if `text` is empty or missing
    - Call `Analyze`; map analyzer errors to 400/422/502/500 HTTP responses
    - Write `SeverityResponse` as JSON with status 200 on success
    - Add CORS middleware that sets appropriate headers on every response
    - _Requirements: 2.1, 2.4, 2.5, 2.6, 2.7_

  - [x] 3.2 Write property test for valid Ollama response produces 200 with matching body
    - **Property 3: Valid Ollama response produces 200 with parsed SeverityResponse**
    - **Validates: Requirements 2.4, 3.1**

  - [x] 3.3 Write property test for CORS headers present on all responses
    - **Property 8: CORS headers present on all responses**
    - **Validates: Requirements 2.7**

  - [x] 3.4 Write unit tests for handler error paths
    - `POST /analyze` with missing `text` → 400
    - `POST /analyze` when Ollama is unreachable → 502
    - _Requirements: 2.1, 2.5_

- [x] 4. Checkpoint — Ensure all backend tests pass
  - Ensure all tests pass, ask the user if questions arise.

- [x] 5. Implement Vue.js frontend components
  - [x] 5.1 Implement `TextInput` component
    - `<textarea>` with `modelValue` / `update:modelValue` props, `disabled` prop
    - Apply `min-height: 120px` style
    - _Requirements: 1.1_

  - [x] 5.2 Implement `ResultDisplay` component
    - Accept `score: number` and `advice: string` props
    - Render both values visibly
    - _Requirements: 4.1_

  - [x] 5.3 Implement `BookAppointmentButton` component
    - Accept `visible: boolean` prop; render only when `visible` is true
    - Style with red background, `min-width: 200px`, `min-height: 48px`
    - _Requirements: 5.1, 5.2, 5.3_

  - [x] 5.4 Write unit tests for component rendering
    - Empty input → validation message shown, no POST sent
    - Book Appointment button has correct min dimensions
    - Text area has min-height 120px
    - _Requirements: 1.1, 1.3, 5.3_

- [x] 6. Implement `SeverityChecker` view and API integration
  - [x] 6.1 Implement `SeverityChecker.vue`
    - Own reactive state: `inputText`, `loading`, `result`, `error`
    - On submit: validate non-empty, set `loading`, clear previous `result`/`error`, POST to `/analyze`, update state on response
    - Disable button and show loading indicator while `loading` is true
    - Map error HTTP statuses to human-readable messages
    - _Requirements: 1.2, 1.3, 1.4, 1.5, 4.2, 4.3_

  - [x] 6.2 Write property test for non-empty input triggers POST request
    - **Property 1: Non-empty input triggers POST request**
    - **Validates: Requirements 1.4**

  - [x] 6.3 Write property test for Book Appointment button visibility matches score threshold (score >= 8)
    - **Property 9: Book Appointment button visibility matches score threshold**
    - **Validates: Requirements 5.1, 5.2**

  - [x] 6.4 Write property test for frontend displays score and advice for any valid response
    - **Property 10: Frontend displays score and advice for any valid response**
    - **Validates: Requirements 4.1**

  - [x] 6.5 Write property test for frontend displays error message for any error response
    - **Property 11: Frontend displays error message for any error response**
    - **Validates: Requirements 4.2**

  - [x] 6.6 Write unit tests for SeverityChecker view behaviour
    - While loading → button disabled and loading indicator present
    - New request starts → previous result is cleared
    - _Requirements: 1.5, 4.3_

- [x] 7. Final checkpoint — Ensure all tests pass
  - Ensure all tests pass, ask the user if questions arise.

## Notes

- Tasks marked with `*` are optional and can be skipped for a faster MVP
- Each task references specific requirements for traceability
- Backend property tests use `pgregory.net/rapid` (minimum 100 iterations each)
- Frontend property tests use `fast-check` with `vitest` (minimum 100 iterations each)
- Checkpoints ensure incremental validation before moving to the next layer
