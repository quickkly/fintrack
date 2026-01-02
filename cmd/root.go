package cmd

import (
	"fmt"
	"os"

	"github.com/quickkly/fintrack/internal/config"

	"github.com/spf13/cobra"
)

// Global flags - moved to top for clarity
var (
	cfgFile string
	verbose bool
	dryRun  bool
	quiet   bool
	logHTTP bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "fintrack",
	Short: "Finance tracking CLI tool with advanced filtering capabilities",
	Long: `FinTrack is a CLI tool for financial transaction tracking and management with advanced filtering capabilities.

The tool provides:
- Fetching transaction data from financial APIs
- Advanced filtering and sorting options
- Session management and authentication
- Configuration management
- HTTP request/response logging

For more information, visit the project documentation.`,
	PersistentPreRunE: setupRootCommand,
	SilenceUsage:      true, // Don't show usage on errors
	SilenceErrors:     true, // Don't show errors twice
}

// setupRootCommand initializes the root command and loads configuration
func setupRootCommand(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Validate configuration
	if err := validateConfiguration(cfg); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Store configuration in command context
	config.SetInContext(cmd, cfg)

	// Set up logging based on flags
	setupLogging()

	return nil
}

// validateConfiguration performs basic validation of the loaded configuration
func validateConfiguration(cfg *config.Config) error {
	if cfg == nil {
		return fmt.Errorf("configuration is nil")
	}

	// Validate required fields
	if cfg.Bend.BaseURL == "" {
		return fmt.Errorf("bend.base_url is required")
	}

	if cfg.Bend.Timeout <= 0 {
		return fmt.Errorf("bend.timeout must be positive")
	}

	return nil
}

// setupLogging configures logging based on global flags
func setupLogging() {
	// This could be expanded to set up structured logging
	// For now, we just use the global flags
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	setupGlobalFlags()
	setupSubcommands()
}

// setupGlobalFlags configures all global flags
func setupGlobalFlags() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ~/.config/fintrack/config.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "show what would happen without executing")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "suppress output except errors")
	rootCmd.PersistentFlags().BoolVar(&logHTTP, "log-http", false, "enable HTTP request/response logging")

	// Mark config flag as deprecated in favor of environment variable
	rootCmd.PersistentFlags().MarkDeprecated("config", "use FINTRACK_CONFIG environment variable instead")
}

// setupSubcommands adds all subcommands to the root command
func setupSubcommands() {
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(bendCmd)
}

// =============================================================================
// GLOBAL FLAG ACCESSORS
// =============================================================================

// IsVerbose returns whether verbose mode is enabled
func IsVerbose() bool {
	return verbose
}

// IsDryRun returns whether dry-run mode is enabled
func IsDryRun() bool {
	return dryRun
}

// IsQuiet returns whether quiet mode is enabled
func IsQuiet() bool {
	return quiet
}

// IsHTTPLoggingEnabled returns whether HTTP logging is enabled
func IsHTTPLoggingEnabled() bool {
	return logHTTP
}
