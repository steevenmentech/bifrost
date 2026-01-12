package views

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/steevenmentech/bifrost/internal/tui/styles"
)

// ConfirmationModalModel represents a confirmation dialog
type ConfirmationModalModel struct {
	title    string
	message  string
	selected int // 0=Yes, 1=No

	confirmed bool
	cancelled bool
}

// NewConfirmationModal creates a new confirmation modal
func NewConfirmationModal(title, message string) ConfirmationModalModel {
	return ConfirmationModalModel{
		title:    title,
		message:  message,
		selected: 1, // Default to "No" for safety
	}
}

// Init initializes the modal
func (m ConfirmationModalModel) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (m ConfirmationModalModel) Update(msg tea.Msg) (ConfirmationModalModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.cancelled = true
			return m, nil

		case "left", "h":
			if m.selected > 0 {
				m.selected--
			}
			return m, nil

		case "right", "l":
			if m.selected < 1 {
				m.selected++
			}
			return m, nil

		case "enter":
			if m.selected == 0 {
				m.confirmed = true
			} else {
				m.cancelled = true
			}
			return m, nil
		}
	}

	return m, nil
}

// View renders the modal
func (m ConfirmationModalModel) View() string {
	// Title
	titleText := styles.ModalTitleStyle.Render(m.title)

	// Message
	messageText := styles.SubtleStyle.Render(m.message)

	// Buttons
	yesText := " Yes "
	noText := " No "

	if m.selected == 0 {
		yesText = styles.SelectedStyle.Render(yesText)
		noText = styles.ItemStyle.Render(noText)
	} else {
		yesText = styles.ItemStyle.Render(yesText)
		noText = styles.SelectedStyle.Render(noText)
	}

	buttons := lipgloss.JoinHorizontal(lipgloss.Left, yesText, "  ", noText)

	// Help text
	help := styles.SubtleStyle.Render("←→ select • enter confirm • esc cancel")

	// Combine all parts
	content := lipgloss.JoinVertical(lipgloss.Left,
		titleText,
		"",
		messageText,
		"",
		buttons,
		"",
		help,
	)

	return styles.ModalStyle.Render(content)
}

// IsConfirmed returns whether user confirmed
func (m ConfirmationModalModel) IsConfirmed() bool {
	return m.confirmed
}

// IsCancelled returns whether user cancelled
func (m ConfirmationModalModel) IsCancelled() bool {
	return m.cancelled
}
