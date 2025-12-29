package views

import (
	"fmt"
	"strconv"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
	"github.com/steevenmentech/bifrost/internal/config"
	"github.com/steevenmentech/bifrost/internal/keyring"
	"github.com/steevenmentech/bifrost/internal/tui/styles"
)

// FormMode represents whether we're adding or editing
type FormMode int

const (
	FormModeAdd FormMode = iota
	FormModeEdit
)

// FormField represents which field is currently focused
type FormField int

const (
	FieldLabel FormField = iota
	FieldHost
	FieldPort
	FieldUsername
	FieldPassword
	FieldIcon
	FieldSubmit
	FieldCancel
)

// ConnectionFormModel is the model for the connection form
type ConnectionFormModel struct {
	mode   FormMode
	connID string // Only set when editing

	// Text inputs
	labelInput    textinput.Model
	hostInput     textinput.Model
	portInput     textinput.Model
	usernameInput textinput.Model
	passwordInput textinput.Model

	// Icon selection
	iconIndex  int
	icons      []string
	iconLabels []string

	// Form state
	focusIndex int
	inputs     []textinput.Model

	// Results
	submitted bool
	cancelled bool
	err       error
}

// NewConnectionForm creates a new connection form
func NewConnectionForm(mode FormMode, conn *config.Connection) ConnectionFormModel {
	m := ConnectionFormModel{
		mode:       mode,
		icons:      []string{"", "", "", "🖥️"},
		iconLabels: []string{"Apple", "Linux", "Windows", "Server"},
		iconIndex:  3, // Default to server icon
	}

	// Initialize text inputs
	m.labelInput = textinput.New()
	m.labelInput.Placeholder = "Production Server"
	m.labelInput.Focus()
	m.labelInput.CharLimit = 50
	m.labelInput.Width = 40

	m.hostInput = textinput.New()
	m.hostInput.Placeholder = "192.168.1.100 or example.com"
	m.hostInput.CharLimit = 100
	m.hostInput.Width = 40

	m.portInput = textinput.New()
	m.portInput.Placeholder = "22"
	m.portInput.CharLimit = 5
	m.portInput.Width = 10

	m.usernameInput = textinput.New()
	m.usernameInput.Placeholder = "admin"
	m.usernameInput.CharLimit = 50
	m.usernameInput.Width = 40

	m.passwordInput = textinput.New()
	m.passwordInput.Placeholder = "password"
	m.passwordInput.EchoMode = textinput.EchoPassword
	m.passwordInput.EchoCharacter = '•'
	m.passwordInput.CharLimit = 100
	m.passwordInput.Width = 40

	m.inputs = []textinput.Model{
		m.labelInput,
		m.hostInput,
		m.portInput,
		m.usernameInput,
		m.passwordInput,
	}

	// If editing, populate with existing values
	if mode == FormModeEdit && conn != nil {
		m.connID = conn.ID
		m.inputs[FieldLabel].SetValue(conn.Label)
		m.inputs[FieldHost].SetValue(conn.Host)
		m.inputs[FieldPort].SetValue(strconv.Itoa(conn.Port))
		m.inputs[FieldUsername].SetValue(conn.Username)

		// Find icon index
		for i, icon := range m.icons {
			if icon == conn.Icon {
				m.iconIndex = i
				break
			}
		}

		// Load password from keyring
		if conn.AuthType == "password" {
			password, err := keyring.GetConnectionPassword(conn.ID)
			if err == nil {
				m.inputs[FieldPassword].SetValue(password)
			}
		}
	}

	return m
}

// Init initializes the form
func (m ConnectionFormModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages
func (m ConnectionFormModel) Update(msg tea.Msg) (ConnectionFormModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.cancelled = true
			return m, nil

		case "tab", "down":
			m.nextField()
			return m, nil

		case "shift+tab", "up":
			m.prevField()
			return m, nil

		case "left":
			if m.focusIndex == int(FieldIcon) {
				m.prevIcon()
				return m, nil
			}

		case "right":
			if m.focusIndex == int(FieldIcon) {
				m.nextIcon()
				return m, nil
			}

		case "enter":
			if m.focusIndex == int(FieldSubmit) {
				return m.submit()
			} else if m.focusIndex == int(FieldCancel) {
				m.cancelled = true
				return m, nil
			} else {
				m.nextField()
				return m, nil
			}
		}
	}

	// Update the focused input
	if m.focusIndex < len(m.inputs) {
		var cmd tea.Cmd
		m.inputs[m.focusIndex], cmd = m.inputs[m.focusIndex].Update(msg)
		return m, cmd
	}

	return m, nil
}

// View renders the form
func (m ConnectionFormModel) View() string {
	if m.submitted || m.cancelled {
		return ""
	}

	var s string

	// Title
	title := "Add New Connection"
	if m.mode == FormModeEdit {
		title = "Edit Connection"
	}
	s += styles.TitleStyle.Render(title) + "\n\n"

	// Form fields
	s += m.renderField(FieldLabel, "Label:", m.inputs[FieldLabel].View())
	s += m.renderField(FieldHost, "Host:", m.inputs[FieldHost].View())
	s += m.renderField(FieldPort, "Port:", m.inputs[FieldPort].View())
	s += m.renderField(FieldUsername, "Username:", m.inputs[FieldUsername].View())
	s += m.renderField(FieldPassword, "Password:", m.inputs[FieldPassword].View())
	s += m.renderIconField()

	// Buttons
	s += "\n\n"
	s += m.renderButtons()

	// Help text
	s += "\n\n"
	s += styles.HelpStyle.Render("tab/shift+tab navigate • enter submit • esc cancel")

	// Error message
	if m.err != nil {
		s += "\n\n"
		s += styles.ErrorStyle.Render(fmt.Sprintf("Error: %v", m.err))
	}

	return styles.BorderStyle.Render(s)
}

// renderField renders a form field
func (m ConnectionFormModel) renderField(field FormField, label, input string) string {
	fieldStyle := lipgloss.NewStyle().Width(12)
	labelText := fieldStyle.Render(label)

	if m.focusIndex == int(field) {
		labelText = styles.SelectedStyle.Render(label)
	}

	return fmt.Sprintf("  %s %s\n", labelText, input)
}

// renderIconField renders the icon selection field
func (m ConnectionFormModel) renderIconField() string {
	label := "Icon:"
	if m.focusIndex == int(FieldIcon) {
		label = styles.SelectedStyle.Render(label)
	} else {
		label = lipgloss.NewStyle().Width(12).Render(label)
	}

	// Show all icons with current one highlighted
	var icons string
	for i, icon := range m.icons {
		iconText := fmt.Sprintf(" %s %s ", icon, m.iconLabels[i])
		if i == m.iconIndex {
			iconText = styles.SelectedStyle.Render(iconText)
		} else {
			iconText = styles.ItemStyle.Render(iconText)
		}
		icons += iconText
	}

	return fmt.Sprintf("  %s %s\n", label, icons)
}

// renderButtons renders the submit/cancel buttons
func (m ConnectionFormModel) renderButtons() string {
	submitText := " Submit "
	cancelText := " Cancel "

	if m.focusIndex == int(FieldSubmit) {
		submitText = styles.SelectedStyle.Render(submitText)
	} else {
		submitText = styles.ItemStyle.Render(submitText)
	}

	if m.focusIndex == int(FieldCancel) {
		cancelText = styles.SelectedStyle.Render(cancelText)
	} else {
		cancelText = styles.ItemStyle.Render(cancelText)
	}

	return "  " + submitText + "  " + cancelText
}

// nextField moves to the next field
func (m *ConnectionFormModel) nextField() {
	// Only blur if current focus is on a text input
	if m.focusIndex < len(m.inputs) {
		m.inputs[m.focusIndex].Blur()
	}

	m.focusIndex++
	if m.focusIndex > int(FieldCancel) {
		m.focusIndex = 0
	}

	// Only focus if new focus is on a text input
	if m.focusIndex < len(m.inputs) {
		m.inputs[m.focusIndex].Focus()
	}
}

// prevField moves to the previous field
func (m *ConnectionFormModel) prevField() {
	// Only blur if current focus is on a text input
	if m.focusIndex < len(m.inputs) {
		m.inputs[m.focusIndex].Blur()
	}

	m.focusIndex--
	if m.focusIndex < 0 {
		m.focusIndex = int(FieldCancel)
	}

	// Only focus if new focus is on a text input
	if m.focusIndex < len(m.inputs) {
		m.inputs[m.focusIndex].Focus()
	}
}

// nextIcon cycles to the next icon
func (m *ConnectionFormModel) nextIcon() {
	m.iconIndex++
	if m.iconIndex >= len(m.icons) {
		m.iconIndex = 0
	}
}

// prevIcon cycles to the previous icon
func (m *ConnectionFormModel) prevIcon() {
	m.iconIndex--
	if m.iconIndex < 0 {
		m.iconIndex = len(m.icons) - 1
	}
}

// submit validates and submits the form
func (m ConnectionFormModel) submit() (ConnectionFormModel, tea.Cmd) {
	// Validate
	if m.inputs[FieldLabel].Value() == "" {
		m.err = fmt.Errorf("label is required")
		return m, nil
	}
	if m.inputs[FieldHost].Value() == "" {
		m.err = fmt.Errorf("host is required")
		return m, nil
	}

	// Parse port
	port := 22
	if m.inputs[FieldPort].Value() != "" {
		var err error
		port, err = strconv.Atoi(m.inputs[FieldPort].Value())
		if err != nil || port < 1 || port > 65535 {
			m.err = fmt.Errorf("invalid port number")
			return m, nil
		}
	}

	m.submitted = true
	return m, nil
}

// GetConnection returns the connection from form values
func (m ConnectionFormModel) GetConnection() config.Connection {
	port := 22
	if m.inputs[FieldPort].Value() != "" {
		port, _ = strconv.Atoi(m.inputs[FieldPort].Value())
	}

	connID := m.connID
	if connID == "" {
		connID = uuid.New().String()
	}

	return config.Connection{
		ID:       connID,
		Label:    m.inputs[FieldLabel].Value(),
		Host:     m.inputs[FieldHost].Value(),
		Port:     port,
		Username: m.inputs[FieldUsername].Value(),
		Icon:     m.icons[m.iconIndex],
		AuthType: "password",
	}
}

// GetPassword returns the password from the form
func (m ConnectionFormModel) GetPassword() string {
	return m.inputs[FieldPassword].Value()
}

// IsSubmitted returns whether the form was submitted
func (m ConnectionFormModel) IsSubmitted() bool {
	return m.submitted
}

// IsCancelled returns whether the form was cancelled
func (m ConnectionFormModel) IsCancelled() bool {
	return m.cancelled
}

// GetMode returns the form mode (add or edit)
func (m ConnectionFormModel) GetMode() FormMode {
	return m.mode
}
