package tui

import (
	"fmt"
	"strings"

	"github.com/Harschmann/community-guardian/db"
	"github.com/Harschmann/community-guardian/models"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Styling definitions
var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FAFAFA")).Background(lipgloss.Color("#7D56F4")).Padding(0, 1).MarginBottom(1)
	paneStyle  = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1, 2).Width(50).Height(15)
	selected   = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00")).Bold(true)
	unselected = lipgloss.NewStyle().Foreground(lipgloss.Color("#FAFAFA"))
	subtext    = lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
	alertBadge = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Bold(true)
)

type model struct {
	threats []models.ProcessedAlert
	cursor  int
	err     error
}

func InitialModel() model {
	// Fetch ONLY the verified threats from our SQLite database
	threats, err := db.GetThreats()
	return model{
		threats: threats,
		err:     err,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.threats)-1 {
				m.cursor++
			}
		}
	}
	return m, nil
}

func (m model) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error loading data: %v\nPress 'q' to quit.", m.err)
	}

	if len(m.threats) == 0 {
		return "No verified threats found in your area. Stay safe!\nPress 'q' to quit."
	}

	header := titleStyle.Render("🛡️  Community Guardian - Active Threats")

	// Render the left pane (The List)
	var listBuilder strings.Builder
	for i, t := range m.threats {
		cursorStr := "  "
		itemStyle := unselected

		if m.cursor == i {
			cursorStr = "> "
			itemStyle = selected
		}

		timeStr := t.Timestamp[11:16] // Just grab the HH:MM
		row := fmt.Sprintf("%s%s [%s]\n   %s", cursorStr, t.Category, timeStr, subtext.Render("via "+t.Source))
		listBuilder.WriteString(itemStyle.Render(row) + "\n\n")
	}
	leftPane := paneStyle.Render(listBuilder.String())

	// Render the right pane (The Details)
	var detailsBuilder strings.Builder
	active := m.threats[m.cursor]

	detailsBuilder.WriteString(alertBadge.Render("⚠️  "+active.Category) + "\n\n")
	detailsBuilder.WriteString("Processed By: " + active.ProcessedBy + "\n\n")
	detailsBuilder.WriteString(subtext.Render("Action Plan:") + "\n")

	for i, step := range active.ActionPlan {
		detailsBuilder.WriteString(fmt.Sprintf("%d. %s\n", i+1, step))
	}

	rightPane := paneStyle.Render(detailsBuilder.String())

	// Join them side-by-side
	mainUI := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)
	footer := subtext.Render("\n  (j/k or ↑/↓ to navigate • q to quit)")

	return fmt.Sprintf("%s\n%s\n%s", header, mainUI, footer)
}
