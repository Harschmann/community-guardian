package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/Harschmann/community-guardian/ai"
	"github.com/Harschmann/community-guardian/db"
	"github.com/Harschmann/community-guardian/models"
	"github.com/Harschmann/community-guardian/tui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/joho/godotenv"
)

func main() {
	// 1. Load API Keys from .env
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: No .env file found. Proceeding without one.")
	}

	// 2. Initialize SQLite (This automatically creates guardian.sqlite!)
	db.InitDB("guardian.sqlite")

	// 3. Read the synthetic raw data
	file, err := os.ReadFile("feed.json")
	if err != nil {
		log.Fatalf("Fatal: Failed to read feed.json: %v", err)
	}

	var rawAlerts []models.RawAlert
	if err := json.Unmarshal(file, &rawAlerts); err != nil {
		log.Fatalf("Fatal: Failed to parse JSON: %v", err)
	}

	// 4. Ingest & Process (The Pipeline)
	fmt.Println("🛡️  Community Guardian Initializing...")
	fmt.Println("⏳ Ingesting and analyzing local network feed...")

	for _, raw := range rawAlerts {
		// This routes to AI, and falls back to local rules if the AI fails
		processed := ai.ProcessAlert(raw)

		// Save to SQLite
		if err := db.SaveProcessed(processed); err != nil {
			log.Printf("Warning: Failed to save alert %s: %v\n", processed.ID, err)
		}
	}

	// 5. Boot the Terminal UI
	// We use WithAltScreen to make it take up the full terminal gracefully
	p := tea.NewProgram(tui.InitialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatalf("Fatal: Error running TUI: %v", err)
	}
}
