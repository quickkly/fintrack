package config

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	Bend BendConfig `mapstructure:"bend"`
}

// BendConfig represents Bend financial service configuration
type BendConfig struct {
	BaseURL        string        `mapstructure:"base_url"`
	RateLimit      time.Duration `mapstructure:"rate_limit"`
	SessionFile    string        `mapstructure:"session_file"`
	Timeout        time.Duration `mapstructure:"timeout"`
	RefreshToken   string        `mapstructure:"refresh_token"`   // Initial refresh token
	DeviceHash     string        `mapstructure:"device_hash"`     // Device identifier
	DeviceType     string        `mapstructure:"device_type"`     // Device type (Web/Mobile)
	DeviceLocation string        `mapstructure:"device_location"` // Device location
}

// Load initializes and loads the configuration
func Load(configFile string) (*Config, error) {
	v := viper.New()

	// Set defaults
	setDefaults(v)

	// Set config file
	if configFile != "" {
		v.SetConfigFile(configFile)
	} else {
		// Set config search paths (local first, then global)
		v.SetConfigName("config")
		v.SetConfigType("yaml")

		// Local config paths (like Git's local config)
		v.AddConfigPath(".fintrack")
		v.AddConfigPath("./configs")
		v.AddConfigPath(".")

		// Global config path (like Git's global config)
		configDir, err := getConfigDir()
		if err == nil {
			v.AddConfigPath(configDir)
		}
	}

	// Environment variable support
	v.AutomaticEnv()
	v.SetEnvPrefix("FINTRACK")

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	// Unmarshal config
	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Get the directory of the config file used
	configFileDir := ""
	if usedConfig := v.ConfigFileUsed(); usedConfig != "" {
		configFileDir = filepath.Dir(usedConfig)
	}

	// Expand file paths relative to config file location
	if err := expandPaths(&config, configFileDir); err != nil {
		return nil, fmt.Errorf("failed to expand paths: %w", err)
	}

	// Handle device hash - generate if not provided
	if err := ensureDeviceHash(&config); err != nil {
		return nil, fmt.Errorf("failed to ensure device hash: %w", err)
	}

	return &config, nil
}

// setDefaults sets default configuration values
func setDefaults(v *viper.Viper) {
	// Bend defaults
	v.SetDefault("bend.base_url", "https://bend.example.com")
	v.SetDefault("bend.rate_limit", "1s")
	v.SetDefault("bend.timeout", "30s")
	v.SetDefault("bend.device_type", "Web")
	v.SetDefault("bend.device_location", "Default")

}

// getConfigDir returns the configuration directory path
func getConfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".config", "fintrack"), nil
}

// expandPaths expands ~ and environment variables in file paths
func expandPaths(config *Config, configFileDir string) error {
	var err error

	config.Bend.SessionFile, err = expandPath(config.Bend.SessionFile, configFileDir)
	if err != nil {
		return err
	}

	fmt.Printf("[config] session_file resolved to: %s\n", config.Bend.SessionFile)

	return nil
}

// expandPath expands ~ and environment variables in a file path
// Relative paths are resolved against the config file directory
func expandPath(path string, configFileDir string) (string, error) {
	if path == "" {
		return path, nil
	}

	// Expand environment variables
	path = os.ExpandEnv(path)

	// Expand ~ to home directory
	if len(path) > 0 && path[0] == '~' {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		path = filepath.Join(homeDir, path[1:])
	}

	// If path is relative, resolve against config file directory
	if !filepath.IsAbs(path) && configFileDir != "" {
		path = filepath.Join(configFileDir, path)
	}

	return path, nil
}

// EnsureConfigDir creates the configuration directory if it doesn't exist
func EnsureConfigDir() error {
	configDir, err := getConfigDir()
	if err != nil {
		return err
	}

	return os.MkdirAll(configDir, 0755)
}

// GetConfigFilePath returns the default config file path
func GetConfigFilePath() (string, error) {
	configDir, err := getConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "config.yaml"), nil
}

// ensureDeviceHash ensures the configuration has a device hash, generating one if needed
func ensureDeviceHash(config *Config) error {
	if config.Bend.DeviceHash != "" {
		return nil // Already has a device hash
	}

	// Get config directory for storing device hash
	configDir, err := getConfigDir()
	if err != nil {
		// If we can't get config dir, generate a temporary one
		config.Bend.DeviceHash = generateDeviceHash()
		return nil
	}

	// Try to get or create persistent device hash
	deviceHash, err := getOrCreateDeviceHash(configDir)
	if err != nil {
		// If we can't persist, generate a temporary one
		config.Bend.DeviceHash = generateDeviceHash()
		return nil
	}

	config.Bend.DeviceHash = deviceHash
	return nil
}

// generateDeviceHash generates a unique device hash (UUID v4 format)
func generateDeviceHash() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		// Fallback to a deterministic but unique hash based on hostname and user
		hostname, _ := os.Hostname()
		user := os.Getenv("USER")
		if user == "" {
			user = os.Getenv("USERNAME") // Windows fallback
		}
		fallback := fmt.Sprintf("fintrack-%s-%s", hostname, user)
		// Convert to UUID format (truncate/pad as needed)
		if len(fallback) < 16 {
			fallback = fallback + "0000000000000000"
		}
		return fmt.Sprintf("%x-%x-%x-%x-%x",
			[]byte(fallback)[:4],
			[]byte(fallback)[4:6],
			[]byte(fallback)[6:8],
			[]byte(fallback)[8:10],
			[]byte(fallback)[10:16])
	}

	// Set version (4) and variant bits
	b[6] = (b[6] & 0x0f) | 0x40 // Version 4
	b[8] = (b[8] & 0x3f) | 0x80 // Variant 10

	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

// getOrCreateDeviceHash returns existing device hash from config dir or creates a new one
func getOrCreateDeviceHash(configDir string) (string, error) {
	deviceHashFile := filepath.Join(configDir, "device_hash")

	// Try to read existing device hash
	if data, err := os.ReadFile(deviceHashFile); err == nil {
		deviceHash := string(data)
		if len(deviceHash) > 0 {
			return deviceHash, nil
		}
	}

	// Generate new device hash
	deviceHash := generateDeviceHash()

	// Ensure config directory exists
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return deviceHash, fmt.Errorf("failed to create config directory: %w", err)
	}

	// Save device hash to file
	if err := os.WriteFile(deviceHashFile, []byte(deviceHash), 0644); err != nil {
		return deviceHash, fmt.Errorf("failed to save device hash: %w", err)
	}

	return deviceHash, nil
}

// configContextKey is the key used to store config in context
type configContextKey struct{}

// GetFromContext retrieves the configuration from the command context
func GetFromContext(cmd *cobra.Command) (*Config, error) {
	if cmd == nil || cmd.Context() == nil {
		return nil, fmt.Errorf("command or context is nil")
	}

	cfg, ok := cmd.Context().Value(configContextKey{}).(*Config)
	if !ok || cfg == nil {
		return nil, fmt.Errorf("configuration not found in context")
	}

	return cfg, nil
}

// SetInContext stores the configuration in the command context
func SetInContext(cmd *cobra.Command, cfg *Config) {
	if cmd != nil {
		ctx := cmd.Context()
		if ctx == nil {
			ctx = cmd.Root().Context()
		}
		if ctx != nil {
			ctx = context.WithValue(ctx, configContextKey{}, cfg)
			cmd.SetContext(ctx)
		}
	}
}
