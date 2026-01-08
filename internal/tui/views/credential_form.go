package views

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
	"github.com/steevenmentech/bifrost/internal/config"
	"github.com/steevenmentech/bifrost/internal/keyring"
	"github.com/steevenmentech/bifrost/internal/tui/styles"
)

// CredentialField represents which field is currently focused
type CredentialField int

const (
	CredFieldLabel CredentialField = iota
	CredFieldUsername
	CredFieldPassword
	CredFieldSubmit
	CredFieldCancel
)

// CredentialFormModel is the model for the credential form
type CredentialFormModel struct {
	mode   FormMode
	credID string // Only set when editing

	// Text inputs
	labelInput    textinput.Model
	usernameInput textinput.Model
	passwordInput textinput.Model

	// Form state
	focusIndex int
	inputs     []textinput.Model

	// Results
	submitted bool
	cancelled bool
	err       error
}

// NewCredentialForm creates a new credential form
func NewCredentialForm(mode FormMode, cred *config.Credential) CredentialFormModel {
	m := CredentialFormModel{
		mode: mode,
	}

	// Initialize text inputs
	m.labelInput = textinput.New()
	m.labelInput.Placeholder = "Work Servers"
	m.labelInput.Focus()
	m.labelInput.CharLimit = 50
	m.labelInput.Width = 40

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
		m.usernameInput,
		m.passwordInput,
	}

	// If editing, populate with existing values
	if mode == FormModeEdit && cred != nil {
		m.credID = cred.ID
		m.inputs[CredFieldLabel].SetValue(cred.Label)
		m.inputs[CredFieldUsername].SetValue(cred.Username)

		// Load password from keyring
		password, err := keyring.GetCredentialPassword(cred.ID)
		if err == nil {
			m.inputs[CredFieldPassword].SetValue(password)
		}
	}

	return m
}

// Init initializes the form
func (m CredentialFormModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages
func (m CredentialFormModel) Update(msg tea.Msg) (CredentialFormModel, tea.Cmd) {
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

		case "enter":
			if m.focusIndex == int(CredFieldSubmit) {
				return m.submit()
			} else if m.focusIndex == int(CredFieldCancel) {
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
func (m CredentialFormModel) View() string {
	if m.submitted || m.cancelled {
		return ""
	}

	var s string

	// Title
	title := "Add New Credential"
	if m.mode == FormModeEdit {
		title = "Edit Credential"
	}
	s += styles.TitleStyle.Render(title) + "\n\n"

	// Form fields
	s += m.renderField(CredFieldLabel, "Label:", m.inputs[CredFieldLabel].View())
	s += m.renderField(CredFieldUsername, "Username:", m.inputs[CredFieldUsername].View())
	s += m.renderField(CredFieldPassword, "Password:", m.inputs[CredFieldPassword].View())

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
func (m CredentialFormModel) renderField(field CredentialField, label, input string) string {
	fieldStyle := lipgloss.NewStyle().Width(12)
	labelText := fieldStyle.Render(label)

	if m.focusIndex == int(field) {
		labelText = styles.SelectedStyle.Render(label)
	}

	return fmt.Sprintf("  %s %s\n", labelText, input)
}

// renderButtons renders the submit/cancel buttons
func (m CredentialFormModel) renderButtons() string {
	submitText := " Submit "
	cancelText := " Cancel "

	if m.focusIndex == int(CredFieldSubmit) {
		submitText = styles.SelectedStyle.Render(submitText)
	} else {
		submitText = styles.ItemStyle.Render(submitText)
	}

	if m.focusIndex == int(CredFieldCancel) {
		cancelText = styles.SelectedStyle.Render(cancelText)
	} else {
		cancelText = styles.ItemStyle.Render(cancelText)
	}

	return "  " + submitText + "  " + cancelText
}

// nextField moves to the next field
func (m *CredentialFormModel) nextField() {
	if m.focusIndex < len(m.inputs) {
		m.inputs[m.focusIndex].Blur()
	}

	m.focusIndex++
	if m.focusIndex > int(CredFieldCancel) {
		m.focusIndex = 0
	}

	if m.focusIndex < len(m.inputs) {
		m.inputs[m.focusIndex].Focus()
	}
}

// prevField moves to the previous field
func (m *CredentialFormModel) prevField() {
	if m.focusIndex < len(m.inputs) {
		m.inputs[m.focusIndex].Blur()
	}

	m.focusIndex--
	if m.focusIndex < 0 {
		m.focusIndex = int(CredFieldCancel)
	}

	if m.focusIndex < len(m.inputs) {
		m.inputs[m.focusIndex].Focus()
	}
}

// submit validates and submits the form
func (m CredentialFormModel) submit() (CredentialFormModel, tea.Cmd) {
	// Validate
	if m.inputs[CredFieldLabel].Value() == "" {
		m.err = fmt.Errorf("label is required")
		return m, nil
	}
	if m.inputs[CredFieldUsername].Value() == "" {
		m.err = fmt.Errorf("username is required")
		return m, nil
	}

	m.submitted = true
	return m, nil
}

// GetCredential returns the credential from form values
func (m CredentialFormModel) GetCredential() config.Credential {
	credID := m.credID
	if credID == "" {
		credID = uuid.New().String()
	}

	return config.Credential{
		ID:       credID,
		Label:    m.inputs[CredFieldLabel].Value(),
		Username: m.inputs[CredFieldUsername].Value(),
	}
}

// GetPassword returns the password from the form
func (m CredentialFormModel) GetPassword() string {
	return m.inputs[CredFieldPassword].Value()
}

// IsSubmitted returns whether the form was submitted
func (m CredentialFormModel) IsSubmitted() bool {
	return m.submitted
}

// IsCancelled returns whether the form was cancelled
func (m CredentialFormModel) IsCancelled() bool {
	return m.cancelled
}
