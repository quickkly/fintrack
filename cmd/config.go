package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/quickkly/fintrack/internal/config"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// =============================================================================
// CONFIGURATION COMMAND DEFINITIONS
// =============================================================================

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configuration management",
	Long: `Manage fintrack configuration settings.

Available subcommands:
- show: Display current configuration
- set: Set a configuration value
- get: Get a configuration value
- validate: Validate configuration syntax and values`,
}

// configShowCmd shows the current configuration
var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	Long:  "Display the current configuration in YAML format",
	RunE:  runConfigShow,
}

// configSetCmd sets a configuration value
var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Long:  "Set a configuration value. Use dot notation for nested keys (e.g., bend.base_url)",
	Args:  cobra.ExactArgs(2),
	Example: `  fintrack config set bend.base_url "https://bend.example.com"
  fintrack config set bend.timeout "60s"
  fintrack config set bend.device_type "CLI"`,
	RunE: runConfigSet,
}

// configGetCmd gets a configuration value
var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a configuration value",
	Long:  "Get a configuration value. Use dot notation for nested keys",
	Args:  cobra.ExactArgs(1),
	Example: `  fintrack config get bend.base_url
  fintrack config get bend.timeout`,
	RunE: runConfigGet,
}

// configValidateCmd validates the configuration
var configValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate configuration",
	Long:  "Validate the configuration file syntax and required values",
	RunE:  runConfigValidate,
}

func init() {
	// Add subcommands
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configValidateCmd)
}

// =============================================================================
// CONFIGURATION COMMAND IMPLEMENTATIONS
// =============================================================================

// runConfigShow displays the current configuration
func runConfigShow(cmd *cobra.Command, args []string) error {
	cfg, err := config.GetFromContext(cmd)
	if err != nil {
		return fmt.Errorf("failed to get configuration: %w", err)
	}

	// Marshal to YAML for pretty printing
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	fmt.Print(string(data))
	return nil
}

// runConfigSet sets a configuration value
func runConfigSet(cmd *cobra.Command, args []string) error {
	key := args[0]
	value := args[1]

	// Validate the key format
	if err := validateConfigKey(key); err != nil {
		return fmt.Errorf("invalid config key: %w", err)
	}

	// Validate the value for known keys
	if err := validateConfigValue(key, value); err != nil {
		return fmt.Errorf("invalid config value: %w", err)
	}

	// Load and update configuration
	v, err := loadViperConfig()
	if err != nil {
		return err
	}

	// Set the value
	v.Set(key, value)

	// Write back to file
	if err := v.WriteConfig(); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	if !IsQuiet() {
		fmt.Printf("✓ Set %s = %s\n", key, value)
	}

	return nil
}

// runConfigGet gets a configuration value
func runConfigGet(cmd *cobra.Command, args []string) error {
	key := args[0]

	// Validate the key format
	if err := validateConfigKey(key); err != nil {
		return fmt.Errorf("invalid config key: %w", err)
	}

	// Load configuration
	v, err := loadViperConfig()
	if err != nil {
		return err
	}

	value := v.Get(key)
	if value == nil {
		return fmt.Errorf("key '%s' not found", key)
	}

	fmt.Println(value)
	return nil
}

// runConfigValidate validates the configuration
func runConfigValidate(cmd *cobra.Command, args []string) error {
	// Load configuration
	v, err := loadViperConfig()
	if err != nil {
		return err
	}

	// Parse into config struct to validate
	var cfg config.Config
	if err := v.Unmarshal(&cfg); err != nil {
		return fmt.Errorf("configuration syntax error: %w", err)
	}

	// Validate required fields
	if err := validateConfiguration(&cfg); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	if !IsQuiet() {
		fmt.Println("✓ Configuration is valid")
	}

	return nil
}

// =============================================================================
// CONFIGURATION UTILITIES
// =============================================================================

// loadViperConfig loads the viper configuration
func loadViperConfig() (*viper.Viper, error) {
	v := viper.New()

	// Set config file path
	if cfgFile != "" {
		v.SetConfigFile(cfgFile)
	} else {
		// Use the same local-first approach as the main config loading
		v.SetConfigName("config")
		v.SetConfigType("yaml")

		// Local config paths (like Git's local config)
		v.AddConfigPath(".fintrack")
		v.AddConfigPath("./configs")
		v.AddConfigPath(".")

		// Global config path (like Git's global config)
		homeDir, err := os.UserHomeDir()
		if err == nil {
			globalConfigDir := filepath.Join(homeDir, ".config", "fintrack")
			v.AddConfigPath(globalConfigDir)
		}
	}

	// Read existing config
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config: %w", err)
		}
		// Config file doesn't exist, that's okay for set operations
	}

	return v, nil
}

// validateConfigKey validates the configuration key format
func validateConfigKey(key string) error {
	if key == "" {
		return fmt.Errorf("key cannot be empty")
	}

	// Check for valid key format (alphanumeric, dots, underscores)
	for _, char := range key {
		if !isValidConfigKeyChar(char) {
			return fmt.Errorf("key contains invalid character '%c'", char)
		}
	}

	// Check for common valid keys
	validKeys := []string{
		"bend.base_url", "bend.rate_limit", "bend.timeout", "bend.session_file",
		"bend.refresh_token", "bend.device_hash", "bend.device_type", "bend.device_location",
	}

	isValid := false
	for _, validKey := range validKeys {
		if key == validKey {
			isValid = true
			break
		}
	}

	if !isValid {
		// Allow unknown keys but warn
		if IsVerbose() {
			fmt.Printf("Warning: Unknown configuration key '%s'\n", key)
		}
	}

	return nil
}

// validateConfigValue validates configuration values for known keys
func validateConfigValue(key, value string) error {
	switch key {
	case "bend.base_url":
		if !strings.HasPrefix(value, "http://") && !strings.HasPrefix(value, "https://") {
			return fmt.Errorf("base_url must be a valid HTTP/HTTPS URL")
		}
	case "bend.timeout":
		if !strings.HasSuffix(value, "s") && !strings.HasSuffix(value, "m") && !strings.HasSuffix(value, "h") {
			return fmt.Errorf("timeout must include unit (s, m, h)")
		}
	case "bend.rate_limit":
		if !strings.HasSuffix(value, "s") && !strings.HasSuffix(value, "ms") {
			return fmt.Errorf("rate_limit must include unit (s, ms)")
		}
	case "bend.device_type":
		validTypes := []string{"Web", "Mobile", "CLI"}
		isValid := false
		for _, validType := range validTypes {
			if value == validType {
				isValid = true
				break
			}
		}
		if !isValid {
			return fmt.Errorf("device_type must be one of: %s", strings.Join(validTypes, ", "))
		}
	}

	return nil
}

// isValidConfigKeyChar checks if a character is valid in a config key
func isValidConfigKeyChar(char rune) bool {
	return (char >= 'a' && char <= 'z') ||
		(char >= 'A' && char <= 'Z') ||
		(char >= '0' && char <= '9') ||
		char == '.' || char == '_' || char == '-'
}
