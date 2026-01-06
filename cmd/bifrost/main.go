package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/steevenmentech/bifrost/internal/config"
	"github.com/steevenmentech/bifrost/internal/keyring"
	"github.com/steevenmentech/bifrost/internal/ssh"
	"github.com/steevenmentech/bifrost/internal/tui"
)

func main() {
	for {
		// Create the TUI model
		model, err := tui.New()
		if err != nil {
			fmt.Printf("Error initializing app: %v\n", err)
			os.Exit(1)
		}

		// Create and run the Bubble Tea program
		p := tea.NewProgram(
			model,
			tea.WithAltScreen(),
		)

		// Run the program and get the final model
		finalModel, err := p.Run()
		if err != nil {
			fmt.Printf("Error running app: %v\n", err)
			os.Exit(1)
		}

		// Check if user selected a connection
		tuiModel, ok := finalModel.(tui.Model)
		if !ok {
			// Model type assertion failed, exit
			break
		}

		selectedConn := tuiModel.GetSelectedConnection()
		if selectedConn == nil {
			// No connection selected, user quit normally
			break
		}

		// Get connection type (0=SSH, 1=SFTP)
		connType := tuiModel.GetConnectionType()

		// Start appropriate session
		if connType == 0 {
			// SSH
			if err := startSSHSession(*selectedConn); err != nil {
				fmt.Printf("\nSSH Error: %v\n", err)
				fmt.Println("Press Enter to continue...")
				var input string
				fmt.Scanln(&input)
				continue
			}

			fmt.Println("\nDisconnected from server.")
			fmt.Println("Press Enter to return to Bifrost...")
			var input string
			fmt.Scanln(&input)
		} else {
			// SFTP (not implemented yet)
			fmt.Println("\nSFTP functionality coming soon!")
			fmt.Println("Press Enter to return to Bifrost...")
			var input string
			fmt.Scanln(&input)
		}
	}
}

// startSSHSession connects to a server via SSH
func startSSHSession(conn config.Connection) error {
	// Get password from keyring
	password, err := keyring.GetConnectionPassword(conn.ID)
	if err != nil {
		return fmt.Errorf("failed to get password: %w", err)
	}

	// Create and connect SSH client
	fmt.Printf("\nConnecting to %s@%s:%d...\n\n", conn.Username, conn.Host, conn.Port)

	sshClient, err := ssh.ConnectFromConfig(conn.Host, conn.Port, conn.Username, password)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer sshClient.Close()

	// Start interactive session
	if err := sshClient.StartInteractiveSession(); err != nil {
		return fmt.Errorf("session error: %w", err)
	}

	return nil
}
