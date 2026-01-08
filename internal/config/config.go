package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
	"github.com/google/uuid"
	"github.com/spf13/viper"
)

// Config represents the application configuration.
type Config struct {
	Version     int          `yaml:"version"`
	Settings    Settings     `yaml:"settings"`
	Connections []Connection `yaml:"connections"`
	Credentials []Credential `yaml:"credentials"`
}

// Settings contains user preferences and application settings.
type Settings struct {
	Editor          string `yaml:"editor"`
	Theme           string `yaml:"theme"`
	ShowHiddenFiles string `yaml:"show_hidden_files"`
	ConfirmDelete   string `yaml:"confirm_delete"`
	DefaultPort     int    `yaml:"default_port"`
}

// Connection repesents a sing SSH/SFTP connection.
type Connection struct {
	ID           string `yaml:"id"`
	Label        string `yaml:"label"`
	Icon         string `yaml:"icon"`
	Host         string `yaml:"host"`
	Port         int    `yaml:"port"`
	Username     string `yaml:"username"`
	AuthType     string `yaml:"auth_type"`     // "password" | "key" | "credential"
	CredentialID string `yaml:"credential_id"` // if using shared credential
	KeyPath      string `yaml:"key_path"`      // if using SSH key
}

// Credential represents shared authentication details.
type Credential struct {
	ID       string `yaml:"id"`
	Label    string `yaml:"label"`
	Username string `yaml:"username"`
	// Password stored in OS keyring; not in config file
}

// GetConfigPath returns the path to the configuration file, creating if necessary.
func GetConfigPath() (string, error) {
	configDir := filepath.Join(xdg.ConfigHome, "bifrost")

	// create config directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create config directory: %w", err)
	}

	return filepath.Join(configDir, "config.yaml"), nil
}

// Load loads the configuration from disk
func Load() (*Config, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	// check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// create default config
		cfg := defaultConfig()
		if err := cfg.Save(); err != nil {
			return nil, fmt.Errorf("Failed to create default config: %w", err)
		}
		return cfg, nil
	}

	// Read existing config
	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("Failed to read config: %w", err)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("Failed to parse config: %w", err)
	}

	return &cfg, nil
}

// Save saves the configuration to disk
func (cfg *Config) Save() error {
	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	viper.Set("version", cfg.Version)
	viper.Set("settings", cfg.Settings)
	viper.Set("connections", cfg.Connections)
	viper.Set("credentials", cfg.Credentials)

	return viper.WriteConfigAs(configPath)
}

func defaultConfig() *Config {
	return &Config{
		Version: 1,
		Settings: Settings{
			Editor:          "nvim",
			Theme:           "default",
			ShowHiddenFiles: "false",
			ConfirmDelete:   "true",
			DefaultPort:     22,
		},
		Connections: []Connection{},
		Credentials: []Credential{},
	}
}

// AddConnection adds a new connection to the configuration
func (cfg *Config) AddConnection(conn Connection) error {
	if conn.ID == "" {
		conn.ID = uuid.New().String()
	}

	cfg.Connections = append(cfg.Connections, conn)
	return cfg.Save()
}

// DeleteConnection removes a connection from the configuration by ID
func (cfg *Config) DeleteConnection(id string) error {
	for i, conn := range cfg.Connections {
		if conn.ID == id {
			cfg.Connections = append(cfg.Connections[:i], cfg.Connections[i+1:]...)
			return cfg.Save()
		}
	}
	return fmt.Errorf("connection with ID %s not found", id)
}

// UpdateConnection updates an existing connection in the configuration
func (cfg *Config) UpdateConnection(updated Connection) error {
	for i, conn := range cfg.Connections {
		if conn.ID == updated.ID {
			cfg.Connections[i] = updated
			return cfg.Save()
		}
	}
	return fmt.Errorf("connection with ID %s not found", updated.ID)
}

// GetConnection retrieves a connection by ID
func (cfg *Config) GetConnection(id string) (*Connection, error) {
	for _, conn := range cfg.Connections {
		if conn.ID == id {
			return &conn, nil
		}
	}
	return nil, fmt.Errorf("connection with ID %s not found", id)
}

// AddCredential adds a new credential to the configuration
func (cfg *Config) AddCredential(cred Credential) error {
	if cred.ID == "" {
		cred.ID = uuid.New().String()
	}

	cfg.Credentials = append(cfg.Credentials, cred)
	return cfg.Save()
}

// DeleteCredential removes a credential from the configuration by ID
func (cfg *Config) DeleteCredential(id string) error {
	for i, cred := range cfg.Credentials {
		if cred.ID == id {
			cfg.Credentials = append(cfg.Credentials[:i], cfg.Credentials[i+1:]...)
			return cfg.Save()
		}
	}
	return fmt.Errorf("credential with ID %s not found", id)
}

// UpdateCredential updates an existing credential in the configuration
func (cfg *Config) UpdateCredential(updated Credential) error {
	for i, cred := range cfg.Credentials {
		if cred.ID == updated.ID {
			cfg.Credentials[i] = updated
			return cfg.Save()
		}
	}
	return fmt.Errorf("credential with ID %s not found", updated.ID)
}

// GetCredential retrieves a credential by ID
func (cfg *Config) GetCredential(id string) (*Credential, error) {
	for _, cred := range cfg.Credentials {
		if cred.ID == id {
			return &cred, nil
		}
	}
	return nil, fmt.Errorf("credential with ID %s not found", id)
}
