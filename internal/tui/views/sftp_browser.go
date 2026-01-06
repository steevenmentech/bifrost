package views

import (
	"fmt"
	"path"
	"strings"

	"github.com/atotto/clipboard"
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
	CreateFileState
	CreateDirState
	RenameState
	DeleteConfirmState
)

type SFTPBrowserModel struct {
	client        *sftp.Client
	files         []sftp.FileInfo
	selectedIndex int
	currentPath   string
	state         SFTPBrowserState
	pathInput     textinput.Model
	nameInput     textinput.Model
	showHidden    bool
	err           error
	successMsg    string
	width         int
	height        int
	keys          keys.KeyMap
}

func NewSFTPBrowser(client *sftp.Client, keymap keys.KeyMap) *SFTPBrowserModel {
	pathInput := textinput.New()
	pathInput.Placeholder = "/path/to/directory"
	pathInput.CharLimit = 256
	pathInput.Width = 50

	nameInput := textinput.New()
	nameInput.Placeholder = "filename"
	nameInput.CharLimit = 256
	nameInput.Width = 50

	browser := &SFTPBrowserModel{
		client:        client,
		selectedIndex: 0,
		currentPath:   client.GetWorkingDir(),
		state:         BrowsingState,
		pathInput:     pathInput,
		nameInput:     nameInput,
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
	case CreateFileState:
		return m.updateCreateFile(msg)
	case CreateDirState:
		return m.updateCreateDir(msg)
	case RenameState:
		return m.updateRename(msg)
	case DeleteConfirmState:
		return m.updateDeleteConfirm(msg)
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
		case msg.String() == "n": // Create new file
			m.state = CreateFileState
			m.nameInput.SetValue("")
			m.nameInput.Focus()
			return m, textinput.Blink
		case msg.String() == "N": // Create new directory
			m.state = CreateDirState
			m.nameInput.SetValue("")
			m.nameInput.Focus()
			return m, textinput.Blink
		case msg.String() == "d": // Delete
			if len(m.files) > 0 && m.selectedIndex < len(m.files) {
				m.state = DeleteConfirmState
				return m, nil
			}
		case msg.String() == "r": // Rename
			if len(m.files) > 0 && m.selectedIndex < len(m.files) {
				m.state = RenameState
				m.nameInput.SetValue(m.files[m.selectedIndex].Name)
				m.nameInput.Focus()
				return m, textinput.Blink
			}
		case msg.String() == "y": // Copy path to clipboard
			if len(m.files) > 0 && m.selectedIndex < len(m.files) {
				fullPath := path.Join(m.currentPath, m.files[m.selectedIndex].Name)
				m.copyToClipboard(fullPath)
				m.successMsg = "Path copied to clipboard"
			}
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		}

		// Clear messages after showing
		m.err = nil
		m.successMsg = ""

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

func (m *SFTPBrowserModel) updateCreateFile(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			fileName := m.nameInput.Value()
			if fileName != "" {
				filePath := path.Join(m.currentPath, fileName)
				err := m.client.CreateFile(filePath)
				if err != nil {
					m.err = err
				} else {
					m.successMsg = fmt.Sprintf("Created file: %s", fileName)
					m.loadCurrentDirectory()
				}
			}
			m.state = BrowsingState
			m.nameInput.Blur()
			return m, nil
		case "esc":
			m.state = BrowsingState
			m.nameInput.Blur()
			return m, nil
		}
	}

	m.nameInput, cmd = m.nameInput.Update(msg)
	return m, cmd
}

func (m *SFTPBrowserModel) updateCreateDir(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			dirName := m.nameInput.Value()
			if dirName != "" {
				dirPath := path.Join(m.currentPath, dirName)
				err := m.client.CreateDirectory(dirPath)
				if err != nil {
					m.err = err
				} else {
					m.successMsg = fmt.Sprintf("Created directory: %s", dirName)
					m.loadCurrentDirectory()
				}
			}
			m.state = BrowsingState
			m.nameInput.Blur()
			return m, nil
		case "esc":
			m.state = BrowsingState
			m.nameInput.Blur()
			return m, nil
		}
	}

	m.nameInput, cmd = m.nameInput.Update(msg)
	return m, cmd
}

func (m *SFTPBrowserModel) updateRename(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			newName := m.nameInput.Value()
			if newName != "" && m.selectedIndex < len(m.files) {
				oldPath := path.Join(m.currentPath, m.files[m.selectedIndex].Name)
				newPath := path.Join(m.currentPath, newName)
				err := m.client.Rename(oldPath, newPath)
				if err != nil {
					m.err = err
				} else {
					m.successMsg = fmt.Sprintf("Renamed to: %s", newName)
					m.loadCurrentDirectory()
				}
			}
			m.state = BrowsingState
			m.nameInput.Blur()
			return m, nil
		case "esc":
			m.state = BrowsingState
			m.nameInput.Blur()
			return m, nil
		}
	}

	m.nameInput, cmd = m.nameInput.Update(msg)
	return m, cmd
}

func (m *SFTPBrowserModel) updateDeleteConfirm(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "y", "Y": // Confirm delete
			if m.selectedIndex < len(m.files) {
				itemPath := path.Join(m.currentPath, m.files[m.selectedIndex].Name)
				err := m.client.Delete(itemPath)
				if err != nil {
					m.err = err
				} else {
					m.successMsg = fmt.Sprintf("Deleted: %s", m.files[m.selectedIndex].Name)
					m.loadCurrentDirectory()
				}
			}
			m.state = BrowsingState
			return m, nil
		case "n", "N", "esc": // Cancel delete
			m.state = BrowsingState
			return m, nil
		}
	}

	return m, nil
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

func (m *SFTPBrowserModel) copyToClipboard(text string) {
	err := clipboard.WriteAll(text)
	if err != nil {
		m.err = fmt.Errorf("failed to copy to clipboard: %w", err)
	}
}

func (m *SFTPBrowserModel) View() string {
	switch m.state {
	case GoToPathState:
		return m.viewGoToPath()
	case CreateFileState:
		return m.viewCreateFile()
	case CreateDirState:
		return m.viewCreateDir()
	case RenameState:
		return m.viewRename()
	case DeleteConfirmState:
		return m.viewDeleteConfirm()
	default:
		return m.viewBrowsing()
	}
}

func (m *SFTPBrowserModel) viewGoToPath() string {
	var s strings.Builder
	s.WriteString("\n")
	s.WriteString(styles.TitleStyle.Render("  Go to path") + "\n\n")
	s.WriteString("  " + m.pathInput.View() + "\n\n")
	s.WriteString(styles.SubtleStyle.Render("  enter to navigate • esc to cancel") + "\n")
	return s.String()
}

func (m *SFTPBrowserModel) viewCreateFile() string {
	var s strings.Builder
	s.WriteString("\n")
	s.WriteString(styles.TitleStyle.Render("  Create new file") + "\n\n")
	s.WriteString("  " + m.nameInput.View() + "\n\n")
	s.WriteString(styles.SubtleStyle.Render("  enter to create • esc to cancel") + "\n")
	return s.String()
}

func (m *SFTPBrowserModel) viewCreateDir() string {
	var s strings.Builder
	s.WriteString("\n")
	s.WriteString(styles.TitleStyle.Render("  Create new directory") + "\n\n")
	s.WriteString("  " + m.nameInput.View() + "\n\n")
	s.WriteString(styles.SubtleStyle.Render("  enter to create • esc to cancel") + "\n")
	return s.String()
}

func (m *SFTPBrowserModel) viewRename() string {
	var s strings.Builder
	s.WriteString("\n")
	if m.selectedIndex < len(m.files) {
		s.WriteString(styles.TitleStyle.Render(fmt.Sprintf("  Rename: %s", m.files[m.selectedIndex].Name)) + "\n\n")
	} else {
		s.WriteString(styles.TitleStyle.Render("  Rename") + "\n\n")
	}
	s.WriteString("  " + m.nameInput.View() + "\n\n")
	s.WriteString(styles.SubtleStyle.Render("  enter to rename • esc to cancel") + "\n")
	return s.String()
}

func (m *SFTPBrowserModel) viewDeleteConfirm() string {
	var s strings.Builder
	s.WriteString("\n\n")
	if m.selectedIndex < len(m.files) {
		itemType := "file"
		if m.files[m.selectedIndex].IsDir {
			itemType = "directory"
		}
		s.WriteString(styles.TitleStyle.Render(fmt.Sprintf("  Delete %s: %s", itemType, m.files[m.selectedIndex].Name)) + "\n\n")
		s.WriteString("  Are you sure? This action cannot be undone.\n\n")
		s.WriteString(styles.SubtleStyle.Render("  y to confirm • n/esc to cancel") + "\n")
	}
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

	// Success message display
	if m.successMsg != "" {
		s.WriteString(styles.SuccessStyle.Render("  "+m.successMsg) + "\n\n")
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
	helpText := "  j/k: up/down • h: parent • l/enter: open • g: go to path • ~: home • .: toggle hidden\n"
	helpText += "  n: new file • N: new dir • d: delete • r: rename • y: copy path • q: quit"
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
