# 📝 Design Documentation & Architecture

**Candidate Name:** Harsh Kumar  
**Scenario Chosen:** 3. Community Safety & Digital Wellness

---

## 1. The Problem & Solution Approach
To solve the problem of alert fatigue, I built a highly concurrent, terminal-based application (TUI) in Go. It aggregates local data from a synthetic dataset and uses AI to filter out noise, providing calm, actionable safety digests.

## 2. Tech Stack
* **Language:** Go (Golang)
* **UI:** `bubbletea` & `lipgloss` (for a non-blocking, highly responsive Terminal UI).
* **Database:** `modernc.org/sqlite` (A CGO-free SQLite implementation for seamless cross-platform local persistence).
* **AI:** Groq API leveraging `llama-3.1-8b-instant` for rapid, JSON-enforced categorization.

## 3. Architectural Decisions & Reasons
1. **Concurrent Data Ingestion:** To ensure the UI never blocks or freezes, ingestion from the synthetic `feed.json` occurs in a background Goroutine. It deduplicates against the SQLite database to save API quotas, processes the alert, and passes a `RefreshMsg` to the Bubbletea UI to dynamically update the screen.
2. **Defensive AI Pipeline:** Knowing that free-tier LLMs occasionally hallucinate formatting or fail dependent logic, I implemented a strict **Sanitization Block** in Go. This layer parses, corrects, and enforces schema compliance on the AI's JSON output before it ever reaches the database.
3. **Graceful Degradation (Fallback):** As required, I implemented a manual rule-based fallback. If the Groq API rate-limits (`HTTP 429`) or crashes (`HTTP 500`), the pipeline instantly routes to a local substring matcher to categorize the threat. This guarantees zero downtime and immediate user safety.
4. **API Mocking for Tests:** Rather than relying on a live, rate-limited API for unit testing, I utilized Go's `httptest` package to spin up a local mock server. This ensures the test suite (covering happy paths, edge cases, and API failures) is fast, free, and completely deterministic.

## 4. Future Enhancements
* **Export to Safe Circle:** A feature allowing users to press a keybind to copy a text-only actionable checklist to their clipboard for quick SMS sharing.
* **Persona-Based Filtering:** UI toggles tailored to target audiences (e.g., an "Elderly Mode" prioritizing physical threats, and a "Remote Worker Mode" prioritizing data breaches).
* **Historical Trend Analysis:** A TUI dashboard pane leveraging the SQLite database to display weekly community safety trends.