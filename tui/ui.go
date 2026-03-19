package tui

import (
	"fmt"
	"strings"

	"github.com/Harschmann/community-guardian/db"
	"github.com/Harschmann/community-guardian/models"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FAFAFA")).Background(lipgloss.Color("#7D56F4")).Padding(0, 1).MarginBottom(1)
	paneStyle   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1, 2).Width(50).Height(15)
	selected    = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00")).Bold(true)
	unselected  = lipgloss.NewStyle().Foreground(lipgloss.Color("#FAFAFA"))
	subtext     = lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
	alertBadge  = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Bold(true)
	filterBadge = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FFFF")).Bold(true)
)

type model struct {
	allThreats     []models.ProcessedAlert // The master list from the DB
	visibleThreats []models.ProcessedAlert // The filtered list we display
	cursor         int
	err            error
	filterIndex    int
	filters        []string
}

type RefreshMsg struct{}

func InitialModel() model {
	threats, err := db.GetThreats()
	return model{
		allThreats:     threats,
		visibleThreats: threats, // Initially show all
		err:            err,
		filterIndex:    0,
		filters:        []string{"All", "Physical Threat", "Phishing Scam", "Data Breach"},
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

// Helper to update the visible list based on the active filter
func (m *model) applyFilter() {
	m.cursor = 0
	currentFilter := m.filters[m.filterIndex]

	if currentFilter == "All" {
		m.visibleThreats = m.allThreats
		return
	}

	var filtered []models.ProcessedAlert
	for _, t := range m.allThreats {
		if t.Category == currentFilter {
			filtered = append(filtered, t)
		}
	}
	m.visibleThreats = filtered
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case RefreshMsg:
		threats, err := db.GetThreats()
		if err == nil {
			m.allThreats = threats
			m.applyFilter()
		}
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.visibleThreats)-1 {
				m.cursor++
			}
		case "f":
			m.filterIndex = (m.filterIndex + 1) % len(m.filters)
			m.applyFilter()
		}
	}
	return m, nil
}

func (m model) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error loading data: %v\nPress 'q' to quit.", m.err)
	}

	header := titleStyle.Render("🛡️  Community Guardian - Active Threats")

	currentFilter := m.filters[m.filterIndex]
	filterStatus := fmt.Sprintf(" Active Filter: %s", filterBadge.Render(currentFilter))

	if len(m.visibleThreats) == 0 {
		emptyMsg := fmt.Sprintf("\nNo threats found for category: %s\nPress 'f' to change filter or 'q' to quit.", currentFilter)
		return fmt.Sprintf("%s\n%s\n%s", header, filterStatus, emptyMsg)
	}

	var listBuilder strings.Builder
	for i, t := range m.visibleThreats {
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

	var detailsBuilder strings.Builder
	active := m.visibleThreats[m.cursor]

	detailsBuilder.WriteString(alertBadge.Render("⚠️  "+active.Category) + "\n\n")
	detailsBuilder.WriteString("Processed By: " + active.ProcessedBy + "\n\n")
	detailsBuilder.WriteString(subtext.Render("Action Plan:") + "\n")

	for i, step := range active.ActionPlan {
		detailsBuilder.WriteString(fmt.Sprintf("%d. %s\n", i+1, step))
	}

	rightPane := paneStyle.Render(detailsBuilder.String())

	mainUI := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)
	footer := subtext.Render("\n  (j/k or ↑/↓ to navigate • f to filter • q to quit)")

	return fmt.Sprintf("%s\n%s\n\n%s\n%s", header, filterStatus, mainUI, footer)
}
