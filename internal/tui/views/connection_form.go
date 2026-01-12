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
	FieldAuthType
	FieldCredential
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

	// Auth type selection (0=password, 1=credential)
	authTypeIndex int
	authTypes     []string

	// Credential selection
	credentials     []config.Credential
	credentialIndex int

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
	return NewConnectionFormWithCredentials(mode, conn, nil)
}

// NewConnectionFormWithCredentials creates a new connection form with credentials list
func NewConnectionFormWithCredentials(mode FormMode, conn *config.Connection, credentials []config.Credential) ConnectionFormModel {
	m := ConnectionFormModel{
		mode:        mode,
		icons:       []string{"\uf179", "\uf17c", "\uf17a", "\uf233"},  // Nerd Font: Apple, Linux, Windows, Server
		iconLabels:  []string{"Apple", "Linux", "Windows", "Server"},
		iconIndex:   3,
		authTypes:   []string{"Password", "Credential"},
		credentials: credentials,
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
		m.inputs[0].SetValue(conn.Label)
		m.inputs[1].SetValue(conn.Host)
		m.inputs[2].SetValue(strconv.Itoa(conn.Port))

		// Find icon index
		for i, icon := range m.icons {
			if icon == conn.Icon {
				m.iconIndex = i
				break
			}
		}

		// Set auth type
		if conn.AuthType == "credential" {
			m.authTypeIndex = 1
			// Find credential index
			for i, cred := range m.credentials {
				if cred.ID == conn.CredentialID {
					m.credentialIndex = i
					break
				}
			}
		} else {
			m.authTypeIndex = 0
			m.inputs[3].SetValue(conn.Username)
			// Load password from keyring
			password, err := keyring.GetConnectionPassword(conn.ID)
			if err == nil {
				m.inputs[4].SetValue(password)
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
			if m.focusIndex == int(FieldAuthType) {
				m.prevAuthType()
				return m, nil
			}
			if m.focusIndex == int(FieldCredential) {
				m.prevCredential()
				return m, nil
			}

		case "right":
			if m.focusIndex == int(FieldIcon) {
				m.nextIcon()
				return m, nil
			}
			if m.focusIndex == int(FieldAuthType) {
				m.nextAuthType()
				return m, nil
			}
			if m.focusIndex == int(FieldCredential) {
				m.nextCredential()
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

	// Update the focused input (only for text inputs)
	if m.isTextInputField(m.focusIndex) {
		inputIdx := m.getInputIndex(m.focusIndex)
		if inputIdx >= 0 && inputIdx < len(m.inputs) {
			var cmd tea.Cmd
			m.inputs[inputIdx], cmd = m.inputs[inputIdx].Update(msg)
			return m, cmd
		}
	}

	return m, nil
}

// isTextInputField returns whether the field is a text input
func (m ConnectionFormModel) isTextInputField(field int) bool {
	switch FormField(field) {
	case FieldLabel, FieldHost, FieldPort:
		return true
	case FieldUsername, FieldPassword:
		return m.authTypeIndex == 0 // Only when using password auth
	default:
		return false
	}
}

// getInputIndex returns the index in the inputs slice for a field
func (m ConnectionFormModel) getInputIndex(field int) int {
	switch FormField(field) {
	case FieldLabel:
		return 0
	case FieldHost:
		return 1
	case FieldPort:
		return 2
	case FieldUsername:
		return 3
	case FieldPassword:
		return 4
	default:
		return -1
	}
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
	s += m.renderField(FieldLabel, "Label:", m.inputs[0].View())
	s += m.renderField(FieldHost, "Host:", m.inputs[1].View())
	s += m.renderField(FieldPort, "Port:", m.inputs[2].View())
	s += m.renderAuthTypeField()

	// Show credential selector or username/password based on auth type
	if m.authTypeIndex == 1 {
		// Credential mode
		s += m.renderCredentialField()
	} else {
		// Password mode
		s += m.renderField(FieldUsername, "Username:", m.inputs[3].View())
		s += m.renderField(FieldPassword, "Password:", m.inputs[4].View())
	}

	s += m.renderIconField()

	// Buttons
	s += "\n\n"
	s += m.renderButtons()

	// Help text
	s += "\n\n"
	s += styles.HelpStyle.Render("tab/↑↓ navigate • ←→ select • enter submit • esc cancel")

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

// renderAuthTypeField renders the auth type selection field
func (m ConnectionFormModel) renderAuthTypeField() string {
	label := "Auth:"
	if m.focusIndex == int(FieldAuthType) {
		label = styles.SelectedStyle.Render(label)
	} else {
		label = lipgloss.NewStyle().Width(12).Render(label)
	}

	var types string
	for i, authType := range m.authTypes {
		text := fmt.Sprintf(" %s ", authType)
		if i == m.authTypeIndex {
			text = styles.SelectedStyle.Render(text)
		} else {
			text = styles.ItemStyle.Render(text)
		}
		types += text
	}

	return fmt.Sprintf("  %s %s\n", label, types)
}

// renderCredentialField renders the credential selection field
func (m ConnectionFormModel) renderCredentialField() string {
	label := "Credential:"
	if m.focusIndex == int(FieldCredential) {
		label = styles.SelectedStyle.Render(label)
	} else {
		label = lipgloss.NewStyle().Width(12).Render(label)
	}

	if len(m.credentials) == 0 {
		return fmt.Sprintf("  %s %s\n", label, styles.SubtleStyle.Render("(no credentials - press 'c' to add)"))
	}

	cred := m.credentials[m.credentialIndex]
	credText := fmt.Sprintf("← %s (%s) →", cred.Label, cred.Username)
	if m.focusIndex == int(FieldCredential) {
		credText = styles.SelectedStyle.Render(credText)
	}

	return fmt.Sprintf("  %s %s\n", label, credText)
}

// renderIconField renders the icon selection field
func (m ConnectionFormModel) renderIconField() string {
	label := "Icon:"
	if m.focusIndex == int(FieldIcon) {
		label = styles.SelectedStyle.Render(label)
	} else {
		label = lipgloss.NewStyle().Width(12).Render(label)
	}

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
	// Blur current text input if applicable
	if m.isTextInputField(m.focusIndex) {
		inputIdx := m.getInputIndex(m.focusIndex)
		if inputIdx >= 0 {
			m.inputs[inputIdx].Blur()
		}
	}

	m.focusIndex++

	// Skip credential field if using password auth
	if m.focusIndex == int(FieldCredential) && m.authTypeIndex == 0 {
		m.focusIndex++
	}

	// Skip username/password fields if using credential auth
	if m.authTypeIndex == 1 {
		if m.focusIndex == int(FieldUsername) || m.focusIndex == int(FieldPassword) {
			m.focusIndex = int(FieldIcon)
		}
	}

	if m.focusIndex > int(FieldCancel) {
		m.focusIndex = 0
	}

	// Focus new text input if applicable
	if m.isTextInputField(m.focusIndex) {
		inputIdx := m.getInputIndex(m.focusIndex)
		if inputIdx >= 0 {
			m.inputs[inputIdx].Focus()
		}
	}
}

// prevField moves to the previous field
func (m *ConnectionFormModel) prevField() {
	// Blur current text input if applicable
	if m.isTextInputField(m.focusIndex) {
		inputIdx := m.getInputIndex(m.focusIndex)
		if inputIdx >= 0 {
			m.inputs[inputIdx].Blur()
		}
	}

	m.focusIndex--

	// Skip username/password fields if using credential auth
	if m.authTypeIndex == 1 {
		if m.focusIndex == int(FieldPassword) || m.focusIndex == int(FieldUsername) {
			m.focusIndex = int(FieldCredential)
		}
	}

	// Skip credential field if using password auth
	if m.focusIndex == int(FieldCredential) && m.authTypeIndex == 0 {
		m.focusIndex--
	}

	if m.focusIndex < 0 {
		m.focusIndex = int(FieldCancel)
	}

	// Focus new text input if applicable
	if m.isTextInputField(m.focusIndex) {
		inputIdx := m.getInputIndex(m.focusIndex)
		if inputIdx >= 0 {
			m.inputs[inputIdx].Focus()
		}
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

// nextAuthType cycles to the next auth type
func (m *ConnectionFormModel) nextAuthType() {
	m.authTypeIndex++
	if m.authTypeIndex >= len(m.authTypes) {
		m.authTypeIndex = 0
	}
}

// prevAuthType cycles to the previous auth type
func (m *ConnectionFormModel) prevAuthType() {
	m.authTypeIndex--
	if m.authTypeIndex < 0 {
		m.authTypeIndex = len(m.authTypes) - 1
	}
}

// nextCredential cycles to the next credential
func (m *ConnectionFormModel) nextCredential() {
	if len(m.credentials) == 0 {
		return
	}
	m.credentialIndex++
	if m.credentialIndex >= len(m.credentials) {
		m.credentialIndex = 0
	}
}

// prevCredential cycles to the previous credential
func (m *ConnectionFormModel) prevCredential() {
	if len(m.credentials) == 0 {
		return
	}
	m.credentialIndex--
	if m.credentialIndex < 0 {
		m.credentialIndex = len(m.credentials) - 1
	}
}

// submit validates and submits the form
func (m ConnectionFormModel) submit() (ConnectionFormModel, tea.Cmd) {
	// Validate
	if m.inputs[0].Value() == "" {
		m.err = fmt.Errorf("label is required")
		return m, nil
	}
	if m.inputs[1].Value() == "" {
		m.err = fmt.Errorf("host is required")
		return m, nil
	}

	// Parse port
	port := 22
	if m.inputs[2].Value() != "" {
		var err error
		port, err = strconv.Atoi(m.inputs[2].Value())
		if err != nil || port < 1 || port > 65535 {
			m.err = fmt.Errorf("invalid port number")
			return m, nil
		}
	}

	// Validate credential selection
	if m.authTypeIndex == 1 && len(m.credentials) == 0 {
		m.err = fmt.Errorf("no credentials available - create one first with 'c'")
		return m, nil
	}

	m.submitted = true
	return m, nil
}

// GetConnection returns the connection from form values
func (m ConnectionFormModel) GetConnection() config.Connection {
	port := 22
	if m.inputs[2].Value() != "" {
		port, _ = strconv.Atoi(m.inputs[2].Value())
	}

	connID := m.connID
	if connID == "" {
		connID = uuid.New().String()
	}

	conn := config.Connection{
		ID:    connID,
		Label: m.inputs[0].Value(),
		Host:  m.inputs[1].Value(),
		Port:  port,
		Icon:  m.icons[m.iconIndex],
	}

	if m.authTypeIndex == 1 && len(m.credentials) > 0 {
		// Using credential
		conn.AuthType = "credential"
		conn.CredentialID = m.credentials[m.credentialIndex].ID
		conn.Username = m.credentials[m.credentialIndex].Username
	} else {
		// Using password
		conn.AuthType = "password"
		conn.Username = m.inputs[3].Value()
	}

	return conn
}

// GetPassword returns the password from the form
func (m ConnectionFormModel) GetPassword() string {
	if m.authTypeIndex == 1 {
		// Using credential - no password to return
		return ""
	}
	return m.inputs[4].Value()
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

// IsUsingCredential returns whether the form is set to use credential auth
func (m ConnectionFormModel) IsUsingCredential() bool {
	return m.authTypeIndex == 1
}
