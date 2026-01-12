package views

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/steevenmentech/bifrost/internal/config"
	"github.com/steevenmentech/bifrost/internal/keyring"
	"github.com/steevenmentech/bifrost/internal/tui/keys"
	"github.com/steevenmentech/bifrost/internal/tui/styles"
)

// CredentialsManagerModel manages the credentials list view
type CredentialsManagerModel struct {
	config        *config.Config
	keys          keys.KeyMap
	selectedIndex int
	err           error
	width         int
	height        int

	// Form for add/edit
	form *CredentialFormModel

	// Confirmation modal
	confirmationModal   *ConfirmationModalModel
	showingConfirmation bool

	// State
	showingForm bool
	formMode    FormMode
	done        bool
}

// NewCredentialsManager creates a new credentials manager
func NewCredentialsManager(cfg *config.Config, keyMap keys.KeyMap) *CredentialsManagerModel {
	return &CredentialsManagerModel{
		config: cfg,
		keys:   keyMap,
	}
}

// Init initializes the model
func (m *CredentialsManagerModel) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (m *CredentialsManagerModel) Update(msg tea.Msg) (*CredentialsManagerModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		// If showing confirmation modal, handle it first
		if m.showingConfirmation && m.confirmationModal != nil {
			return m.updateConfirmationModal(msg)
		}

		// If showing form, delegate to form
		if m.showingForm && m.form != nil {
			return m.updateForm(msg)
		}

		switch {
		case key.Matches(msg, m.keys.Quit):
			m.done = true
			return m, nil

		case key.Matches(msg, m.keys.Back):
			m.done = true
			return m, nil

		case key.Matches(msg, m.keys.Up):
			if m.selectedIndex > 0 {
				m.selectedIndex--
			}
			return m, nil

		case key.Matches(msg, m.keys.Down):
			if m.selectedIndex < len(m.config.Credentials)-1 {
				m.selectedIndex++
			}
			return m, nil

		case key.Matches(msg, m.keys.Add):
			return m.showAddForm()

		case key.Matches(msg, m.keys.Edit):
			if len(m.config.Credentials) > 0 {
				return m.showEditForm()
			}
			return m, nil

		case key.Matches(msg, m.keys.Delete):
			if len(m.config.Credentials) > 0 {
				return m.showDeleteConfirmation()
			}
			return m, nil
		}
	}

	return m, nil
}

// updateForm handles form updates
func (m *CredentialsManagerModel) updateForm(msg tea.KeyMsg) (*CredentialsManagerModel, tea.Cmd) {
	updatedForm, cmd := m.form.Update(msg)
	m.form = &updatedForm

	if m.form.IsSubmitted() {
		return m.handleFormSubmit()
	}

	if m.form.IsCancelled() {
		m.showingForm = false
		m.form = nil
		return m, nil
	}

	return m, cmd
}

// showAddForm shows the credential add form
func (m *CredentialsManagerModel) showAddForm() (*CredentialsManagerModel, tea.Cmd) {
	form := NewCredentialForm(FormModeAdd, nil)
	m.form = &form
	m.showingForm = true
	m.formMode = FormModeAdd
	return m, form.Init()
}

// showEditForm shows the credential edit form
func (m *CredentialsManagerModel) showEditForm() (*CredentialsManagerModel, tea.Cmd) {
	if m.selectedIndex >= len(m.config.Credentials) {
		return m, nil
	}

	cred := &m.config.Credentials[m.selectedIndex]
	form := NewCredentialForm(FormModeEdit, cred)
	m.form = &form
	m.showingForm = true
	m.formMode = FormModeEdit
	return m, form.Init()
}

// handleFormSubmit handles form submission
func (m *CredentialsManagerModel) handleFormSubmit() (*CredentialsManagerModel, tea.Cmd) {
	if m.form == nil {
		return m, nil
	}

	cred := m.form.GetCredential()
	password := m.form.GetPassword()

	// Save password to keyring
	if password != "" {
		err := keyring.SetCredentialPassword(cred.ID, password)
		if err != nil {
			m.err = fmt.Errorf("failed to save password to keyring: %w", err)
			m.showingForm = false
			m.form = nil
			return m, nil
		}
	} else if m.formMode == FormModeAdd {
		// Password is required for new credentials
		m.err = fmt.Errorf("password is required for new credential")
		return m, nil
	}

	// Add or update credential in config
	var err error
	if m.formMode == FormModeAdd {
		err = m.config.AddCredential(cred)
	} else {
		err = m.config.UpdateCredential(cred)
	}

	if err != nil {
		m.err = fmt.Errorf("failed to save credential: %w", err)
		m.showingForm = false
		m.form = nil
		return m, nil
	}

	// Reload config
	cfg, err := config.Load()
	if err != nil {
		m.err = fmt.Errorf("failed to reload config: %w", err)
	} else {
		m.config = cfg
	}

	m.showingForm = false
	m.form = nil
	m.err = nil

	return m, nil
}

// deleteCredential deletes the selected credential
func (m *CredentialsManagerModel) deleteCredential() (*CredentialsManagerModel, tea.Cmd) {
	if m.selectedIndex >= len(m.config.Credentials) {
		return m, nil
	}

	cred := m.config.Credentials[m.selectedIndex]

	// Delete password from keyring
	_ = keyring.DeleteCredentialPassword(cred.ID)

	// Delete from config
	err := m.config.DeleteCredential(cred.ID)
	if err != nil {
		m.err = fmt.Errorf("failed to delete credential: %w", err)
		return m, nil
	}

	// Adjust selection
	if m.selectedIndex >= len(m.config.Credentials) && m.selectedIndex > 0 {
		m.selectedIndex--
	}

	m.err = nil
	return m, nil
}

// View renders the credentials manager
func (m *CredentialsManagerModel) View() string {
	// If showing form, render form centered
	if m.showingForm && m.form != nil {
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
	}

	content := m.renderContent()

	// If showing confirmation modal, render it centered
	if m.showingConfirmation && m.confirmationModal != nil {
		return m.renderConfirmationModal(content)
	}

	return content
}

// renderContent renders the main credentials list content
func (m *CredentialsManagerModel) renderContent() string {

	var s string
	s += styles.TitleStyle.Render("🔑 Credentials Manager") + "\n\n"

	if len(m.config.Credentials) == 0 {
		s += styles.SubtleStyle.Render("  No credentials yet. Press 'a' to add one.") + "\n"
	} else {
		s += "  Saved Credentials:\n\n"

		for i, cred := range m.config.Credentials {
			line := fmt.Sprintf("  🔐  %s", cred.Label)
			if cred.Username != "" {
				line += styles.SubtleStyle.Render(fmt.Sprintf("  (%s)", cred.Username))
			}

			if i == m.selectedIndex {
				line = styles.SelectedStyle.Render(line)
			} else {
				line = styles.ItemStyle.Render(line)
			}

			s += line + "\n"
		}
	}

	// Show error if any
	if m.err != nil {
		s += "\n" + styles.ErrorStyle.Render(fmt.Sprintf("  Error: %v", m.err)) + "\n"
	}

	// Help text
	s += "\n\n"
	s += styles.HelpStyle.Render("  Navigate: ↑↓/jk | Add: a | Edit: e | Delete: d | Back: esc")

	return s
}

// renderConfirmationModal renders the confirmation modal centered over base content
func (m *CredentialsManagerModel) renderConfirmationModal(baseContent string) string {
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

// IsDone returns whether the user wants to exit
func (m *CredentialsManagerModel) IsDone() bool {
	return m.done
}

// GetConfig returns the current config
func (m *CredentialsManagerModel) GetConfig() *config.Config {
	return m.config
}

// SetSize sets the width and height of the view
func (m *CredentialsManagerModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func (m *CredentialsManagerModel) showDeleteConfirmation() (*CredentialsManagerModel, tea.Cmd) {
	cred := m.config.Credentials[m.selectedIndex]
	message := fmt.Sprintf("Are you sure you want to delete the credential '%s'?", cred.Label)
	modal := NewConfirmationModal("Delete Credential", message)
	m.confirmationModal = &modal
	m.showingConfirmation = true
	return m, m.confirmationModal.Init()
}

func (m *CredentialsManagerModel) updateConfirmationModal(msg tea.KeyMsg) (*CredentialsManagerModel, tea.Cmd) {
	updatedModal, cmd := m.confirmationModal.Update(msg)
	m.confirmationModal = &updatedModal

	if m.confirmationModal.IsConfirmed() {
		// User confirmed deletion
		m.showingConfirmation = false
		m.confirmationModal = nil
		return m.deleteCredential()
	}

	// If cancelled, just hide the modal
	if msg.String() == "esc" || m.confirmationModal.cancelled {
		m.showingConfirmation = false
		m.confirmationModal = nil
		return m, nil
	}

	return m, cmd
}
