package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/steevenmentech/bifrost/internal/config"
	"github.com/steevenmentech/bifrost/internal/tui/keys"
	"github.com/steevenmentech/bifrost/internal/tui/styles"
)

// ViewState represents which view is currently active
type ViewState int

const (
	ViewConnections ViewState = iota
	ViewConnectionForm
	ViewSSH
	ViewSFTP
)

// Model is the main Bubble Tea model
type Model struct {
	config *config.Config
	keys   keys.KeyMap
	state  ViewState
	width  int
	height int
	ready  bool
	err    error
}

// New creates a new TUI model
func New() (*Model, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	return &Model{
		config: cfg,
		keys:   keys.DefaultKeyMap(),
		state:  ViewConnections,
	}, nil
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "?":
			// TODO: Show help
			return m, nil
		}
	}

	return m, nil
}

// View renders the UI
func (m Model) View() string {
	if !m.ready {
		return "Initializing..."
	}

	// Build the UI
	title := m.renderTitle()
	content := m.renderContent()
	statusBar := m.renderStatusBar()

	// Calculate heights
	titleHeight := lipgloss.Height(title)
	statusHeight := lipgloss.Height(statusBar)
	contentHeight := m.height - titleHeight - statusHeight - 2

	// Style the content area with proper height
	contentStyle := lipgloss.NewStyle().
		Height(contentHeight).
		Width(m.width)

	styledContent := contentStyle.Render(content)

	// Combine all parts
	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		styledContent,
		statusBar,
	)
}

// renderTitle renders the title bar
func (m Model) renderTitle() string {
	title := styles.TitleStyle.Render("🌈 Bifrost - SSH & SFTP Manager")

	// Add view indicator
	viewName := m.getViewName()
	viewIndicator := styles.SubtleStyle.Render(fmt.Sprintf(" [%s]", viewName))

	return lipgloss.JoinHorizontal(lipgloss.Top, title, viewIndicator)
}

// renderContent renders the main content area
func (m Model) renderContent() string {
	switch m.state {
	case ViewConnections:
		return m.renderConnectionsList()
	default:
		return "View not implemented yet"
	}
}

// renderConnectionsList renders the connections list view
func (m Model) renderConnectionsList() string {
	if len(m.config.Connections) == 0 {
		return styles.SubtleStyle.Render("\n  No connections yet. Press 'a' to add one.")
	}

	var content string
	content += "\n  Connections:\n\n"

	for i, conn := range m.config.Connections {
		icon := conn.Icon
		if icon == "" {
			icon = "🖥️"
		}

		line := fmt.Sprintf("  %s  %s", icon, conn.Label)
		if conn.Host != "" {
			line += styles.SubtleStyle.Render(fmt.Sprintf("  (%s)", conn.Host))
		}

		// Highlight first item for now (we'll add proper selection later)
		if i == 0 {
			line = styles.SelectedStyle.Render(line)
		} else {
			line = styles.ItemStyle.Render(line)
		}

		content += line + "\n"
	}

	return content
}

// renderStatusBar renders the bottom status bar
func (m Model) renderStatusBar() string {
	helpText := "↑↓/jk navigate • enter select • a add • e edit • d delete • q quit • ? help"

	statusText := styles.HelpStyle.Render(helpText)

	// Create a bar that spans the full width
	bar := lipgloss.NewStyle().
		Width(m.width).
		Foreground(styles.Muted).
		Render(statusText)

	return styles.StatusBarStyle.Render(bar)
}

// getViewName returns the name of the current view
func (m Model) getViewName() string {
	switch m.state {
	case ViewConnections:
		return "Connections"
	case ViewConnectionForm:
		return "Add/Edit Connection"
	case ViewSSH:
		return "SSH Terminal"
	case ViewSFTP:
		return "SFTP Browser"
	default:
		return "Unknown"
	}
}
