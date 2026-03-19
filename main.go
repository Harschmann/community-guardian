package main

import (
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/Harschmann/community-guardian/ai"
	"github.com/Harschmann/community-guardian/db"
	"github.com/Harschmann/community-guardian/models"
	"github.com/Harschmann/community-guardian/tui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: No .env file found. Proceeding without one.")
	}

	db.InitDB("guardian.sqlite")

	file, err := os.ReadFile("feed.json")
	if err != nil {
		log.Fatalf("Fatal: Failed to read feed.json: %v", err)
	}

	var rawAlerts []models.RawAlert
	if err := json.Unmarshal(file, &rawAlerts); err != nil {
		log.Fatalf("Fatal: Failed to parse JSON: %v", err)
	}

	// 1. Initialize the bubbletea program FIRST
	p := tea.NewProgram(tui.InitialModel(), tea.WithAltScreen())

	// 2. Spin up a background worker for ingestion
	go func() {
		for _, raw := range rawAlerts {
			if db.AlertExists(raw.ID) {
				continue
			}

			processed := ai.ProcessAlert(raw)

			if err := db.SaveProcessed(processed); err == nil {
				p.Send(tui.RefreshMsg{})
			}

			time.Sleep(4 * time.Second)
		}
	}()

	// 3. Run the UI on the main thread
	if _, err := p.Run(); err != nil {
		log.Fatalf("Fatal: Error running TUI: %v", err)
	}
}
