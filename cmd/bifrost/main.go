package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/steevenmentech/bifrost/internal/tui"
)

func main() {
	// Create the TUI model
	model, err := tui.New()
	if err != nil {
		fmt.Printf("Error initializing app: %v\n", err)
		os.Exit(1)
	}

	// Create the Bubble Tea program
	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),       // Use alternate screen buffer
		tea.WithMouseCellMotion(), // Enable mouse support
	)

	// Run the program
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running app: %v\n", err)
		os.Exit(1)
	}
}
