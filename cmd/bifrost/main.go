package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/steevenmentech/bifrost/internal/config"
	"github.com/steevenmentech/bifrost/internal/keyring"
	"github.com/steevenmentech/bifrost/internal/sftp"
	"github.com/steevenmentech/bifrost/internal/ssh"
	"github.com/steevenmentech/bifrost/internal/tui"
	"github.com/steevenmentech/bifrost/internal/tui/keys"
	"github.com/steevenmentech/bifrost/internal/tui/views"
	"golang.org/x/term"
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
			// SFTP
			if err := startSFTPSession(*selectedConn); err != nil {
				fmt.Printf("\nSFTP Error: %v\n", err)
				fmt.Println("Press Enter to continue...")
				var input string
				fmt.Scanln(&input)
				continue
			}

			fmt.Println("\nDisconnected from server.")
			fmt.Println("Press Enter to return to Bifrost...")
			var input string
			fmt.Scanln(&input)
		}
	}
}

// getConnectionPassword retrieves password based on auth type, prompts if not found
func getConnectionPassword(conn config.Connection) (string, error) {
	var password string
	var err error

	if conn.AuthType == "credential" && conn.CredentialID != "" {
		// Get password from credential
		password, err = keyring.GetCredentialPassword(conn.CredentialID)
	} else {
		// Get password from connection
		password, err = keyring.GetConnectionPassword(conn.ID)
	}

	// If password not found, prompt user
	if err != nil {
		fmt.Printf("Password for %s@%s: ", conn.Username, conn.Host)
		// Read password (hidden input)
		passwordBytes, readErr := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Println() // New line after password input
		if readErr != nil {
			return "", fmt.Errorf("failed to read password: %w", readErr)
		}
		password = string(passwordBytes)

		// Offer to save password
		fmt.Print("Save password to keyring? (y/n): ")
		var save string
		fmt.Scanln(&save)
		if save == "y" || save == "Y" {
			if conn.AuthType == "credential" && conn.CredentialID != "" {
				_ = keyring.SetCredentialPassword(conn.CredentialID, password)
			} else {
				_ = keyring.SetConnectionPassword(conn.ID, password)
			}
			fmt.Println("Password saved!")
		}
	}

	return password, nil
}

// startSSHSession connects to a server via SSH
func startSSHSession(conn config.Connection) error {
	// Get password based on auth type
	password, err := getConnectionPassword(conn)
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

// startSFTPSession connects to a server via SFTP and shows the file browser
func startSFTPSession(conn config.Connection) error {
	// Get password based on auth type
	password, err := getConnectionPassword(conn)
	if err != nil {
		return fmt.Errorf("failed to get password: %w", err)
	}

	// Create and connect SFTP client
	fmt.Printf("\nConnecting to %s@%s:%d via SFTP...\n", conn.Username, conn.Host, conn.Port)

	sftpClient, err := sftp.ConnectFromConfig(conn.Host, conn.Port, conn.Username, password)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer sftpClient.Close()

	fmt.Println("Connected! Loading SFTP browser...\n")

	// Loop to handle file editing
	for {
		// Create SFTP browser model
		browser := views.NewSFTPBrowser(sftpClient, keys.DefaultKeyMap())

		// Create and run the Bubble Tea program
		p := tea.NewProgram(
			browser,
			tea.WithAltScreen(),
		)

		// Run the program
		finalModel, err := p.Run()
		if err != nil {
			return fmt.Errorf("SFTP browser error: %w", err)
		}

		// Check if we need to edit a file
		browserModel, ok := finalModel.(*views.SFTPBrowserModel)
		if !ok {
			break
		}

		fileToEdit := browserModel.GetFileToEdit()
		if fileToEdit == "" {
			// User quit normally, exit
			break
		}

		// Edit the file
		if err := editRemoteFile(sftpClient, fileToEdit); err != nil {
			fmt.Printf("\nEdit Error: %v\n", err)
			fmt.Println("Press Enter to continue...")
			var input string
			fmt.Scanln(&input)
		}
	}

	return nil
}

// editRemoteFile downloads a file, opens it in an editor, and uploads it back
func editRemoteFile(client *sftp.Client, remotePath string) error {
	// Extract filename from path
	fileName := filepath.Base(remotePath)

	// Create temp file path
	tempDir := os.TempDir()
	localPath := filepath.Join(tempDir, fileName)

	// Download file
	fmt.Printf("\nDownloading %s...\n", fileName)
	err := client.DownloadFile(remotePath, localPath)
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}
	defer os.Remove(localPath) // Clean up temp file

	// Get editor: $EDITOR > config > fallback to nano
	editor := os.Getenv("EDITOR")
	if editor == "" {
		// Try to get from config
		cfg, err := config.Load()
		if err == nil && cfg.Settings.Editor != "" {
			editor = cfg.Settings.Editor
		} else {
			editor = "nano"
		}
	}

	// Open editor
	fmt.Printf("Opening %s in %s...\n\n", fileName, editor)

	// Parse editor command and arguments
	// Look for flags starting with " -" to separate command from args
	var editorCmd string
	var editorArgs []string

	// Find first occurrence of " -" which usually indicates flags
	flagIndex := -1
	for i := 0; i < len(editor)-1; i++ {
		if editor[i] == ' ' && editor[i+1] == '-' {
			flagIndex = i
			break
		}
	}

	if flagIndex > 0 {
		// Split into command and args
		editorCmd = strings.TrimSpace(editor[:flagIndex])
		argsStr := strings.TrimSpace(editor[flagIndex:])
		editorArgs = strings.Fields(argsStr)
	} else {
		// No flags found, use entire string as command
		editorCmd = strings.TrimSpace(editor)
	}

	// Build command with editor args + file path
	args := append(editorArgs, localPath)
	cmd := exec.Command(editorCmd, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to run editor: %w", err)
	}

	// Upload file back
	fmt.Printf("\nUploading %s...\n", fileName)
	err = client.UploadFile(localPath, remotePath)
	if err != nil {
		return fmt.Errorf("failed to upload file: %w", err)
	}

	fmt.Printf("File saved successfully!\n")
	fmt.Println("Press Enter to return to browser...")
	var input string
	fmt.Scanln(&input)

	return nil
}
