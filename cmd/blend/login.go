package blend

import (
	"fmt"

	"github.com/quickkly/fintrack/internal/blend"
	"github.com/quickkly/fintrack/internal/config"

	"github.com/spf13/cobra"
)

// LoginCmd represents the bend login command
var LoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with Bend",
	Long: `Interactive authentication with Bend financial service.
This command will set up authentication using a refresh token
and save the session for future use.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runLogin(cmd)
	},
}

var (
	email    string
	password string
)

func init() {
	LoginCmd.Flags().StringVar(&email, "email", "", "Email address for authentication")
	LoginCmd.Flags().StringVar(&password, "password", "", "Password (not recommended, use interactive mode)")
}

func runLogin(cmd *cobra.Command) error {
	cfg, err := config.GetFromContext(cmd)
	if err != nil {
		return fmt.Errorf("failed to get configuration: %w", err)
	}

	// Create client and session manager
	client := blend.NewClient(cfg)
	sessionManager := blend.NewSessionManager(cfg.Bend.SessionFile)

	// Check if session already exists and is valid
	sessionInfo, err := sessionManager.GetSessionInfo()
	if err == nil && sessionInfo.Exists && sessionInfo.Valid {
		fmt.Println("✅ Already authenticated with Bend")

		// Load and test session
		session, err := sessionManager.LoadSession()
		if err != nil {
			return fmt.Errorf("failed to load session: %w", err)
		}

		client.SetSession(session)
		userInfo, err := client.CheckSession()
		if err == nil {
			fmt.Printf("👤 Logged in as: %s (%s)\n", userInfo.GetFullName(), userInfo.Email)
			fmt.Println("Use 'fintrack bend check' to see session details")
			return nil
		}
	}

	fmt.Println("🔐 Bend Authentication")
	fmt.Println("============================")

	// Check if refresh token is available in config
	if cfg.Bend.RefreshToken != "" {
		fmt.Println("🔄 Using refresh token from configuration...")

		// Initialize session from refresh token
		if err := client.InitializeFromRefreshToken(cfg.Bend.RefreshToken); err != nil {
			return fmt.Errorf("failed to initialize from refresh token: %w", err)
		}

		// Save session
		session := client.GetSession()
		if err := sessionManager.SaveSession(session); err != nil {
			return fmt.Errorf("failed to save session: %w", err)
		}

		fmt.Println("✅ Authentication successful!")
		fmt.Printf("💾 Session saved to: %s\n", cfg.Bend.SessionFile)
		fmt.Printf("⏰ Token expires: %s\n", session.ExpiresAt.Format("2006-01-02 15:04:05"))

		// Test the session
		_, err = client.CheckSession()
		if err != nil {
			fmt.Printf("⚠️  Warning: Session verification failed: %v\n", err)
		} else {
			fmt.Printf("👤 Authenticated successfully\n")
		}

		fmt.Println("\nNext steps:")
		fmt.Println("- Check accounts: fintrack bend accounts")
		fmt.Println("- Fetch transactions: fintrack bend transactions")

		return nil
	}

	// Fallback to manual token input
	fmt.Println("No refresh token found in configuration.")
	fmt.Println("Please add your refresh token to the config file:")
	fmt.Printf("  bend.refresh_token: \"your-refresh-token-here\"\n")
	fmt.Println("\nAlternatively, you can set it using:")
	fmt.Println("  fintrack config set bend.refresh_token \"your-refresh-token\"")

	return fmt.Errorf("refresh token required for authentication")
}
