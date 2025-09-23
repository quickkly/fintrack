package blend

import (
	"fmt"
	"time"

	"fintrack/internal/blend"
	"fintrack/internal/config"

	"github.com/spf13/cobra"
)

// CheckCmd represents the bend check command
var CheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Check Bend session status",
	Long: `Check the current session status with Bend financial service.
This command will verify if your authentication token is valid and show session information.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runCheck(cmd)
	},
}

func runCheck(cmd *cobra.Command) error {
	cfg, err := config.GetFromContext(cmd)
	if err != nil {
		return fmt.Errorf("failed to get configuration: %w", err)
	}

	// Create session manager
	sessionManager := blend.NewSessionManager(cfg.Bend.SessionFile)

	// Get session info
	sessionInfo, err := sessionManager.GetSessionInfo()
	if err != nil {
		return fmt.Errorf("failed to get session info: %w", err)
	}

	// Display session status
	fmt.Println("Bend Session Status:")
	fmt.Println("=========================")

	if !sessionInfo.Exists {
		fmt.Println("❌ No session file found")
		fmt.Println("Run 'fintrack bend login' to authenticate")
		return nil
	}

	fmt.Printf("📁 Session file: %s\n", cfg.Bend.SessionFile)

	if !sessionInfo.Valid {
		fmt.Println("❌ Session expired or invalid")
		if sessionInfo.HasRefreshToken {
			fmt.Println("💡 Trying to refresh session...")
			return refreshSession(cfg, sessionManager)
		} else {
			fmt.Println("Run 'fintrack bend login' to re-authenticate")
			return nil
		}
	}

	fmt.Println("✅ Session is valid")
	fmt.Printf("⏰ Expires: %s\n", sessionInfo.ExpiresAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("⏳ Time remaining: %s\n", sessionInfo.TimeRemaining.Round(time.Minute))

	if sessionInfo.HasRefreshToken {
		fmt.Println("🔄 Refresh token available")
	}

	// Test API connection
	fmt.Println("\nTesting API connection...")
	client := blend.NewClient(cfg)

	session, err := sessionManager.LoadSession()
	if err != nil {
		return fmt.Errorf("failed to load session: %w", err)
	}

	client.SetSession(session)

	userInfo, err := client.CheckSession()
	if err != nil {
		fmt.Printf("❌ API test failed: %v\n", err)
		if sessionInfo.HasRefreshToken {
			fmt.Println("💡 Trying to refresh session...")
			return refreshSession(cfg, sessionManager)
		}
		return fmt.Errorf("API connection failed")
	}

	fmt.Println("✅ API connection successful")
	fmt.Printf("👤 User: %s (%s)\n", userInfo.GetFullName(), userInfo.Email)
	fmt.Printf("🆔 ID: %s\n", userInfo.UUID)
	fmt.Printf("📱 Phone: %s\n", userInfo.Phone)
	fmt.Printf("🌍 Timezone: %s\n", userInfo.Timezone)
	fmt.Printf("👑 Role: %s\n", userInfo.Role)

	if userInfo.EmailVerified {
		fmt.Println("✅ Email verified")
	} else {
		fmt.Println("⚠️  Email not verified")
	}

	if userInfo.PhoneVerified {
		fmt.Println("✅ Phone verified")
	} else {
		fmt.Println("⚠️  Phone not verified")
	}

	if userInfo.BetaAccess {
		fmt.Println("🧪 Beta access enabled")
	}

	if userInfo.GoogleLinked {
		fmt.Println("🔗 Google account linked")
	}

	if userInfo.AppleLinked {
		fmt.Println("🍎 Apple account linked")
	}

	return nil
}

func refreshSession(cfg *config.Config, sessionManager *blend.SessionManager) error {
	client := blend.NewClient(cfg)

	session, err := sessionManager.LoadSession()
	if err != nil {
		return fmt.Errorf("failed to load session: %w", err)
	}

	client.SetSession(session)

	if err := client.RefreshSession(); err != nil {
		fmt.Printf("❌ Session refresh failed: %v\n", err)
		fmt.Println("Run 'fintrack bend login' to re-authenticate")
		return err
	}

	// Save updated session
	if err := sessionManager.SaveSession(client.GetSession()); err != nil {
		return fmt.Errorf("failed to save refreshed session: %w", err)
	}

	fmt.Println("✅ Session refreshed successfully")

	// Test the refreshed session
	userInfo, err := client.CheckSession()
	if err != nil {
		return fmt.Errorf("refreshed session test failed: %w", err)
	}

	fmt.Printf("👤 User: %s (%s)\n", userInfo.GetFullName(), userInfo.Email)
	return nil
}
