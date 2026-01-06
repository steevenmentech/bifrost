package views

import (
	"fmt"
	"path"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/steevenmentech/bifrost/internal/sftp"
	"github.com/steevenmentech/bifrost/internal/tui/keys"
	"github.com/steevenmentech/bifrost/internal/tui/styles"
)

type SFTPBrowserState int

const (
	BrowsingState SFTPBrowserState = iota
	GoToPathState
)

type SFTPBrowserModel struct {
	client        *sftp.Client
	files         []sftp.FileInfo
	selectedIndex int
	currentPath   string
	state         SFTPBrowserState
	pathInput     textinput.Model
	showHidden    bool
	err           error
	width         int
	height        int
	keys          keys.KeyMap
}

func NewSFTPBrowser(client *sftp.Client, keymap keys.KeyMap) *SFTPBrowserModel {
	ti := textinput.New()
	ti.Placeholder = "/path/to/directory"
	ti.CharLimit = 256
	ti.Width = 50

	browser := &SFTPBrowserModel{
		client:        client,
		selectedIndex: 0,
		currentPath:   client.GetWorkingDir(),
		state:         BrowsingState,
		pathInput:     ti,
		showHidden:    false,
		keys:          keymap,
	}

	// Load initial directory
	browser.loadCurrentDirectory()
	return browser
}

func (m *SFTPBrowserModel) loadCurrentDirectory() {
	files, err := m.client.ListDir(m.currentPath)
	if err != nil {
		m.err = err
		return
	}

	// Filter hidden files if needed
	if !m.showHidden {
		filtered := make([]sftp.FileInfo, 0)
		for _, f := range files {
			if !strings.HasPrefix(f.Name, ".") {
				filtered = append(filtered, f)
			}
		}
		files = filtered
	}

	m.files = files
	m.err = nil

	// Reset selection if out of bounds
	if m.selectedIndex >= len(m.files) {
		m.selectedIndex = 0
	}
}

func (m *SFTPBrowserModel) Init() tea.Cmd {
	return nil
}

func (m *SFTPBrowserModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.state {
	case GoToPathState:
		return m.updateGoToPath(msg)
	default:
		return m.updateBrowsing(msg)
	}
}

func (m *SFTPBrowserModel) updateBrowsing(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Up):
			if m.selectedIndex > 0 {
				m.selectedIndex--
			}
		case key.Matches(msg, m.keys.Down):
			if m.selectedIndex < len(m.files)-1 {
				m.selectedIndex++
			}
		case key.Matches(msg, m.keys.Left): // h - parent directory
			m.goToParentDirectory()
		case key.Matches(msg, m.keys.Right), msg.String() == "enter": // l or Enter
			m.enterSelected()
		case msg.String() == "g": // Go to path
			m.state = GoToPathState
			m.pathInput.SetValue(m.currentPath)
			m.pathInput.Focus()
			return m, textinput.Blink
		case msg.String() == "~": // Go to home
			m.currentPath = "~"
			m.loadCurrentDirectory()
		case msg.String() == ".": // Toggle hidden files
			m.showHidden = !m.showHidden
			m.loadCurrentDirectory()
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	return m, nil
}

func (m *SFTPBrowserModel) updateGoToPath(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			newPath := m.pathInput.Value()
			if newPath != "" {
				// Try to change to this directory
				err := m.client.ChangeDir(newPath)
				if err == nil {
					m.currentPath = m.client.GetWorkingDir()
					m.loadCurrentDirectory()
				} else {
					m.err = err
				}
			}
			m.state = BrowsingState
			m.pathInput.Blur()
			return m, nil
		case "esc":
			m.state = BrowsingState
			m.pathInput.Blur()
			return m, nil
		}
	}

	m.pathInput, cmd = m.pathInput.Update(msg)
	return m, cmd
}

func (m *SFTPBrowserModel) goToParentDirectory() {
	if m.currentPath == "/" {
		return
	}
	parentPath := path.Dir(m.currentPath)
	err := m.client.ChangeDir(parentPath)
	if err == nil {
		m.currentPath = m.client.GetWorkingDir()
		m.loadCurrentDirectory()
	} else {
		m.err = err
	}
}

func (m *SFTPBrowserModel) enterSelected() {
	if len(m.files) == 0 || m.selectedIndex >= len(m.files) {
		return
	}

	selected := m.files[m.selectedIndex]
	if selected.IsDir {
		// Enter directory
		newPath := path.Join(m.currentPath, selected.Name)
		err := m.client.ChangeDir(newPath)
		if err == nil {
			m.currentPath = m.client.GetWorkingDir()
			m.loadCurrentDirectory()
		} else {
			m.err = err
		}
	}
	// For files, we'll handle download/edit in PYC-23
}

func (m *SFTPBrowserModel) View() string {
	if m.state == GoToPathState {
		return m.viewGoToPath()
	}
	return m.viewBrowsing()
}

func (m *SFTPBrowserModel) viewGoToPath() string {
	var s strings.Builder
	s.WriteString("\n")
	s.WriteString(styles.TitleStyle.Render("  Go to path") + "\n\n")
	s.WriteString("  " + m.pathInput.View() + "\n\n")
	s.WriteString(styles.SubtleStyle.Render("  enter to navigate • esc to cancel") + "\n")
	return s.String()
}

func (m *SFTPBrowserModel) viewBrowsing() string {
	var s strings.Builder

	// Title and breadcrumb
	s.WriteString("\n")
	s.WriteString(styles.TitleStyle.Render("  SFTP Browser") + "\n")
	s.WriteString(styles.SubtleStyle.Render("  "+m.renderBreadcrumb()) + "\n\n")

	// Error display
	if m.err != nil {
		s.WriteString(styles.ErrorStyle.Render("  Error: "+m.err.Error()) + "\n\n")
	}

	// File list
	if len(m.files) == 0 {
		s.WriteString(styles.SubtleStyle.Render("  (empty directory)") + "\n")
	} else {
		for i, file := range m.files {
			line := m.renderFileLine(file, i == m.selectedIndex)
			s.WriteString(line + "\n")
		}
	}

	// Help text
	s.WriteString("\n")
	helpText := "  j/k: up/down • h: parent • l/enter: open • g: go to path • ~: home • .: toggle hidden • q: quit"
	s.WriteString(styles.SubtleStyle.Render(helpText) + "\n")

	return s.String()
}

func (m *SFTPBrowserModel) renderBreadcrumb() string {
	parts := strings.Split(m.currentPath, "/")
	if len(parts) == 0 || (len(parts) == 1 && parts[0] == "") {
		return "/"
	}
	return strings.Join(parts, " / ")
}

func (m *SFTPBrowserModel) renderFileLine(file sftp.FileInfo, isSelected bool) string {
	// Icon
	icon := "📄"
	if file.IsDir {
		icon = "📁"
	}

	// Format size
	sizeStr := formatFileSize(file.Size)
	if file.IsDir {
		sizeStr = "-"
	}

	// Format line
	line := fmt.Sprintf("%s  %-40s  %10s  %s",
		icon,
		file.Name,
		sizeStr,
		file.Permissions,
	)

	if isSelected {
		return styles.SelectedStyle.Render("  " + line)
	}
	return styles.ItemStyle.Render("  " + line)
}

func formatFileSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}
