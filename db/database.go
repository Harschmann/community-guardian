package db

import (
	"database/sql"
	"encoding/json"
	"log"

	"github.com/Harschmann/community-guardian/models"
)

var DB *sql.DB

func InitDB(filepath string) {
	var err error
	DB, err := sql.Open("sqlite", filepath)
	if err != nil {
		log.Fatalf("Fatal: Failed to open SQLite database: %v", err)
	}

	createTablesQuery := `
	CREATE TABLE IF NOT EXISTS raw_alerts (
    	id TEXT PRIMARY KEY, 
    	timestamp TEXT, 
    	source TEXT, 
    	raw_text TEXT
	);
	CREATE TABLE IF NOT EXISTS processed_threats (
    	id TEXT PRIMARY KEY, 
    	timestamp TEXT,
    	source TEXT,
    	category TEXT,
    	is_threat BOOLEAN,
    	action_plan TEXT, -- we will store the json array as a string here 
    	processed_by TEXT
	);
	`

	_, err = DB.Exec(createTablesQuery)
	if err != nil {
		log.Fatalf("Fatal: Failed to create tables: %v", err)
	}
}

func SaveProcessed(alert models.ProcessedAlert) error {
	actionPlanJSON, _ := json.Marshal(alert.ActionPlan)
	query := `
           INSERT INTO processed_threats (id, timestamp, source, category, is_threat, action_plan, processed_by)
           VALUES (?, ?, ?, ?, ?, ?, ?)
           ON CONFLICT(id) DO NOTHING;  -- prevent duplicate entries if we restart the app
    `
	_, err := DB.Exec(query, alert.ID, alert.Timestamp, alert.Source, alert.Category, alert.IsThreat, string(actionPlanJSON), alert.ProcessedBy)
	return err
}

func GetThreats() ([]models.ProcessedAlert, error) {
	query := `SELECT id, timestamp, source, category, is_threat, action_plan, processed_by FROM processed_threats WHERE is_threat = true ORDER BY timestamp DESC`
	rows, err := DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var threats []models.ProcessedAlert
	for rows.Next() {
		var t models.ProcessedAlert
		var actionPlanStr string

		err := rows.Scan(&t.ID, &t.Timestamp, &t.Source, &t.Category, &t.IsThreat, &actionPlanStr, &t.ProcessedBy)
		if err != nil {
			log.Printf("Warning: Error scanning database row: %v", err)
			continue
		}

		json.Unmarshal([]byte(actionPlanStr), &t.ActionPlan)
		threats = append(threats, t)
	}
	return threats, nil
}
