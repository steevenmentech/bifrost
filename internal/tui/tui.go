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
	ViewSelectionMenu
	ViewCredentials
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
	credentialsManager *views.CredentialsManagerModel
	selectedConnection *config.Connection
	menuSelection      int // 0=SSH, 1=SFTP

	// Confirmation modal
	confirmationModal    *views.ConfirmationModalModel
	showingConfirmation  bool
	confirmationCallback func() (Model, tea.Cmd) // Called when user confirms
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

		// Pass window size to credentials manager if active
		if m.state == ViewCredentials && m.credentialsManager != nil {
			updatedManager, cmd := m.credentialsManager.Update(msg)
			m.credentialsManager = updatedManager
			return m, cmd
		}

		return m, nil

	case tea.KeyMsg:
		// If showing confirmation modal, handle it first
		if m.showingConfirmation && m.confirmationModal != nil {
			return m.updateConfirmationModal(msg)
		}

		// Global keys that work everywhere
		switch msg.String() {
		case "ctrl+c", "q":
			// Don't quit if in form or credentials view - let them handle it
			if m.state != ViewConnectionForm && m.state != ViewCredentials {
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

		case ViewSelectionMenu:
			return m.updateSelectionMenu(msg)

		case ViewCredentials:
			return m.updateCredentialsManager(msg)
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

	case "c":
		// Open credentials manager
		return m.showCredentialsManager()
	}

	return m, nil
}

// updateConfirmationModal handles updates for the confirmation modal
func (m Model) updateConfirmationModal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.confirmationModal == nil {
		m.showingConfirmation = false
		return m, nil
	}

	// Update the modal
	updatedModal, cmd := m.confirmationModal.Update(msg)
	m.confirmationModal = &updatedModal

	// Check if user confirmed
	if m.confirmationModal.IsConfirmed() {
		m.showingConfirmation = false
		m.confirmationModal = nil

		// Execute the callback if set
		if m.confirmationCallback != nil {
			callback := m.confirmationCallback
			m.confirmationCallback = nil
			return callback()
		}
		return m, nil
	}

	// Check if user cancelled
	if m.confirmationModal.IsCancelled() {
		m.showingConfirmation = false
		m.confirmationModal = nil
		m.confirmationCallback = nil
		return m, nil
	}

	return m, cmd
}

// updateCredentialsManager handles updates for the credentials manager
func (m Model) updateCredentialsManager(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.credentialsManager == nil {
		m.state = ViewConnections
		return m, nil
	}

	// Update the credentials manager
	updatedManager, cmd := m.credentialsManager.Update(msg)
	m.credentialsManager = updatedManager

	// Check if done
	if m.credentialsManager.IsDone() {
		// Get updated config
		m.config = m.credentialsManager.GetConfig()
		m.credentialsManager = nil
		m.state = ViewConnections
		return m, nil
	}

	return m, cmd
}

// showCredentialsManager switches to the credentials manager view
func (m Model) showCredentialsManager() (tea.Model, tea.Cmd) {
	manager := views.NewCredentialsManager(m.config, m.keys)
	manager.SetSize(m.width, m.height)
	m.credentialsManager = manager
	m.state = ViewCredentials
	return m, manager.Init()
}

// updateSelectionMenu handles key presses in the selection menu
func (m Model) updateSelectionMenu(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		// Move selection down
		if m.menuSelection < 1 {
			m.menuSelection++
		}
		return m, nil

	case "k", "up":
		// Move selection up
		if m.menuSelection > 0 {
			m.menuSelection--
		}
		return m, nil

	case "enter", "l", "right":
		// Confirm selection and quit to start session
		return m, tea.Quit

	case "esc", "h", "left":
		// Go back to connections list
		m.state = ViewConnections
		m.selectedConnection = nil
		m.menuSelection = 0
		return m, nil
	}

	return m, nil
}

// showAddConnectionForm switches to the connection form view in add mode
func (m Model) showAddConnectionForm() (tea.Model, tea.Cmd) {
	form := views.NewConnectionFormWithCredentials(views.FormModeAdd, nil, m.config.Credentials)
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
	form := views.NewConnectionFormWithCredentials(views.FormModeEdit, conn, m.config.Credentials)
	m.form = &form
	m.state = ViewConnectionForm
	m.err = nil
	return m, form.Init()
}

// handleDeleteConnection shows confirmation modal for deleting connection
func (m Model) handleDeleteConnection() (tea.Model, tea.Cmd) {
	if m.selectedIndex >= len(m.config.Connections) {
		return m, nil
	}

	conn := m.config.Connections[m.selectedIndex]

	// Show confirmation modal
	modal := views.NewConfirmationModal(
		"Delete Connection",
		fmt.Sprintf("Are you sure you want to delete '%s'?", conn.Label),
	)
	m.confirmationModal = &modal
	m.showingConfirmation = true

	// Set callback for when user confirms
	m.confirmationCallback = func() (Model, tea.Cmd) {
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

	return m, modal.Init()
}

// handleConnectionSelect is called when user presses Enter on a connection
func (m Model) handleConnectionSelect() (tea.Model, tea.Cmd) {
	if m.selectedIndex >= len(m.config.Connections) {
		return m, nil
	}

	// Store selected connection and show menu
	m.selectedConnection = &m.config.Connections[m.selectedIndex]
	m.state = ViewSelectionMenu
	m.menuSelection = 0 // Default to SSH
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

// renderTitle renders the title bar with ASCII logo
func (m Model) renderTitle() string {
	// Use compact logo or full logo based on width
	var logo string
	if m.width < 60 {
		logo = styles.LogoCompact
	} else {
		logo = styles.Logo
	}

	styledLogo := styles.LogoStyle.Render(logo)

	// Add view indicator below
	viewName := m.getViewName()
	viewIndicator := styles.SubtleStyle.Render(fmt.Sprintf("[%s]", viewName))

	return lipgloss.JoinVertical(lipgloss.Left, styledLogo, viewIndicator)
}

// renderContent renders the main content area
func (m Model) renderContent() string {
	var baseContent string

	switch m.state {
	case ViewConnections:
		baseContent = m.renderConnectionsList()
	case ViewConnectionForm:
		if m.form != nil {
			// Center the form like a modal
			formContent := m.form.View()
			return lipgloss.Place(
				m.width,
				m.height-15,
				lipgloss.Center,
				lipgloss.Center,
				formContent,
				lipgloss.WithWhitespaceChars(" "),
				lipgloss.WithWhitespaceForeground(styles.Dim),
			)
		} else {
			baseContent = "Loading form..."
		}
	case ViewSelectionMenu:
		baseContent = m.renderSelectionMenu()
	case ViewCredentials:
		if m.credentialsManager != nil {
			baseContent = m.credentialsManager.View()
		} else {
			baseContent = "Loading credentials..."
		}
	default:
		baseContent = "View not implemented yet"
	}

	// If showing confirmation modal, overlay it on top
	if m.showingConfirmation && m.confirmationModal != nil {
		return m.renderConfirmationModal(baseContent)
	}

	return baseContent
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
			icon = "\uf233" // Nerd Font server icon
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

// renderConfirmationModal renders the confirmation modal centered over base content
func (m Model) renderConfirmationModal(baseContent string) string {
	if m.confirmationModal == nil {
		return baseContent
	}

	modal := m.confirmationModal.View()

	// Center the modal on screen
	return lipgloss.Place(
		m.width,
		m.height-15, // Account for title and status bar
		lipgloss.Center,
		lipgloss.Center,
		modal,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(styles.Dim),
	)
}

// renderSelectionMenu renders a floating modal for SSH/SFTP selection
func (m Model) renderSelectionMenu() string {
	if m.selectedConnection == nil {
		return "Error: No connection selected"
	}

	// Build modal content
	title := styles.ModalTitleStyle.Render(fmt.Sprintf("Connect to: %s", m.selectedConnection.Label))

	// SSH option
	sshText := " \uf120  SSH Terminal"
	if m.menuSelection == 0 {
		sshText = styles.SelectedStyle.Render(sshText)
	} else {
		sshText = styles.ItemStyle.Render(sshText)
	}

	// SFTP option
	sftpText := " \uf07c  SFTP Browser"
	if m.menuSelection == 1 {
		sftpText = styles.SelectedStyle.Render(sftpText)
	} else {
		sftpText = styles.ItemStyle.Render(sftpText)
	}

	help := styles.SubtleStyle.Render("↑↓ navigate • enter select • esc cancel")

	modalContent := lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		sshText,
		sftpText,
		"",
		help,
	)

	// Style the modal
	modal := styles.ModalStyle.Render(modalContent)

	// Center the modal on screen
	modalWidth := lipgloss.Width(modal)
	modalHeight := lipgloss.Height(modal)

	// Calculate position to center
	x := (m.width - modalWidth) / 2
	y := (m.height - modalHeight) / 3 // Slightly above center

	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}

	// Place modal over background (positioned higher)
	return lipgloss.Place(
		m.width,
		m.height-15, // Account for title and status bar, push modal up
		lipgloss.Center,
		lipgloss.Top,
		modal,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(styles.Dim),
	)
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
	helpText := "Navigate: ↑↓/jk | Select: enter | Add: a | Edit: e | Delete: d | Credentials: c | Quit: q"

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
	case ViewSelectionMenu:
		return "Select Mode"
	case ViewCredentials:
		return "Credentials"
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

// GetConnectionType returns whether user selected SSH (0) or SFTP (1)
func (m Model) GetConnectionType() int {
	return m.menuSelection
}
