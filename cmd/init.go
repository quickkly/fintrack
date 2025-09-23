package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// =============================================================================
// INIT COMMAND DEFINITION
// =============================================================================

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init [directory]",
	Short: "Initialize fintrack in a directory",
	Long: `Initialize fintrack in the current directory or specified directory.

This command will:
- Create .fintrack directory in the target location
- Create default config.yaml file with sensible defaults
- Create .fintrackignore file for excluding files
- Set up proper file permissions

The configuration will be created with default values suitable for most users.
You can customize it later using 'fintrack config set' commands.

Examples:
  fintrack init                    # Initialize in current directory
  fintrack init /path/to/project  # Initialize in specified directory`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		targetDir := "."
		if len(args) > 0 {
			targetDir = args[0]
		}
		force, _ := cmd.Flags().GetBool("force")
		return runInit(targetDir, force)
	},
}

func init() {
	initCmd.Flags().BoolP("force", "f", false, "Force initialization even if .fintrack directory already exists")
}

func runInit(targetDir string, force bool) error {
	if !IsQuiet() {
		fmt.Printf("🚀 Initializing fintrack in: %s\n", targetDir)
	}

	// Step 1: Validate and prepare target directory
	if err := validateTargetDirectory(targetDir); err != nil {
		return err
	}

	// Step 2: Ensure .fintrack directory exists
	if err := ensureLocalConfigDirectory(targetDir, force); err != nil {
		return err
	}

	// Step 3: Create default config file
	if err := createLocalConfigFile(targetDir, force); err != nil {
		return err
	}

	// Step 4: Create .fintrackignore file
	if err := createFintrackIgnoreFile(targetDir, force); err != nil {
		return err
	}

	// Step 5: Display success message and next steps
	displayLocalInitSuccess(targetDir)

	return nil
}

// validateTargetDirectory validates the target directory
func validateTargetDirectory(targetDir string) error {
	// Resolve absolute path
	absPath, err := filepath.Abs(targetDir)
	if err != nil {
		return fmt.Errorf("failed to resolve directory path: %w", err)
	}

	// Check if directory exists
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return fmt.Errorf("directory does not exist: %s", absPath)
	}

	if IsVerbose() {
		fmt.Printf("✓ Target directory validated: %s\n", absPath)
	}

	return nil
}

// ensureLocalConfigDirectory creates the .fintrack directory
func ensureLocalConfigDirectory(targetDir string, force bool) error {
	fintrackDir := filepath.Join(targetDir, ".fintrack")

	// Check if .fintrack already exists
	if _, err := os.Stat(fintrackDir); err == nil {
		if !force {
			return fmt.Errorf(".fintrack directory already exists in %s. Use --force to overwrite", targetDir)
		}
		if IsVerbose() {
			fmt.Printf("⚠️  .fintrack directory exists, overwriting due to --force flag\n")
		}
	}

	if err := os.MkdirAll(fintrackDir, 0755); err != nil {
		return fmt.Errorf("failed to create .fintrack directory: %w", err)
	}

	if IsVerbose() {
		fmt.Printf("✓ .fintrack directory created: %s\n", fintrackDir)
	}

	return nil
}

// createLocalConfigFile creates the local configuration file
func createLocalConfigFile(targetDir string, force bool) error {
	configPath := filepath.Join(targetDir, ".fintrack", "config.yaml")

	// Check if config file already exists
	if _, err := os.Stat(configPath); err == nil {
		if !force {
			if !IsQuiet() {
				fmt.Printf("ℹ️  Config file already exists at %s\n", configPath)
				fmt.Printf("   Use 'fintrack config show' to view current settings\n")
				fmt.Printf("   Use 'fintrack config set' to modify specific values\n")
			}
			return nil
		}
	}

	// Get absolute path for session file
	absTargetDir, _ := filepath.Abs(targetDir)
	sessionFile := filepath.Join(absTargetDir, ".fintrack", "session.json")

	// Create default configuration content
	defaultConfig := generateLocalDefaultConfig(sessionFile)

	// Write configuration file with proper permissions
	if err := os.WriteFile(configPath, []byte(defaultConfig), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	if IsVerbose() {
		fmt.Printf("✓ Config file created: %s\n", configPath)
	}

	return nil
}

// createFintrackIgnoreFile creates a .fintrackignore file
func createFintrackIgnoreFile(targetDir string, force bool) error {
	ignorePath := filepath.Join(targetDir, ".fintrackignore")

	// Check if .fintrackignore already exists
	if _, err := os.Stat(ignorePath); err == nil {
		if !force {
			if IsVerbose() {
				fmt.Printf("ℹ️  .fintrackignore already exists, skipping\n")
			}
			return nil
		}
	}

	// Create default .fintrackignore content
	defaultIgnore := generateDefaultFintrackIgnore()

	// Write .fintrackignore file
	if err := os.WriteFile(ignorePath, []byte(defaultIgnore), 0644); err != nil {
		return fmt.Errorf("failed to write .fintrackignore file: %w", err)
	}

	if IsVerbose() {
		fmt.Printf("✓ .fintrackignore created: %s\n", ignorePath)
	}

	return nil
}

// displayLocalInitSuccess shows the success message and next steps
func displayLocalInitSuccess(targetDir string) {
	if IsQuiet() {
		return
	}

	absPath, _ := filepath.Abs(targetDir)
	configPath := filepath.Join(absPath, ".fintrack", "config.yaml")

	fmt.Printf("\n✅ FinTrack initialized successfully in: %s\n", absPath)
	fmt.Printf("📁 Config directory: %s\n", filepath.Dir(configPath))
	fmt.Printf("⚙️  Config file: %s\n", configPath)
	fmt.Printf("🚫 Ignore file: %s\n", filepath.Join(absPath, ".fintrackignore"))

	fmt.Printf("\n🎯 Next steps:\n")
	fmt.Printf("1. Set up your Bend credentials:\n")
	fmt.Printf("   fintrack bend login\n")
	fmt.Printf("\n2. Test the setup:\n")
	fmt.Printf("   fintrack bend check\n")
	fmt.Printf("\n3. View your configuration:\n")
	fmt.Printf("   fintrack config show\n")
	fmt.Printf("\n4. Customize settings (optional):\n")
	fmt.Printf("   fintrack config set bend.device_type \"CLI\"\n")
	fmt.Printf("   fintrack config set bend.timeout \"60s\"\n")
	fmt.Printf("\n5. Start tracking transactions:\n")
	fmt.Printf("   fintrack bend transactions --help\n")
}

// generateLocalDefaultConfig creates the default configuration YAML content for local setup
func generateLocalDefaultConfig(sessionFile string) string {
	return fmt.Sprintf(`# FinTrack Configuration
# This file contains settings for the FinTrack CLI tool
# This is a local configuration file for this project

# Bend Financial Service Configuration
bend:
  # Bend base URL
  base_url: "https://bend.example.com"
  
  # Rate limiting (requests per second)
  rate_limit: "1s"
  
  # Session file location (local to this project)
  session_file: "%s"
  
  # Request timeout
  timeout: "30s"
  
  # Device configuration (required by Bend)
  # device_hash: ""                                     # Will be auto-generated if not provided
  device_type: "Web"                                    # Device type: Web, Mobile, CLI
  device_location: "Default"                            # Device location
  
  # Authentication (set this via 'fintrack bend login')
  # refresh_token: "your-refresh-token-here"

# Configuration notes:
# - This is a local configuration file for this project
# - Modify device_type to "CLI" for better identification
# - Adjust timeout based on your network conditions
# - The refresh_token will be set automatically during login
# - Session data is stored locally in this project
`, sessionFile)
}

// generateDefaultFintrackIgnore creates the default .fintrackignore content
func generateDefaultFintrackIgnore() string {
	return `# FinTrack Ignore File
# Files and directories to exclude from FinTrack operations

# Common financial data files
*.csv
*.xlsx
*.xls
*.pdf
*.json

# Backup files
*.bak
*.backup
*~

# Temporary files
*.tmp
*.temp

# OS generated files
.DS_Store
.DS_Store?
._*
.Spotlight-V100
.Trashes
ehthumbs.db
Thumbs.db

# IDE files
.vscode/
.idea/
*.swp
*.swo

# Log files
*.log

# Sensitive data (customize as needed)
secrets/
private/
confidential/
`
}
