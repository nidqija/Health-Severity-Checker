# Requirements Document

## Introduction

MediQuick is a full-stack medical triage application. The user submits free-form text and/or an image of their condition; the Go backend forwards the data to a local Ollama instance (model: `qwen3-vl:8b`) and returns a structured severity assessment (`score` 1–10, `advice`). For high-severity results (score ≥ 8), the frontend triggers geolocation, fetches nearby clinics via the Google Places API, and presents interactive Clinic Cards so the user can approve a destination and navigate there directly.

## Glossary

- **Frontend**: The Vue.js single-page application rendered in the browser.
- **Backend**: The Go HTTP server exposing `/analyze` and `/clinics` endpoints.
- **Ollama**: A locally running LLM inference service accessible via HTTP.
- **VLM**: Vision-Language Model — an AI model capable of processing images and text simultaneously (`qwen3-vl:8b`).
- **Severity_Response**: A JSON object containing a numeric `score` field (integer, 1–10) and a string `advice` field.
- **Analyzer**: The Go component responsible for communicating with Ollama and parsing its response.
- **Score**: An integer between 1 and 10 (inclusive) representing the severity level of the submitted input.
- **Base64**: Binary-to-text encoding used to transport image data in JSON payloads.
- **Clinic_Card**: A UI component displaying facility details (name, rating, open status) and action buttons (Approve, Reject, Navigate).

---

## Requirements

### Requirement 1: Multimodal Input and Submission

**User Story:** As a user, I want to type text and/or upload an image of my condition and submit it for analysis, so that I can receive an accurate severity assessment.

#### Acceptance Criteria

1. THE Frontend SHALL render a single multi-line text input field that is visually prominent (minimum height 120px).
2. THE Frontend SHALL render an image upload button that accepts `.jpg` and `.png` files.
3. WHEN an image is selected, THE Frontend SHALL convert the file into a Base64-encoded string for inclusion in the request payload.
4. THE Frontend SHALL render a "Check Severity" button adjacent to the inputs.
5. WHEN the "Check Severity" button is clicked and both the text input is empty and no image is selected, THE Frontend SHALL display a validation message indicating that at least one input is required.
6. WHEN the "Check Severity" button is clicked with valid input, THE Frontend SHALL send a POST request to the Backend `/analyze` endpoint with `text` and `images` fields in the request body.
7. WHILE a request to `/analyze` is in progress, THE Frontend SHALL disable the "Check Severity" button and display a loading spinner.

---

### Requirement 2: Backend Analysis Endpoint

**User Story:** As a developer, I want the Go backend to expose a `/analyze` endpoint, so that the frontend can request severity analysis of submitted text and images.

#### Acceptance Criteria

1. THE Backend SHALL expose a POST endpoint at the path `/analyze` that accepts a JSON body with a `text` string field and an `images` array of Base64-encoded strings.
2. WHEN the `/analyze` endpoint receives a valid request, THE Analyzer SHALL forward the data to the Ollama HTTP API using the `qwen3-vl:8b` model.
3. THE Analyzer SHALL instruct the VLM to respond with a JSON object containing exactly two fields: `score` (integer 1–10) and `advice` (string).
4. WHEN the Ollama API returns a valid response, THE Backend SHALL parse the Severity_Response from the VLM output and return it to the caller as a JSON HTTP response with status 200.
5. IF the Ollama API is unreachable, THEN THE Backend SHALL return an HTTP 502 response with a descriptive error message.
6. IF the VLM response cannot be parsed into a valid Severity_Response, THEN THE Backend SHALL return an HTTP 422 response with a descriptive error message.
7. THE Backend SHALL include CORS headers permitting requests from the Frontend origin.
8. THE Backend SHALL read configuration (Ollama URL, Google Maps API key, port) from environment variables defined in a `.env` file.

---

### Requirement 3: LLM Response Parsing (Round-Trip)

**User Story:** As a developer, I want the Analyzer to reliably parse the VLM's JSON output, so that malformed or unexpected responses are handled gracefully.

#### Acceptance Criteria

1. WHEN the VLM returns a JSON string containing `score` and `advice`, THE Analyzer SHALL parse it into a Severity_Response without data loss.
2. IF the parsed `score` value is less than 1 or greater than 10, THEN THE Analyzer SHALL return an error indicating an out-of-range score, which the handler maps to HTTP 422.
3. FOR ALL valid Severity_Response objects, serializing to JSON and then deserializing SHALL produce an equivalent Severity_Response (round-trip property).
4. IF the VLM response contains additional fields beyond `score` and `advice`, THE Analyzer SHALL ignore the extra fields and parse successfully.

---

### Requirement 4: Severity Result Display

**User Story:** As a user, I want to see the severity score and advice after analysis, so that I understand the assessment result.

#### Acceptance Criteria

1. WHEN the Backend returns a valid Severity_Response, THE Frontend SHALL display the `score` value and the `advice` text to the user.
2. WHEN `score < 8`, THE Frontend SHALL display the advice and suggest home monitoring.
3. WHEN the Backend returns an error response, THE Frontend SHALL display a human-readable error message to the user.
4. WHILE a request is in progress, THE Frontend SHALL NOT display a stale result from a previous request.

---

### Requirement 5: High-Severity Call to Action

**User Story:** As a user, I want to see a prominent "Book Appointment" button when my severity score is high, so that I am prompted to seek timely assistance.

#### Acceptance Criteria

1. WHEN the returned Score is greater than or equal to 8, THE Frontend SHALL display a "Book Appointment" button styled with a red background color.
2. WHEN the returned Score is less than 8, THE Frontend SHALL NOT render the "Book Appointment" button.
3. WHEN the "Book Appointment" button is displayed, THE Frontend SHALL render it at a size that is visually prominent (minimum width 200px, minimum height 48px).

---

### Requirement 6: Geolocation and Clinic Discovery

**User Story:** As a user with a high severity score, I want the app to find nearby clinics automatically, so that I can get care quickly without manual searching.

#### Acceptance Criteria

1. WHEN the returned `score` is ≥ 8, THE Frontend SHALL request the user's current geolocation via the browser's `navigator.geolocation` API.
2. IF the user denies location permissions, THE Frontend SHALL display a manual search field for "City or Postcode" as a fallback.
3. WHEN coordinates are obtained, THE Frontend SHALL send a GET request to the Backend endpoint `/clinics?lat={lat}&lng={lng}`.
4. THE Backend SHALL query the Google Places API for results within a 10km radius using keywords: `"clinic"`, `"hospital"`, `"klinik"`, `"medical center"`.
5. THE Backend SHALL return a JSON list of the top 3–5 closest clinics, each including `name`, `address`, `rating`, `open_now` status, and `place_id`.
6. IF the Google Places API is unreachable or returns an error, THE Backend SHALL return an HTTP 502 response with a descriptive error message.

---

### Requirement 7: Clinic Selection and Interaction

**User Story:** As a user, I want to review nearby clinics and choose one, so that I can navigate directly to the best care option.

#### Acceptance Criteria

1. WHEN the `/clinics` data is returned, THE Frontend SHALL render a list of Clinic Cards below the "Book Appointment" button.
2. EACH Clinic Card SHALL display the clinic name, star rating, open/closed status, address, and action buttons.
3. EACH Clinic Card SHALL feature an "Approve" (✓) button and a "Reject" (✗) button.
4. WHEN "Approve" is clicked on a card, THE Frontend SHALL store the selection in application state, hide all other cards, and display a "Visit Confirmed" summary with the chosen clinic's details.
5. WHEN "Reject" is clicked on a card, THE Frontend SHALL hide that specific card from the list without affecting others.
6. EACH Clinic Card SHALL feature a "Navigate" button that opens a new browser tab to `https://www.google.com/maps/search/?api=1&query=Google&query_place_id={place_id}`.

---

### Requirement 8: Automated Pre-Consultation PDF Generation

**User Story:** As a patient arriving at a clinic, I want a printed or digital summary of my AI assessment so that I can accurately communicate my symptoms and the AI's findings to medical staff.

#### Acceptance Criteria

1. WHEN the user clicks "Approve" on a Clinic Card, THE Frontend SHALL render a "Download Triage Summary" button in the confirmed clinic section.
2. WHEN the "Download Triage Summary" button is clicked, THE Frontend SHALL send a POST request to the Backend `/generate-pdf` endpoint.
3. THE request payload SHALL include: `symptom_text` (user's original input), `severity` (numeric score), `ai_advice` (advice string), `clinic_name` (approved clinic name), `clinic_address` (approved clinic address), and `image_data` (optional Base64 image string).
4. THE Backend SHALL use the `github.com/jung-kurt/gofpdf` library to construct a single-page PDF document in memory.
5. THE generated PDF SHALL include:
   - A header with the title "MediQuick Triage Report", a horizontal rule, and a generation timestamp.
   - A "Patient Reported Symptoms" section containing the raw symptom text.
   - If `image_data` is provided, a centered thumbnail (max-width 150mm) rendered on the page.
   - A highlighted box containing the Severity Score and AI Advice.
   - The selected clinic name and address.
6. THE Backend SHALL strip any Data URI prefix (e.g. `data:image/png;base64,`) and decode the Base64 string into a byte buffer for in-memory processing — no image or PDF data SHALL be written to disk.
7. THE Backend SHALL set the `Content-Type` response header to `application/pdf` and stream the PDF bytes directly to the Frontend.
8. THE Backend SHALL include CORS headers on the `/generate-pdf` endpoint permitting requests from the Frontend origin.
