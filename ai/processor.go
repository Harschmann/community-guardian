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

type openAIRequest struct {
	Model    string    `json:"model"`
	Messages []message `json:"messages"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func ProcessAlert(raw models.RawAlert) models.ProcessedAlert {
	alert, err := callOpenAI(raw)
	if err != nil {
		fmt.Printf("⚠️ AI unavailable for alert %s (Error: %v). Routing to fallback...\n", raw.ID, err)
		return fallbackProcess(raw)
	}
	return alert
}

func callOpenAI(raw models.RawAlert) (models.ProcessedAlert, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" || apiKey == "your_api_key_here" {
		return models.ProcessedAlert{}, fmt.Errorf("invalid or missing API key")
	}

	// We use strict prompting to force the AI to return clean JSON matching our struct
	systemPrompt := `You are a cybersecurity analyzer. Analyze the community post. Determine if it is a threat (true/false), categorize it, and provide a 3-step action_plan array.
	Output STRICTLY as JSON: {"category": "Phishing/Breach/Noise", "is_threat": true/false, "action_plan": ["step 1", "step 2", "step 3"]}`

	reqBody := openAIRequest{
		Model: "gpt-3.5-turbo", // Fast and cheap for the prototype
		Messages: []message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: raw.RawText},
		},
	}

	jsonData, _ := json.Marshal(reqBody)

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return models.ProcessedAlert{}, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	// Set a strict 5-second timeout so the TUI doesn't hang forever
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return models.ProcessedAlert{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return models.ProcessedAlert{}, fmt.Errorf("OpenAI API returned status: %d", resp.StatusCode)
	}

	bodyBytes, _ := io.ReadAll(resp.Body)

	// Parse OpenAI's nested response format
	var openAIResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(bodyBytes, &openAIResp); err != nil {
		return models.ProcessedAlert{}, err
	}

	if len(openAIResp.Choices) == 0 {
		return models.ProcessedAlert{}, fmt.Errorf("no AI choices returned")
	}

	// Unmarshal the string content back into our Go struct
	content := openAIResp.Choices[0].Message.Content
	var parsed models.ProcessedAlert
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		return models.ProcessedAlert{}, err
	}

	// Reattach the metadata from the raw alert
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

	// Simple keyword matching for digital defense
	if strings.Contains(text, "password") || strings.Contains(text, "breach") {
		alert.Category = "Data Breach"
		alert.IsThreat = true
		alert.ActionPlan = []string{
			"Change passwords immediately for affected services.",
			"Enable Two-Factor Authentication (2FA).",
			"Monitor bank statements for unusual activity.",
		}
	} else if strings.Contains(text, "link") || strings.Contains(text, "ssn") || strings.Contains(text, "bank") {
		alert.Category = "Phishing Scam"
		alert.IsThreat = true
		alert.ActionPlan = []string{
			"Do NOT click any links or download attachments.",
			"Report the message as spam or phishing.",
			"Contact the institution directly using their official website.",
		}
	} else {
		alert.Category = "Noise / General Info"
		alert.IsThreat = false
		alert.ActionPlan = []string{}
	}

	return alert
}
