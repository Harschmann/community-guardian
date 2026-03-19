package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Harschmann/community-guardian/models"
)

type chatRequest struct {
	Model          string         `json:"model"`
	Messages       []message      `json:"messages"`
	MaxTokens      int            `json:"max_tokens,omitempty"`
	ResponseFormat map[string]any `json:"response_format,omitempty"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

var apiURL = "https://api.groq.com/openai/v1/chat/completions"

// ProcessAlert attempts to use AI first. If it fails, it instantly triggers the manual fallback.
func ProcessAlert(raw models.RawAlert) models.ProcessedAlert {
	alert, err := callOpenRouter(raw)
	if err != nil {
		logFile, _ := os.OpenFile("debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		defer logFile.Close()
		log.SetOutput(logFile)
		log.Printf("⚠️ AI unavailable for alert %s. Error: %v\n", raw.ID, err)

		return fallbackProcess(raw)
	}
	return alert
}

func callOpenRouter(raw models.RawAlert) (models.ProcessedAlert, error) {
	apiKey := os.Getenv("GROQ_API_KEY")
	if apiKey == "" {
		return models.ProcessedAlert{}, fmt.Errorf("invalid or missing API key")
	}

	systemPrompt := `You are a cybersecurity analyzer. Analyze the community post. Determine if it is a threat (true/false), categorize it, and provide a 3-step action_plan array.
	Output STRICTLY as JSON: {"category": "category_name", "is_threat": true/false, "action_plan": ["step 1", "step 2", "step 3"]}
	For the category_name, you MUST choose exactly one of these options: "Physical Threat", "Phishing Scam", "Data Breach", or "Noise".`

	reqBody := chatRequest{
		Model: "llama-3.1-8b-instant",
		Messages: []message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: raw.RawText},
		},
		MaxTokens: 1000,
		ResponseFormat: map[string]any{
			"type": "json_object",
		},
	}

	jsonData, _ := json.Marshal(reqBody)

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return models.ProcessedAlert{}, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("HTTP-Referer", "http://localhost:8080")
	req.Header.Set("X-Title", "Community Guardian CLI")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return models.ProcessedAlert{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return models.ProcessedAlert{}, fmt.Errorf("Groq API returned status: %d", resp.StatusCode)
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

	start := strings.Index(content, "{")
	end := strings.LastIndex(content, "}")

	if start == -1 || end == -1 || start >= end {
		return models.ProcessedAlert{}, fmt.Errorf("could not find valid JSON object in AI response")
	}

	content = content[start : end+1]

	var parsed models.ProcessedAlert
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		return models.ProcessedAlert{}, err
	}

	if parsed.Category == "PhishingScam" || parsed.Category == "Phishing" || parsed.Category == "Phishing Scam" {
		parsed.Category = "Phishing Scam"
	} else if parsed.Category == "Data Breach" || parsed.Category == "Breach" {
		parsed.Category = "Data Breach"
	} else if parsed.Category == "PhysicalThreat" || parsed.Category == "Physical Threat" {
		parsed.Category = "Physical Threat"
	} else if parsed.Category != "Phishing Scam" && parsed.Category != "Data Breach" && parsed.Category != "Physical Threat" {
		parsed.Category = "Noise"
	}

	// 2. Enforce dependent logic (If it's a threat category, it MUST be a threat)
	if parsed.Category == "Phishing Scam" || parsed.Category == "Data Breach" || parsed.Category == "Physical Threat" {
		parsed.IsThreat = true
	} else {
		parsed.IsThreat = false
	}

	parsed.ID = raw.ID
	parsed.Timestamp = raw.Timestamp
	parsed.Source = raw.Source
	parsed.ProcessedBy = "AI"

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
