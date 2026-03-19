package ai

import (
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/Harschmann/community-guardian/models"
)

func init() {
	err := os.Setenv("GROQ_API_KEY", "dummy_test_key")
	if err != nil {
		log.Fatalf("failed to set GROQ_API_KEY env var: %v", err)
	}
}

func TestProcessAlert_MockAI_HappyPath(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// Simulating a perfect Llama-3 response
		_, err := w.Write([]byte(`{
          "choices": [{
             "message": {
                "content": "{\"category\": \"Data Breach\", \"is_threat\": true, \"action_plan\": [\"Change passwords.\", \"Enable 2FA.\"]}"
             }
          }]
       }`))
		if err != nil {
			return
		}
	}))
	defer mockServer.Close()

	originalURL := apiURL
	apiURL = mockServer.URL
	defer func() { apiURL = originalURL }() // Restore it when the test finishes

	raw := models.RawAlert{
		ID:        "test-happy",
		Timestamp: "2026-03-19T12:00:00Z",
		Source:    "Mock Test",
		RawText:   "My password was leaked online!",
	}

	// Run the function
	result := ProcessAlert(raw)

	if result.ProcessedBy != "AI" {
		t.Errorf("Expected processed by AI, got '%s'", result.ProcessedBy)
	}
	if result.Category != "Data Breach" {
		t.Errorf("Expected category 'Data Breach', got '%s'", result.Category)
	}
	if !result.IsThreat {
		t.Errorf("Expected IsThreat to be true")
	}
	if len(result.ActionPlan) != 2 {
		t.Errorf("Expected 2 action plan steps, got %d", len(result.ActionPlan))
	}
}

func TestProcessAlert_MockAI_SanitizationEdgeCase(t *testing.T) {
	// Edge Case: The AI hallucinates the category name (no space) and hallucinates the boolean (false)
	// Our defensive programming in processor.go should catch and fix this!
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{
          "choices": [{
             "message": {
                "content": "{\"category\": \"PhishingScam\", \"is_threat\": false, \"action_plan\": []}"
             }
          }]
       }`))
		if err != nil {
			return
		}
	}))
	defer mockServer.Close()

	originalURL := apiURL
	apiURL = mockServer.URL
	defer func() { apiURL = originalURL }()

	raw := models.RawAlert{
		ID:        "test-edge",
		Timestamp: "2026-03-19T12:05:00Z",
		Source:    "Mock Test",
		RawText:   "Click here to claim your prize!",
	}

	result := ProcessAlert(raw)

	if result.Category != "Phishing Scam" {
		t.Errorf("Sanitization failed! Expected 'Phishing Scam', got '%s'", result.Category)
	}
	if !result.IsThreat {
		t.Errorf("Sanitization failed! Expected IsThreat to be forced to true, got %v", result.IsThreat)
	}
}

func TestProcessAlert_API_FailureFallback(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer mockServer.Close()

	originalURL := apiURL
	apiURL = mockServer.URL
	defer func() { apiURL = originalURL }()

	raw := models.RawAlert{
		ID:        "test-fallback",
		Timestamp: "2026-03-19T12:10:00Z",
		Source:    "Mock Test",
		RawText:   "I got a fake link asking for my SSN.",
	}

	result := ProcessAlert(raw)

	if result.ProcessedBy != "Rule-Based Fallback" {
		t.Errorf("Expected fallback engine to take over, got '%s'", result.ProcessedBy)
	}
	if result.Category != "Phishing Scam" {
		t.Errorf("Expected fallback to categorize as 'Phishing Scam', got '%s'", result.Category)
	}
}
