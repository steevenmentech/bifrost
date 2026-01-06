package sftp

import (
	"fmt"
	"io/fs"
	"path"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// Client represents a SFTP client
type Client struct {
	sshConfig  *ssh.ClientConfig
	sshClient  *ssh.Client
	sftpClient *sftp.Client
	host       string
	port       int
	currentDir string
}

// FileInfo represents file/directory information
type FileInfo struct {
	Name        string
	Size        int64
	Mode        fs.FileMode
	ModTime     time.Time
	IsDir       bool
	Permissions string
}

// NewClient creates a new SFTP client
func NewClient(host string, port int, username, password string) (*Client, error) {
	config := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	return &Client{
		sshConfig: config,
		host:      host,
		port:      port,
	}, nil
}

// Connect establishes the SSH and SFTP connection
func (c *Client) Connect() error {
	// First establish SSH connection
	addr := fmt.Sprintf("%s:%d", c.host, c.port)
	sshClient, err := ssh.Dial("tcp", addr, c.sshConfig)
	if err != nil {
		return fmt.Errorf("failed to dial SSH: %w", err)
	}
	c.sshClient = sshClient

	// Then create SFTP session
	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		c.sshClient.Close()
		return fmt.Errorf("failed to create SFTP client: %w", err)
	}
	c.sftpClient = sftpClient

	// Get initial working directory
	wd, err := sftpClient.Getwd()
	if err != nil {
		c.currentDir = "/"
	} else {
		c.currentDir = wd
	}

	return nil
}

// ListDir lists the contents of a directory
func (c *Client) ListDir(dirPath string) ([]FileInfo, error) {
	if c.sftpClient == nil {
		return nil, fmt.Errorf("not connected")
	}

	entries, err := c.sftpClient.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	fileInfos := make([]FileInfo, 0, len(entries))
	for _, entry := range entries {
		fileInfos = append(fileInfos, FileInfo{
			Name:        entry.Name(),
			Size:        entry.Size(),
			Mode:        entry.Mode(),
			ModTime:     entry.ModTime(),
			IsDir:       entry.IsDir(),
			Permissions: entry.Mode().String(),
		})
	}

	return fileInfos, nil
}

// Stat gets file/directory metadata
func (c *Client) Stat(filePath string) (*FileInfo, error) {
	if c.sftpClient == nil {
		return nil, fmt.Errorf("not connected")
	}

	info, err := c.sftpClient.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	return &FileInfo{
		Name:        info.Name(),
		Size:        info.Size(),
		Mode:        info.Mode(),
		ModTime:     info.ModTime(),
		IsDir:       info.IsDir(),
		Permissions: info.Mode().String(),
	}, nil
}

// ChangeDir changes the current working directory
func (c *Client) ChangeDir(dirPath string) error {
	if c.sftpClient == nil {
		return fmt.Errorf("not connected")
	}

	// Check if directory exists and is accessible
	info, err := c.sftpClient.Stat(dirPath)
	if err != nil {
		return fmt.Errorf("directory not found: %w", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", dirPath)
	}

	// Update current directory
	if path.IsAbs(dirPath) {
		c.currentDir = path.Clean(dirPath)
	} else {
		c.currentDir = path.Clean(path.Join(c.currentDir, dirPath))
	}

	return nil
}

// GetWorkingDir returns the current working directory
func (c *Client) GetWorkingDir() string {
	return c.currentDir
}

// Close closes the SFTP and SSH connections
func (c *Client) Close() error {
	if c.sftpClient != nil {
		c.sftpClient.Close()
	}
	if c.sshClient != nil {
		return c.sshClient.Close()
	}
	return nil
}

// CreateFile creates a new empty file
func (c *Client) CreateFile(filePath string) error {
	if c.sftpClient == nil {
		return fmt.Errorf("not connected")
	}

	file, err := c.sftpClient.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	return nil
}

// CreateDirectory creates a new directory
func (c *Client) CreateDirectory(dirPath string) error {
	if c.sftpClient == nil {
		return fmt.Errorf("not connected")
	}

	err := c.sftpClient.Mkdir(dirPath)
	if err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	return nil
}

// Delete removes a file or directory
func (c *Client) Delete(itemPath string) error {
	if c.sftpClient == nil {
		return fmt.Errorf("not connected")
	}

	// Check if it's a directory
	info, err := c.sftpClient.Stat(itemPath)
	if err != nil {
		return fmt.Errorf("failed to stat item: %w", err)
	}

	if info.IsDir() {
		// For directories, use RemoveDirectory
		err = c.sftpClient.RemoveDirectory(itemPath)
	} else {
		// For files, use Remove
		err = c.sftpClient.Remove(itemPath)
	}

	if err != nil {
		return fmt.Errorf("failed to delete: %w", err)
	}

	return nil
}

// Rename renames or moves a file/directory
func (c *Client) Rename(oldPath, newPath string) error {
	if c.sftpClient == nil {
		return fmt.Errorf("not connected")
	}

	err := c.sftpClient.Rename(oldPath, newPath)
	if err != nil {
		return fmt.Errorf("failed to rename: %w", err)
	}

	return nil
}

// ConnectFromConfig creates and connects an SFTP client using connection details
func ConnectFromConfig(host string, port int, username, password string) (*Client, error) {
	client, err := NewClient(host, port, username, password)
	if err != nil {
		return nil, err
	}

	if err := client.Connect(); err != nil {
		return nil, err
	}

	return client, nil
}
