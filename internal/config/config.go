package config

import (
	"fmt"
	"os"
	"strconv"

	"gopkg.in/yaml.v3"
)

const (
	// Default config file path
	DefaultConfigPath = "config.yaml"
	// Environment variables
	EnvConfigPath   = "RHINO_CONFIG"
	EnvWorkflowsDir = "RHINO_WORKFLOWS_DIR"
	EnvPort         = "RHINO_PORT"
)

type Config struct {
	WorkflowsDir string `yaml:"workflows-dir"`
	Port         int    `yaml:"port"`
}

var (
	// Global config instance (cached)
	globalConfig *Config
)

// NewConfig creates a new Config instance with the given values
func NewConfig(workflowsDir string, port int) *Config {
	return &Config{WorkflowsDir: workflowsDir, Port: port}
}

// GetConfigPath returns the config file path, checking environment variable first
func GetConfigPath() string {
	if path := os.Getenv(EnvConfigPath); path != "" {
		return path
	}
	return DefaultConfigPath
}

// LoadConfig loads configuration from the specified path and applies environment variable overrides
func LoadConfig(configPath string) (*Config, error) {
	file, err := os.Open(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	var config Config
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to decode config: %w", err)
	}

	// Apply environment variable overrides
	if workflowsDir := os.Getenv(EnvWorkflowsDir); workflowsDir != "" {
		config.WorkflowsDir = workflowsDir
	}

	if portStr := os.Getenv(EnvPort); portStr != "" {
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return nil, fmt.Errorf("invalid port in %s: %w", EnvPort, err)
		}
		config.Port = port
	}

	// Validate the config
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &config, nil
}

// Load loads the global config instance
func Load() (*Config, error) {
	if globalConfig != nil {
		return globalConfig, nil
	}

	configPath := GetConfigPath()
	config, err := LoadConfig(configPath)
	if err != nil {
		return nil, err
	}

	globalConfig = config
	return globalConfig, nil
}

// Save saves the config to the specified path (or default)
func (c *Config) Save(configPath string) error {
	if configPath == "" {
		configPath = GetConfigPath()
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	file, err := os.OpenFile(configPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to open config file for writing: %w", err)
	}
	defer file.Close()

	if _, err = file.Write(data); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.WorkflowsDir == "" {
		return fmt.Errorf("workflows directory not set")
	}
	if c.Port == 0 {
		return fmt.Errorf("port not set")
	}
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535, got %d", c.Port)
	}
	return nil
}

// SaveDefaultConfig creates and saves a default configuration
func SaveDefaultConfig() error {
	configPath := GetConfigPath()
	config := NewConfig("workflows", 8888)
	return config.Save(configPath)
}

// ConfigFileExists checks if the config file exists
func ConfigFileExists() bool {
	configPath := GetConfigPath()
	_, err := os.Stat(configPath)
	return !os.IsNotExist(err)
}

// GetString returns a string configuration value by key
func GetString(key string) string {
	config, err := Load()
	if err != nil {
		return ""
	}

	switch key {
	case "workflows-dir":
		return config.WorkflowsDir
	default:
		return ""
	}
}

// GetInt returns an integer configuration value by key
func GetInt(key string) int {
	config, err := Load()
	if err != nil {
		return 0
	}

	switch key {
	case "port":
		return config.Port
	default:
		return 0
	}
}

// Reset clears the global config cache (useful for testing)
func Reset() {
	globalConfig = nil
}