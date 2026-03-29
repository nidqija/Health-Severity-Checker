# Requirements Document

## Introduction

A full-stack feature consisting of a Vue.js frontend and a Go backend. The user enters free-form text into a large input field and clicks "Check Severity". The frontend calls a Go backend endpoint `/analyze`, which forwards the text to a local Ollama instance (model: `gemma3:1b`) and requests a structured JSON response containing a severity `score` (1–10) and `advice`. If the returned score is 8 or higher, the frontend displays a prominent "Book Appointment" button styled in red.

## Glossary

- **Frontend**: The Vue.js single-page application rendered in the browser.
- **Backend**: The Go HTTP server that exposes the `/analyze` endpoint.
- **Ollama**: A locally running LLM inference service accessible via HTTP.
- **LLM**: The `gemma3:1b` language model served by Ollama.
- **Severity_Response**: A JSON object containing a numeric `score` field (integer, 1–10) and a string `advice` field.
- **Analyzer**: The Go component responsible for communicating with Ollama and parsing its response.
- **Score**: An integer between 1 and 10 (inclusive) representing the severity level of the submitted text.


---

## Requirements

### Requirement 1: Text Input and Submission

**User Story:** As a user, I want to type text into a large input field and submit it for analysis, so that I can receive a severity assessment.

#### Acceptance Criteria

1. THE Frontend SHALL render a single multi-line text input field that is visually prominent (minimum height 120px).
2. THE Frontend SHALL render a "Check Severity" button adjacent to the text input.
3. WHEN the "Check Severity" button is clicked and the text input is empty, THE Frontend SHALL display a validation message indicating that text is required.
4. WHEN the "Check Severity" button is clicked and the text input is non-empty, THE Frontend SHALL send a POST request to the Backend `/analyze` endpoint with the input text in the request body.
5. WHILE a request to `/analyze` is in progress, THE Frontend SHALL disable the "Check Severity" button and display a loading indicator.

---

### Requirement 2: Backend Analysis Endpoint

**User Story:** As a developer, I want the Go backend to expose a `/analyze` endpoint, so that the frontend can request severity analysis of submitted text.

#### Acceptance Criteria

1. THE Backend SHALL expose a POST endpoint at the path `/analyze` that accepts a JSON body with a `text` string field.
2. WHEN the `/analyze` endpoint receives a valid request, THE Analyzer SHALL forward the text to the Ollama HTTP API using the `gemma3:1b` model.
3. THE Analyzer SHALL instruct the LLM to respond with a JSON object containing exactly two fields: `score` (integer 1–10) and `advice` (string).
4. WHEN the Ollama API returns a valid response, THE Backend SHALL parse the Severity_Response from the LLM output and return it to the caller as a JSON HTTP response with status 200.
5. IF the Ollama API is unreachable, THEN THE Backend SHALL return an HTTP 502 response with a descriptive error message.
6. IF the LLM response cannot be parsed into a valid Severity_Response, THEN THE Backend SHALL return an HTTP 422 response with a descriptive error message.
7. THE Backend SHALL include CORS headers permitting requests from the Frontend origin.

---

### Requirement 3: LLM Response Parsing (Round-Trip)

**User Story:** As a developer, I want the Analyzer to reliably parse the LLM's JSON output, so that malformed or unexpected responses are handled gracefully.

#### Acceptance Criteria

1. WHEN the LLM returns a JSON string containing `score` and `advice`, THE Analyzer SHALL parse it into a Severity_Response without data loss.
2. IF the parsed `score` value is less than 1 or greater than 10, THEN THE Analyzer SHALL return an HTTP 422 response indicating an out-of-range score.
3. FOR ALL valid Severity_Response objects, serializing to JSON and then deserializing SHALL produce an equivalent Severity_Response (round-trip property).
4. IF the LLM response contains additional fields beyond `score` and `advice`, THE Analyzer SHALL ignore the extra fields and parse successfully.

---

### Requirement 4: Severity Result Display

**User Story:** As a user, I want to see the severity score and advice after analysis, so that I understand the assessment result.

#### Acceptance Criteria

1. WHEN the Backend returns a valid Severity_Response, THE Frontend SHALL display the `score` value and the `advice` text to the user.
2. WHEN the Backend returns an error response, THE Frontend SHALL display a human-readable error message to the user.
3. WHILE a request is in progress, THE Frontend SHALL NOT display a stale result from a previous request.

---

### Requirement 5: High-Severity Call to Action

**User Story:** As a user, I want to see a prominent "Book Appointment" button when my severity score is high, so that I am prompted to seek timely assistance.

#### Acceptance Criteria

1. WHEN the returned Score is greater than or equal to 8, THE Frontend SHALL display a "Book Appointment" button styled with a red background color.
2. WHEN the returned Score is less than 8, THE Frontend SHALL NOT render the "Book Appointment" button.
3. WHEN the "Book Appointment" button is displayed, THE Frontend SHALL render it at a size that is visually prominent (minimum width 200px, minimum height 48px).

### REQ-6: Geolocation and Clinic Discovery
* **Trigger**: WHEN the returned `score` is $\ge 8$, THE Frontend **SHALL** request the user’s current geolocation via the browser’s `navigator.geolocation` API.
* **Fallback**: IF the user denies location permissions, THE Frontend **SHALL** display a manual search field for "City or Postcode."
* **Data Fetching**: WHEN coordinates are obtained, THE Frontend **SHALL** send a `GET` request to the Backend endpoint `/clinics?lat={lat}&lng={lng}`.
* **Provider Integration**: THE Backend **SHALL** query the Google Places API for results within a 10km radius using keywords: `"clinic"`, `"hospital"`, `"klinik"`, `"medical center"`.
* **Response Payload**: THE Backend **SHALL** return a JSON list of the top 3–5 closest clinics, including `name`, `address`, `rating`, `open_now` status, and `place_id`.

### REQ-7: Clinic Selection and Interaction
* **Component**: WHEN the `/clinics` data is returned, THE Frontend **SHALL** render a list of **Clinic Cards** below the "Book Appointment" button.
* **Card Details**: EACH Clinic Card **SHALL** display the clinic name, a star rating, and a "Navigate" button.
* **Action Logic**: EACH Clinic Card **SHALL** feature an "Approve" (Checkmark) and "Reject" (X) button.
* **Approval**: WHEN "Approve" is clicked, THE Frontend **SHALL** store the selection in the application state and display a "Visit Confirmed" summary.
* **Navigation**: WHEN "Navigate" is clicked, THE Frontend **SHALL** open a new browser tab with the URL: `https://www.google.com/maps/search/?api=1&query=Google&query_place_id={place_id}`.
* **Rejection**: WHEN "Reject" is clicked, THE Frontend **SHALL** hide that specific card from the list.