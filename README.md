# 🛡️ Community Guardian CLI

**Video Demo Link:** `[INSERT YOUR YOUTUBE/VIMEO LINK HERE]`

## Candidate Information
* **Candidate Name:** Harsh Kumar
* **Scenario Chosen:** 3. Community Safety & Digital Wellness
* **Estimated Time Spent:** ~5 Hours

---

## Quick Start

* **Prerequisites:**
    * Go 1.21 or higher installed.
    * A Groq API Key (for the Llama 3.1 LLM).
    * Create a `.env` file in the root directory using the provided `.env.example`.

* **Run Commands:**
  ```bash
  # 1. Clone the repository
  git clone https://github.com/Harschmann/community-guardian.git
  cd community-guardian

  # 2. Set up your environment variables
  cp .env.example .env
  
  # Add your GROQ_API_KEY to the .env file
  # GROQ_API_KEY=your_api_key_here

  # 3. Run the application
  go run .
  ```

* **Test Commands:**
  ```bash
  # Run the mocked test suite for the AI pipeline and fallback logic
  go test ./ai -v
  ```

---

## AI Disclosure

_**Did you use an AI assistant (Copilot, ChatGPT, etc.)?**_
Yes. I utilized an AI assistant to accelerate boilerplate generation, brainstorm architectural patterns, and generate the synthetic `feed.json` dataset.

_**How did you verify the suggestions?**_
All AI-generated logic and architectural suggestions were strictly verified via local Go compilation and rigorous edge-case testing. Rather than blindly trusting the generated code against a live API, I utilized Go's `httptest` package to spin up a local mock server. This allowed me to intentionally trigger simulated AI failure states (such as HTTP 429 rate limits and 500 internal server errors) to empirically validate that my application's rule-based fallback mechanisms routed correctly and prevented crashes.

_**Give one example of a suggestion you rejected or changed:**_
Initially, I explored relying solely on prompt engineering to force the LLM to output exact category string matches (e.g., "Phishing Scam") and correct dependent booleans. However, when my testing revealed the model would occasionally hallucinate formatting (outputting "PhishingScam" without a space) or fail dependent logic rules, I rejected the purely prompt-based approach. Instead, I engineered a Defensive Programming sanitization layer in Go to intercept, correct, and enforce schema compliance on the AI's raw JSON output before it ever reached the SQLite database.

___

## Tradeoffs & Prioritization

* _**What did you cut to stay within the 4-6 hour limit?**_
  I cut live web-scraping (Nextdoor/Facebook APIs) and user authentication. Building a fully robust web scraper requires handling CAPTCHAs and changing DOM structures, which would have consumed the entire timebox. I opted to use a synthetic JSON dataset to focus on demonstrating concurrent architecture, AI resilience, and state management.

* _**What would you build next if you had more time?**_
    * **Export to Safe Circle:** I would implement a feature allowing users to press a keybind to copy a heavily simplified, text-only version of the actionable checklist to their clipboard for quick sharing in family group chats or SMS.
    * **Persona-Based Filtering:** I would add UI toggles tailored to the target audiences mentioned in the prompt (e.g., an "Elderly Mode" that aggressively filters for physical threats and scams, and a "Remote Worker Mode" that prioritizes data breaches and network security).
    * **Historical Trend Analysis:** Since the application already persists data in SQLite, I would build a separate TUI dashboard pane to display weekly community trends (e.g., "Phishing reports are up 20% in your area this week").

* _**Known limitations:**_
    * The application currently relies on a static `feed.json` file to simulate live network traffic.
    * Third-party LLM rate limits. While the application gracefully handles this via the rule-based fallback mechanism, prolonged API outages degrade the categorization quality to strict keyword matching.