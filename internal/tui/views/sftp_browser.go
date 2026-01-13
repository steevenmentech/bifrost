package views

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
	scrollOffset  int // First visible file index for scrolling
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
	fileToEdit    string // Path of file to edit (when quitting to edit)
}

func NewSFTPBrowser(client *sftp.Client, keymap keys.KeyMap) *SFTPBrowserModel {
	pathInput := textinput.New()
	pathInput.Placeholder = "Enter path..."
	pathInput.CharLimit = 256
	pathInput.Width = 100 // Will be adjusted on first WindowSizeMsg

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

	// Reset scroll
	m.scrollOffset = 0
}

func (m *SFTPBrowserModel) Init() tea.Cmd {
	return nil
}

func (m *SFTPBrowserModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle window size for all states
	if msg, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = msg.Width
		m.height = msg.Height

		// Adjust path input width based on terminal width
		// Account for: left margin (2) + right margin (2) + border (2) + padding (2) + icon (3) = 11 chars
		m.pathInput.Width = max(30, msg.Width-11)

		return m, nil
	}

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
				m.adjustScroll()
			}
		case key.Matches(msg, m.keys.Down):
			if m.selectedIndex < len(m.files)-1 {
				m.selectedIndex++
				m.adjustScroll()
			}
		case msg.String() == "ctrl+u": // Page up (half page)
			visibleCount := m.getVisibleFileCount()
			jump := max(1, visibleCount/2)
			m.selectedIndex -= jump
			m.selectedIndex = max(0, m.selectedIndex)
			m.adjustScroll()
		case msg.String() == "ctrl+d": // Page down (half page)
			visibleCount := m.getVisibleFileCount()
			jump := max(1, visibleCount/2)
			m.selectedIndex += jump
			if m.selectedIndex >= len(m.files) {
				m.selectedIndex = len(m.files) - 1
			}
			m.adjustScroll()
		case msg.String() == "G": // Go to bottom
			if len(m.files) > 0 {
				m.selectedIndex = len(m.files) - 1
				m.adjustScroll()
			}
		case msg.String() == "g", msg.String() == "tab": // Go to path (inline)
			m.state = GoToPathState
			m.pathInput.SetValue(m.currentPath)
			m.pathInput.Focus()
			return m, textinput.Blink
		case key.Matches(msg, m.keys.Left): // h - parent directory
			m.goToParentDirectory()
		case key.Matches(msg, m.keys.Right), msg.String() == "enter": // l or Enter
			m.enterSelected()
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
		case msg.String() == "e": // Edit file
			if len(m.files) > 0 && m.selectedIndex < len(m.files) {
				if !m.files[m.selectedIndex].IsDir {
					// Mark file for editing and quit to let main handle it
					m.fileToEdit = path.Join(m.currentPath, m.files[m.selectedIndex].Name)
					return m, tea.Quit
				}
			}
		case msg.String() == "D": // Download file to ~/Downloads
			if len(m.files) > 0 && m.selectedIndex < len(m.files) {
				if !m.files[m.selectedIndex].IsDir {
					m.downloadSelectedFile()
				} else {
					m.err = fmt.Errorf("cannot download directories")
				}
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

// downloadSelectedFile downloads the selected file to ~/Downloads
func (m *SFTPBrowserModel) downloadSelectedFile() {
	if m.selectedIndex >= len(m.files) {
		return
	}

	file := m.files[m.selectedIndex]
	remotePath := path.Join(m.currentPath, file.Name)

	// Get home directory and build Downloads path
	homeDir, err := os.UserHomeDir()
	if err != nil {
		m.err = fmt.Errorf("failed to get home directory: %w", err)
		return
	}
	downloadsDir := filepath.Join(homeDir, "Downloads")

	// Ensure Downloads directory exists
	if err := os.MkdirAll(downloadsDir, 0755); err != nil {
		m.err = fmt.Errorf("failed to create Downloads directory: %w", err)
		return
	}

	localPath := filepath.Join(downloadsDir, file.Name)

	// Check if file already exists and add suffix if needed
	localPath = m.getUniqueFilePath(localPath)

	// Download the file
	err = m.client.DownloadFile(remotePath, localPath)
	if err != nil {
		m.err = fmt.Errorf("download failed: %w", err)
		return
	}

	m.successMsg = fmt.Sprintf("Downloaded: %s", filepath.Base(localPath))
}

// getUniqueFilePath returns a unique file path by adding (1), (2), etc. if file exists
func (m *SFTPBrowserModel) getUniqueFilePath(originalPath string) string {
	if _, err := os.Stat(originalPath); os.IsNotExist(err) {
		return originalPath
	}

	dir := filepath.Dir(originalPath)
	ext := filepath.Ext(originalPath)
	name := strings.TrimSuffix(filepath.Base(originalPath), ext)

	counter := 1
	for {
		newPath := filepath.Join(dir, fmt.Sprintf("%s (%d)%s", name, counter, ext))
		if _, err := os.Stat(newPath); os.IsNotExist(err) {
			return newPath
		}
		counter++
	}
}

// adjustScroll adjusts the scroll offset to keep the selected item visible
func (m *SFTPBrowserModel) adjustScroll() {
	visibleLines := m.getVisibleFileCount()
	if visibleLines <= 0 {
		return
	}

	// If selected item is above the visible area, scroll up
	if m.selectedIndex < m.scrollOffset {
		m.scrollOffset = m.selectedIndex
	}

	// If selected item is below the visible area, scroll down
	if m.selectedIndex >= m.scrollOffset+visibleLines {
		m.scrollOffset = m.selectedIndex - visibleLines + 1
	}
}

// getVisibleFileCount calculates how many files can fit in the viewport
func (m *SFTPBrowserModel) getVisibleFileCount() int {
	// Reserve space for:
	// - 2 lines for title + newline
	// - 3 lines for search box (top border, content, bottom border)
	// - 1 line for help text under search box (if in edit mode)
	// - 1 line for spacing
	// - 3 lines for help keys at bottom
	// - 2 lines for spacing/margins
	reservedLines := 12
	if m.err != nil || m.successMsg != "" {
		reservedLines += 2
	}

	availableLines := m.height - reservedLines
	if availableLines < 1 {
		return 1
	}
	return availableLines
}

// GetFileToEdit returns the path of the file to edit (if any)
func (m *SFTPBrowserModel) GetFileToEdit() string {
	return m.fileToEdit
}

func (m *SFTPBrowserModel) View() string {
	switch m.state {
	case CreateFileState:
		return m.viewCreateFile()
	case CreateDirState:
		return m.viewCreateDir()
	case RenameState:
		return m.viewRename()
	case DeleteConfirmState:
		return m.viewDeleteConfirm()
	default:
		// GoToPathState and BrowsingState use the same view (inline editing)
		return m.viewBrowsing()
	}
}

func (m *SFTPBrowserModel) viewCreateFile() string {
	var s strings.Builder
	s.WriteString("\n")
	s.WriteString(styles.TitleStyle.Render("  Create new file") + "\n\n")
	s.WriteString("  " + m.nameInput.View() + "\n\n")
	s.WriteString(styles.SubtleStyle.Render("  Create: enter | Cancel: esc") + "\n")
	return s.String()
}

func (m *SFTPBrowserModel) viewCreateDir() string {
	var s strings.Builder
	s.WriteString("\n")
	s.WriteString(styles.TitleStyle.Render("  Create new directory") + "\n\n")
	s.WriteString("  " + m.nameInput.View() + "\n\n")
	s.WriteString(styles.SubtleStyle.Render("  Create: enter | Cancel: esc") + "\n")
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
	s.WriteString(styles.SubtleStyle.Render("  Rename: enter | Cancel: esc") + "\n")
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
		s.WriteString(styles.SubtleStyle.Render("  Confirm: y | Cancel: n/esc") + "\n")
	}
	return s.String()
}

func (m *SFTPBrowserModel) viewBrowsing() string {
	var s strings.Builder

	// Title
	s.WriteString("\n")
	s.WriteString(styles.TitleStyle.Render("  SFTP Browser") + "\n")

	// Always show path in a search box style
	var pathContent string
	var helpText string
	var icon string

	if m.state == GoToPathState {
		// Active input mode - show cursor and editable
		icon = "\uf002" // Nerd Font: search icon
		pathContent = m.pathInput.View()
		helpText = "  " + styles.SubtleStyle.Render("Navigate: enter | Cancel: esc")
	} else {
		// Display mode - show current path
		icon = "\uf07c" // Nerd Font: folder icon
		pathContent = m.renderBreadcrumb()
		helpText = ""
	}

	// Build the search box manually for pixel-perfect alignment
	// Calculate inner width (terminal - margins - border chars - padding)
	innerWidth := max(30, m.width-8)

	// Build content with icon and path
	var content string

	if m.state == GoToPathState {
		// In edit mode, manually build the line with fixed width
		inputValue := m.pathInput.Value()
		cursor := "█" // Simple block cursor

		// Build the full content first
		content = icon + " > " + inputValue + cursor

		// Calculate visual width and pad to fill
		visualWidth := lipgloss.Width(content)
		if visualWidth < innerWidth {
			content = content + strings.Repeat(" ", innerWidth-visualWidth)
		}
	} else {
		// In display mode, use the content directly
		content = icon + " " + pathContent
		visualWidth := lipgloss.Width(content)
		if visualWidth < innerWidth {
			content = content + strings.Repeat(" ", innerWidth-visualWidth)
		}
	}

	// Draw the box manually with border characters
	borderColor := styles.Primary
	topBorder := lipgloss.NewStyle().Foreground(borderColor).Render("╭" + strings.Repeat("─", innerWidth+2) + "╮")
	bottomBorder := lipgloss.NewStyle().Foreground(borderColor).Render("╰" + strings.Repeat("─", innerWidth+2) + "╯")
	leftBorder := lipgloss.NewStyle().Foreground(borderColor).Render("│")
	rightBorder := lipgloss.NewStyle().Foreground(borderColor).Render("│")

	s.WriteString("  " + topBorder + "\n")
	s.WriteString("  " + leftBorder + " " + content + " " + rightBorder + "\n")
	s.WriteString("  " + bottomBorder + "\n")
	if helpText != "" {
		s.WriteString(helpText + "\n")
	}
	s.WriteString("\n")

	// Error display
	if m.err != nil {
		s.WriteString(styles.ErrorStyle.Render("  Error: "+m.err.Error()) + "\n\n")
	}

	// Success message display
	if m.successMsg != "" {
		s.WriteString(styles.SuccessStyle.Render("  "+m.successMsg) + "\n\n")
	}

	// File list with scrolling
	if len(m.files) == 0 {
		s.WriteString(styles.SubtleStyle.Render("  (empty directory)") + "\n")
	} else {
		visibleCount := m.getVisibleFileCount()
		endIndex := min(m.scrollOffset+visibleCount, len(m.files))

		// Show indicator if there are files above
		if m.scrollOffset > 0 {
			s.WriteString(styles.SubtleStyle.Render(fmt.Sprintf("  ▲ %d more above...", m.scrollOffset)) + "\n")
		}

		// Render visible files
		for i := m.scrollOffset; i < endIndex; i++ {
			line := m.renderFileLine(m.files[i], i == m.selectedIndex)
			s.WriteString(line + "\n")
		}

		// Show indicator if there are files below
		if endIndex < len(m.files) {
			remaining := len(m.files) - endIndex
			s.WriteString(styles.SubtleStyle.Render(fmt.Sprintf("  ▼ %d more below...", remaining)) + "\n")
		}
	}

	// Help text footer - lazygit style
	s.WriteString("\n")

	// All commands in 2 lines with lazygit-style format
	helpLine1 := "  Up/Down: j/k | Page: ctrl-u/d | Bottom: G | Parent: h | Open: l/enter | Path: g/tab | Home: ~ | Hidden: ."
	helpLine2 := "  New file: n | New dir: N | Delete: d | Rename: r | Edit: e | Download: D | Copy path: y | Quit: q"

	s.WriteString(styles.SubtleStyle.Render(helpLine1) + "\n")
	s.WriteString(styles.SubtleStyle.Render(helpLine2) + "\n")

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
	// Icon (using Nerd Font icons)
	icon := "\uf15b" // File icon
	if file.IsDir {
		icon = "\uf07c" // Folder icon
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
