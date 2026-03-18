package models

type RawAlert struct {
	ID        string `json:"id"`
	Timestamp string `json:"timestamp"`
	Source    string `json:"source"`
	RawText   string `json:"raw_text"`
}

type ProcessedAlert struct {
	ID          string   `json:"id"`
	Timestamp   string   `json:"timestamp"`
	Source      string   `json:"source"`
	Category    string   `json:"category"`     // eg: Noise, Phishing Scam, Data Breach
	IsThreat    bool     `json:"is_threat"`    // True, if it needs to be shown to the user
	ActionPlan  []string `json:"action_plan"`  // The 1-2-3 actionable checklist
	ProcessedBy string   `json:"processed_by"` // "AI" or "Rule-Based" fallback
}
