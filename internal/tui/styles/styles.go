package styles

import "github.com/charmbracelet/lipgloss"

// Color palette
var (
	Primary   = lipgloss.Color("#c9c6cbff") // Purple
	Secondary = lipgloss.Color("#06B6D4")   // Cyan
	Success   = lipgloss.Color("#10B981")   // Green
	Warning   = lipgloss.Color("#F59E0B")   // Amber
	Error     = lipgloss.Color("#EF4444")   // Red

	Foreground = lipgloss.Color("#E5E7EB") // Light gray
	Background = lipgloss.Color("#1F2937") // Dark gray
	Muted      = lipgloss.Color("#6B7280") // Medium gray

	// Special colors
	Highlight = lipgloss.Color("#FFFFFF") // White
	Dim       = lipgloss.Color("#4B5563") // Very muted
)

// ASCII Art Logo for Bifrost
const Logo = `
 ██████╗ ██╗███████╗██████╗  ██████╗ ███████╗████████╗
 ██╔══██╗██║██╔════╝██╔══██╗██╔═══██╗██╔════╝╚══██╔══╝
 ██████╔╝██║█████╗  ██████╔╝██║   ██║███████╗   ██║
 ██╔══██╗██║██╔══╝  ██╔══██╗██║   ██║╚════██║   ██║
 ██████╔╝██║██║     ██║  ██║╚██████╔╝███████║   ██║
 ╚═════╝ ╚═╝╚═╝     ╚═╝  ╚═╝ ╚═════╝ ╚══════╝   ╚═╝
`

// Compact logo for smaller terminals
const LogoCompact = `
╔╗ ╦╔═╗╦═╗╔═╗╔═╗╔╦╗
╠╩╗║╠╣ ╠╦╝║ ║╚═╗ ║
╚═╝╩╚  ╩╚═╚═╝╚═╝ ╩ `

// Title style (top of the app)
var TitleStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(Primary).
	Padding(0, 1)

// Status bar style (bottom of the app)
var StatusBarStyle = lipgloss.NewStyle().
	Foreground(Muted).
	Background(Background).
	Padding(0, 1)

// Help text in status bar
var HelpStyle = lipgloss.NewStyle().
	Foreground(Dim)

// Selected item style
var SelectedStyle = lipgloss.NewStyle().
	Foreground(Highlight).
	Background(Primary).
	Bold(true).
	Padding(0, 1)

// Normal item style
var ItemStyle = lipgloss.NewStyle().
	Foreground(Foreground).
	Padding(0, 1)

	// Border style for panels
var BorderStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(Muted).
	Padding(1, 2)

// Error message style
var ErrorStyle = lipgloss.NewStyle().
	Foreground(Error).
	Bold(true)

// Success message style
var SuccessStyle = lipgloss.NewStyle().
	Foreground(Success).
	Bold(true)

// Subtle/muted text style
var SubtleStyle = lipgloss.NewStyle().
	Foreground(Muted)

// Logo style with gradient effect
var LogoStyle = lipgloss.NewStyle().
	Foreground(Primary).
	Bold(true)

// Floating modal/dialog style (like lazygit)
var ModalStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(Primary).
	Padding(1, 2)

// Modal title style
var ModalTitleStyle = lipgloss.NewStyle().
	Foreground(Highlight).
	Bold(true)
