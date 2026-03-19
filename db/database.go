package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"github.com/Harschmann/community-guardian/models"
	_ "modernc.org/sqlite"
)

var DB *sql.DB

func InitDB(filepath string) {
	var err error
	DB, err = sql.Open("sqlite", filepath)
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
    	action_plan TEXT, -- we will store the JSON array as a string here 
    	processed_by TEXT
	);
	`

	_, err = DB.Exec(createTablesQuery)
	if err != nil {
		log.Fatalf("Fatal: Failed to create tables: %v", err)
	}
}

func SaveProcessed(alert models.ProcessedAlert) error {
	actionPlanJSON, err := json.Marshal(alert.ActionPlan)
	if err != nil {
		return fmt.Errorf("failed to marshal action plan: %v", err)
	}
	query := `
           INSERT INTO processed_threats (id, timestamp, source, category, is_threat, action_plan, processed_by)
           VALUES (?, ?, ?, ?, ?, ?, ?)
           ON CONFLICT(id) DO NOTHING;  -- prevent duplicate entries if we restart the app
    `
	_, err = DB.Exec(query, alert.ID, alert.Timestamp, alert.Source, alert.Category, alert.IsThreat, string(actionPlanJSON), alert.ProcessedBy)
	return err
}

func GetThreats() ([]models.ProcessedAlert, error) {
	query := `SELECT id, timestamp, source, category, is_threat, action_plan, processed_by FROM processed_threats WHERE is_threat = true ORDER BY timestamp DESC`
	rows, err := DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			log.Fatalf("Failed to close rows: %v", err)
		}
	}(rows)

	var threats []models.ProcessedAlert
	for rows.Next() {
		var t models.ProcessedAlert
		var actionPlanStr string

		err := rows.Scan(&t.ID, &t.Timestamp, &t.Source, &t.Category, &t.IsThreat, &actionPlanStr, &t.ProcessedBy)
		if err != nil {
			log.Printf("Warning: Error scanning database row: %v", err)
			continue
		}

		err = json.Unmarshal([]byte(actionPlanStr), &t.ActionPlan)
		if err != nil {
			log.Printf("Warning: Error unmarshalling action plan json for alert %s: %v", t.ID, err)
		}
		threats = append(threats, t)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating through database rows: %v", err)
	}

	return threats, nil
}

func AlertExists(id string) bool {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM processed_threats WHERE id = ?)`
	err := DB.QueryRow(query, id).Scan(&exists)
	if err != nil {
		return false
	}
	return exists
}
