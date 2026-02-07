package blend

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/quickkly/fintrack/internal/blend"
	"github.com/quickkly/fintrack/internal/config"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// LoginCmd represents the bend login command
var LoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with Bend",
	Long: `Authenticate with Bend financial service using either:
1. Refresh token (from config file)
2. OTP-based authentication (interactive)

OTP Mode:
  Use --otp-mode or --phone to enable OTP-based authentication.
  This will send an OTP to your phone number and prompt you to enter it.
  After successful verification, it will automatically update your config
  with device_hash and refresh_token, then initialize the session.

Refresh Token Mode:
  If a refresh_token is already configured, it will be used automatically.
  Otherwise, you'll be prompted to configure one.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runLogin(cmd)
	},
}

var (
	email    string
	password string
	phone    string
	otp      string
	useOTP   bool
)

func init() {
	LoginCmd.Flags().StringVar(&email, "email", "", "Email address for authentication (legacy)")
	LoginCmd.Flags().StringVar(&password, "password", "", "Password (not recommended, use interactive mode)")
	LoginCmd.Flags().StringVar(&phone, "phone", "", "Phone number for OTP-based authentication (e.g., +1234567890)")
	LoginCmd.Flags().StringVar(&otp, "otp", "", "OTP code for verification (if not provided, will prompt interactively)")
	LoginCmd.Flags().BoolVar(&useOTP, "otp-mode", false, "Use OTP-based authentication instead of refresh token")
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

	// OTP-based authentication flow
	if useOTP || phone != "" {
		return runOTPLogin(cmd, cfg, client, sessionManager)
	}

	// Check if refresh token is available in config
	if cfg.Bend.RefreshToken != "" {
		return runLoginWithRefreshToken(cmd, cfg)
	}

	// Fallback to manual token input
	fmt.Println("No refresh token found in configuration.")
	fmt.Println("Please add your refresh token to the config file:")
	fmt.Printf("  bend.refresh_token: \"your-refresh-token-here\"\n")
	fmt.Println("\nAlternatively, you can set it using:")
	fmt.Println("  fintrack config set bend.refresh_token \"your-refresh-token\"")
	fmt.Println("\nOr use OTP-based login:")
	fmt.Println("  fintrack bend login --otp-mode --phone +1234567890")

	return fmt.Errorf("refresh token required for authentication")
}

// runOTPLogin handles OTP-based authentication flow
func runOTPLogin(cmd *cobra.Command, cfg *config.Config, client *blend.Client, sessionManager *blend.SessionManager) error {
	// Enable logging for debugging OTP flow
	client.SetLogging(true)

	// Get phone number
	if phone == "" {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Enter phone number (e.g., +1234567890): ")
		phoneInput, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read phone number: %w", err)
		}
		phone = strings.TrimSpace(phoneInput)
	}

	if phone == "" {
		return fmt.Errorf("phone number is required")
	}

	// Generate request ID and device hash (must be same for both OTP and verify)
	// We'll generate these using the client's internal methods
	requestID := generateRequestIDForOTP()
	deviceHash := generateDeviceHashForOTP()

	fmt.Printf("📱 Requesting OTP for %s...\n", phone)
	fmt.Printf("🔑 Using Request ID: %s\n", requestID)
	fmt.Printf("📱 Using Device Hash: %s\n", deviceHash)

	// IMPORTANT: Set device hash BEFORE requesting OTP (must be same for both calls)
	originalDeviceHash := client.GetDeviceHash()
	client.SetDeviceHash(deviceHash)

	// Request OTP
	if err := client.RequestOTP(phone, "sms", requestID); err != nil {
		client.SetDeviceHash(originalDeviceHash)
		return fmt.Errorf("failed to request OTP: %w", err)
	}

	fmt.Println("✅ OTP sent successfully!")

	// Get OTP from user
	otpCode := otp
	if otpCode == "" {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Enter OTP code: ")
		otpInput, err := reader.ReadString('\n')
		if err != nil {
			client.SetDeviceHash(originalDeviceHash)
			return fmt.Errorf("failed to read OTP: %w", err)
		}
		otpCode = strings.TrimSpace(otpInput)
	}

	if otpCode == "" {
		client.SetDeviceHash(originalDeviceHash)
		return fmt.Errorf("OTP code is required")
	}

	fmt.Println("🔐 Verifying OTP...")

	// Verify OTP - device hash is already set from RequestOTP call above

	verifyData, marbleCookie, err := client.VerifyOTP(phone, otpCode, requestID)
	if err != nil {
		client.SetDeviceHash(originalDeviceHash)
		return fmt.Errorf("failed to verify OTP: %w", err)
	}

	// Note: marbleCookie is extracted but not currently used in session
	// It may be needed for future API calls
	_ = marbleCookie

	// Restore original device hash
	client.SetDeviceHash(originalDeviceHash)

	fmt.Println("✅ OTP verified successfully!")

	// Update config with device_hash and refresh_token
	fmt.Println("💾 Updating configuration...")
	if err := updateConfigWithTokens(cfg, deviceHash, verifyData.RefreshToken); err != nil {
		return fmt.Errorf("failed to update config: %w", err)
	}

	fmt.Printf("✅ Configuration updated with device_hash and refresh_token\n")

	// Reload config from file to get updated values
	reloadedCfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("failed to reload configuration: %w", err)
	}

	// Update context with reloaded config
	config.SetInContext(cmd, reloadedCfg)

	// Now call the normal login flow which will use the refresh token from config
	fmt.Println("🔄 Initializing session using refresh token from config...")
	return runLoginWithRefreshToken(cmd, reloadedCfg)
}

// runLoginWithRefreshToken handles the refresh token login flow (extracted from runLogin)
func runLoginWithRefreshToken(cmd *cobra.Command, cfg *config.Config) error {
	// Create client and session manager
	client := blend.NewClient(cfg)
	sessionManager := blend.NewSessionManager(cfg.Bend.SessionFile)

	if cfg.Bend.RefreshToken == "" {
		return fmt.Errorf("refresh token not found in configuration")
	}

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
	_, err := client.CheckSession()
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

// generateRequestIDForOTP generates a UUID-like request ID for OTP flow
func generateRequestIDForOTP() string {
	// Generate a UUID v4 format ID (same format as device hash)
	return blend.GenerateDeviceHash()
}

// generateDeviceHashForOTP generates a device hash for OTP flow
func generateDeviceHashForOTP() string {
	return blend.GenerateDeviceHash()
}

// updateConfigWithTokens updates the config file with device_hash and refresh_token
func updateConfigWithTokens(cfg *config.Config, deviceHash, refreshToken string) error {
	v := viper.New()

	// Set config file path
	configFile := ""
	if cfgFile := os.Getenv("FINTRACK_CONFIG"); cfgFile != "" {
		configFile = cfgFile
	} else {
		// Use the same search paths as config loading
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(".fintrack")
		v.AddConfigPath("./configs")
		v.AddConfigPath(".")

		homeDir, err := os.UserHomeDir()
		if err == nil {
			globalConfigDir := fmt.Sprintf("%s/.config/fintrack", homeDir)
			v.AddConfigPath(globalConfigDir)
		}
	}

	if configFile != "" {
		v.SetConfigFile(configFile)
	}

	// Read existing config
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("failed to read config: %w", err)
		}
		// Config file doesn't exist, we'll create it
	}

	// Set the values
	v.Set("bend.device_hash", deviceHash)
	v.Set("bend.refresh_token", refreshToken)

	// Write config
	if err := v.WriteConfig(); err != nil {
		// If config file doesn't exist, try to create it
		configPath := v.ConfigFileUsed()
		if configPath == "" {
			// Determine where to create the config file
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("failed to get home directory: %w", err)
			}
			configDir := fmt.Sprintf("%s/.config/fintrack", homeDir)
			if err := os.MkdirAll(configDir, 0755); err != nil {
				return fmt.Errorf("failed to create config directory: %w", err)
			}
			configPath = fmt.Sprintf("%s/config.yaml", configDir)
			v.SetConfigFile(configPath)
		}
		if err := v.WriteConfig(); err != nil {
			return fmt.Errorf("failed to write config: %w", err)
		}
	}

	return nil
}
