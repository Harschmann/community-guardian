package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Harschmann/community-guardian/models"
)

type chatRequest struct {
	Model    string    `json:"model"`
	Messages []message `json:"messages"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ProcessAlert attempts to use AI first. If it fails, it instantly triggers the manual fallback.
func ProcessAlert(raw models.RawAlert) models.ProcessedAlert {
	alert, err := callOpenRouter(raw)
	if err != nil {
		fmt.Printf("⚠️ AI unavailable for alert %s (Error: %v). Routing to fallback...\n", raw.ID, err)
		return fallbackProcess(raw)
	}
	return alert
}

// callOpenRouter connects to OpenRouter to categorize and extract actionable steps
func callOpenRouter(raw models.RawAlert) (models.ProcessedAlert, error) {
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" || apiKey == "your_api_key_here" {
		return models.ProcessedAlert{}, fmt.Errorf("invalid or missing API key")
	}

	systemPrompt := `You are a cybersecurity analyzer. Analyze the community post. Determine if it is a threat (true/false), categorize it, and provide a 3-step action_plan array.
	Output STRICTLY as JSON: {"category": "Phishing/Breach/Noise", "is_threat": true/false, "action_plan": ["step 1", "step 2", "step 3"]}`

	reqBody := chatRequest{
		// Using a highly capable, 100% free model on OpenRouter
		Model: "nvidia/nemotron-3-super-120b-a12b:free",
		Messages: []message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: raw.RawText},
		},
	}

	jsonData, _ := json.Marshal(reqBody)

	// Pointing directly to OpenRouter's endpoint
	req, err := http.NewRequest("POST", "https://openrouter.ai/api/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return models.ProcessedAlert{}, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	// OpenRouter specific headers (good practice to include them)
	req.Header.Set("HTTP-Referer", "http://localhost:8080")
	req.Header.Set("X-Title", "Community Guardian CLI")

	// 5-second timeout ensures the terminal UI never freezes if the API lags
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return models.ProcessedAlert{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return models.ProcessedAlert{}, fmt.Errorf("OpenRouter API returned status: %d", resp.StatusCode)
	}

	bodyBytes, _ := io.ReadAll(resp.Body)

	var chatResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(bodyBytes, &chatResp); err != nil {
		return models.ProcessedAlert{}, err
	}

	if len(chatResp.Choices) == 0 {
		return models.ProcessedAlert{}, fmt.Errorf("no AI choices returned")
	}

	content := chatResp.Choices[0].Message.Content

	// Clean up any potential markdown formatting the LLM might wrap the JSON in
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var parsed models.ProcessedAlert
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		return models.ProcessedAlert{}, err
	}

	// Reattach the metadata
	parsed.ID = raw.ID
	parsed.Timestamp = raw.Timestamp
	parsed.Source = raw.Source
	parsed.ProcessedBy = "OpenRouter AI"

	return parsed, nil
}

// fallbackProcess is our manual rule-based engine if the AI is offline
func fallbackProcess(raw models.RawAlert) models.ProcessedAlert {
	text := strings.ToLower(raw.RawText)

	alert := models.ProcessedAlert{
		ID:          raw.ID,
		Timestamp:   raw.Timestamp,
		Source:      raw.Source,
		ProcessedBy: "Rule-Based Fallback",
	}

	if strings.Contains(text, "password") || strings.Contains(text, "breach") {
		alert.Category = "Data Breach"
		alert.IsThreat = true
		alert.ActionPlan = []string{
			"Change passwords immediately for affected services.",
			"Enable Two-Factor Authentication (2FA).",
			"Monitor bank statements for unusual activity.",
		}
	} else if strings.Contains(text, "link") || strings.Contains(text, "ssn") || strings.Contains(text, "bank") || strings.Contains(text, "fee") {
		alert.Category = "Phishing Scam"
		alert.IsThreat = true
		alert.ActionPlan = []string{
			"Do NOT click any links or download attachments.",
			"Report the message as spam or phishing.",
			"Contact the institution directly using their official website.",
		}
	} else if strings.Contains(text, "scammer") || strings.Contains(text, "police") || strings.Contains(text, "broken into") {
		alert.Category = "Physical Threat"
		alert.IsThreat = true
		alert.ActionPlan = []string{
			"Lock all doors and windows.",
			"Do not engage with suspicious individuals.",
			"Contact local authorities via non-emergency lines if needed.",
		}
	} else {
		alert.Category = "Noise / General Info"
		alert.IsThreat = false
		alert.ActionPlan = []string{}
	}

	return alert
}
