package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/steevenmentech/bifrost/internal/config"
	"github.com/steevenmentech/bifrost/internal/keyring"
	"github.com/steevenmentech/bifrost/internal/tui/keys"
	"github.com/steevenmentech/bifrost/internal/tui/styles"
	"github.com/steevenmentech/bifrost/internal/tui/views"
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
	config             *config.Config
	keys               keys.KeyMap
	state              ViewState
	selectedIndex      int
	width              int
	height             int
	ready              bool
	err                error
	form               *views.ConnectionFormModel
	selectedConnection *config.Connection
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
		// Global keys that work everywhere
		switch msg.String() {
		case "ctrl+c", "q":
			// Don't quit if in form - let form handle it
			if m.state != ViewConnectionForm {
				return m, tea.Quit
			}
		case "?":
			// TODO: Show help
			return m, nil
		}

		// View-specific keys
		switch m.state {
		case ViewConnections:
			return m.updateConnectionsList(msg)

		case ViewConnectionForm:
			return m.updateConnectionForm(msg)
		}
	}

	return m, nil
}

// updateConnectionForm handles updates for the connection form
func (m Model) updateConnectionForm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.form == nil {
		m.state = ViewConnections
		return m, nil
	}

	// Update the form
	updatedForm, cmd := m.form.Update(msg)
	m.form = &updatedForm

	// Check if form was submitted
	if m.form.IsSubmitted() {
		return m.handleFormSubmit()
	}

	// Check if form was cancelled
	if m.form.IsCancelled() {
		m.form = nil
		m.state = ViewConnections
		return m, nil
	}

	return m, cmd
}

// handleFormSubmit saves the connection from the form
func (m Model) handleFormSubmit() (tea.Model, tea.Cmd) {
	if m.form == nil {
		return m, nil
	}

	// Get connection and password from form
	conn := m.form.GetConnection()
	password := m.form.GetPassword()

	// Save password to keyring if provided
	if password != "" {
		err := keyring.SetConnectionPassword(conn.ID, password)
		if err != nil {
			m.err = fmt.Errorf("failed to save password: %w", err)
			return m, nil
		}
	}

	// Add or update connection in config
	var err error
	if m.form.GetMode() == views.FormModeAdd {
		err = m.config.AddConnection(conn)
	} else {
		err = m.config.UpdateConnection(conn)
	}

	if err != nil {
		m.err = fmt.Errorf("failed to save connection: %w", err)
		return m, nil
	}

	// Reload config to get fresh data
	cfg, err := config.Load()
	if err != nil {
		m.err = fmt.Errorf("failed to reload config: %w", err)
		return m, nil
	}
	m.config = cfg

	// Clear form and return to connections list
	m.form = nil
	m.state = ViewConnections
	m.err = nil

	return m, nil
}

// updateConnectionsList handles key presses in the connections list view
func (m Model) updateConnectionsList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		// Move selection down
		if m.selectedIndex < len(m.config.Connections)-1 {
			m.selectedIndex++
		}
		return m, nil

	case "k", "up":
		// Move selection up
		if m.selectedIndex > 0 {
			m.selectedIndex--
		}
		return m, nil

	case "enter", "l", "right":
		// Select current connection
		if len(m.config.Connections) > 0 {
			return m.handleConnectionSelect()
		}
		return m, nil

	case "a":
		// Add new connection - show form
		return m.showAddConnectionForm()

	case "e":
		// Edit selected connection
		if len(m.config.Connections) > 0 {
			return m.showEditConnectionForm()
		}
		return m, nil

	case "d":
		// Delete selected connection
		if len(m.config.Connections) > 0 {
			return m.handleDeleteConnection()
		}
		return m, nil
	}

	return m, nil
}

// showAddConnectionForm switches to the connection form view in add mode
func (m Model) showAddConnectionForm() (tea.Model, tea.Cmd) {
	form := views.NewConnectionForm(views.FormModeAdd, nil)
	m.form = &form
	m.state = ViewConnectionForm
	m.err = nil
	return m, form.Init()
}

// showEditConnectionForm switches to the connection form view in edit mode
func (m Model) showEditConnectionForm() (tea.Model, tea.Cmd) {
	if m.selectedIndex >= len(m.config.Connections) {
		return m, nil
	}

	conn := &m.config.Connections[m.selectedIndex]
	form := views.NewConnectionForm(views.FormModeEdit, conn)
	m.form = &form
	m.state = ViewConnectionForm
	m.err = nil
	return m, form.Init()
}

// handleDeleteConnection deletes the selected connection
func (m Model) handleDeleteConnection() (tea.Model, tea.Cmd) {
	if m.selectedIndex >= len(m.config.Connections) {
		return m, nil
	}

	conn := m.config.Connections[m.selectedIndex]

	// Delete password from keyring
	_ = keyring.DeleteConnectionPassword(conn.ID)

	// Delete from config
	err := m.config.DeleteConnection(conn.ID)
	if err != nil {
		m.err = fmt.Errorf("failed to delete connection: %w", err)
		return m, nil
	}

	// Adjust selection
	if m.selectedIndex >= len(m.config.Connections) && m.selectedIndex > 0 {
		m.selectedIndex--
	}

	m.err = nil
	return m, nil
}

// handleConnectionSelect is called when user presses Enter on a connection
func (m Model) handleConnectionSelect() (tea.Model, tea.Cmd) {
	if m.selectedIndex >= len(m.config.Connections) {
		return m, nil
	}

	selectedConn := m.config.Connections[m.selectedIndex]

	// Store the selected connection and quit to start SSH
	m.selectedConnection = &selectedConn
	return m, tea.Quit
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
	case ViewConnectionForm:
		if m.form != nil {
			return m.form.View()
		}
		return "Loading form..."
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

		// Highlight the SELECTED item (not just the first one)
		if i == m.selectedIndex {
			line = styles.SelectedStyle.Render(line)
		} else {
			line = styles.ItemStyle.Render(line)
		}

		content += line + "\n"
	}

	// Show error message if any (like the "Selected: ..." message)
	if m.err != nil {
		content += "\n" + styles.SuccessStyle.Render(fmt.Sprintf("  %v", m.err)) + "\n"
	}

	return content
}

// ensureValidSelection makes sure selectedIndex is within bounds
func (m *Model) ensureValidSelection() {
	if len(m.config.Connections) == 0 {
		m.selectedIndex = 0
		return
	}

	if m.selectedIndex >= len(m.config.Connections) {
		m.selectedIndex = len(m.config.Connections) - 1
	}

	if m.selectedIndex < 0 {
		m.selectedIndex = 0
	}
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

// GetSelectedConnection returns the connection selected for SSH (if any)
func (m Model) GetSelectedConnection() *config.Connection {
	return m.selectedConnection
}
