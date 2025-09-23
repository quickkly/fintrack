package blend

import (
	"encoding/json"
	"fmt"
	"strings"

	"fintrack/internal/blend"
	"fintrack/internal/config"

	"github.com/spf13/cobra"
)

var AccountsCmd = &cobra.Command{
	Use:   "accounts",
	Short: "List all connected accounts",
	Long: `List all bank accounts connected to your Bend profile.
Shows account details including balances, bank information, and recent activity.`,
	RunE: runAccounts,
}

var output string

func init() {
	AccountsCmd.Flags().StringVarP(&output, "output", "o", "table", "Output format (table, json, csv)")
}

func runAccounts(cmd *cobra.Command, args []string) error {
	cfg, err := config.GetFromContext(cmd)
	if err != nil {
		return fmt.Errorf("failed to get configuration: %w", err)
	}

	// Initialize session manager
	sessionManager := blend.NewSessionManager(cfg.Bend.SessionFile)

	// Check if session exists and is valid
	sessionInfo, err := sessionManager.GetSessionInfo()
	if err != nil {
		return fmt.Errorf("failed to get session info: %w", err)
	}
	if !sessionInfo.Exists {
		return fmt.Errorf("no session found. Run 'fintrack bend login' to authenticate")
	}

	// Load session
	session, err := sessionManager.LoadSession()
	if err != nil {
		return fmt.Errorf("failed to load session: %w", err)
	}

	// Check session validity
	if !sessionInfo.Valid {
		if sessionInfo.HasRefreshToken {
			fmt.Println("⚠️  Session expired, attempting to refresh...")
			// Create client to refresh session
			client := blend.NewClient(cfg)
			client.SetSession(session)
			if err := client.RefreshSession(); err != nil {
				return fmt.Errorf("session expired. Run 'fintrack bend check' to refresh or 'fintrack bend login' to re-authenticate")
			}
			// Save refreshed session
			if err := sessionManager.SaveSession(client.GetSession()); err != nil {
				return fmt.Errorf("failed to save refreshed session: %w", err)
			}
			session = client.GetSession()
		} else {
			return fmt.Errorf("session expired. Run 'fintrack bend check' to refresh or 'fintrack bend login' to re-authenticate")
		}
	}

	fmt.Println("🔄 Fetching accounts...")

	// Create client and get accounts
	client := blend.NewClient(cfg)
	client.SetSession(session)

	accounts, err := client.GetAccounts()
	if err != nil {
		return fmt.Errorf("failed to fetch accounts: %w", err)
	}

	if len(accounts) == 0 {
		fmt.Println("📭 No accounts found")
		return nil
	}

	fmt.Printf("\n📋 Found %d account(s):\n\n", len(accounts))

	switch output {
	case "table":
		fmt.Printf("%-36s | %-33s | %-19s | %-7s | %12s | %-16s\n",
			"ID", "Holder Name", "Bank", "Type", "Balance", "Last Updated")
		fmt.Printf("-------------------------------------+-----------------------------------+---------------------+---------+--------------+------------------\n")
		for _, account := range accounts {
			bankName := account.FinancialInformationProvider.Name
			if len(bankName) > 19 {
				bankName = bankName[:16] + "..."
			}

			holderName := account.HolderName
			if len(holderName) > 33 {
				holderName = holderName[:30] + "..."
			}

			lastUpdate := account.LastFetchedAt.Format("2006-01-02 15:04")

			fmt.Printf("%-36s | %-33s | %-19s | %-7s | %10.2f %s | %s\n",
				account.UUID, holderName, bankName, account.Type,
				account.CurrentBalance, account.Currency, lastUpdate)
		}

	case "json":
		jsonData, err := json.MarshalIndent(accounts, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal accounts to JSON: %w", err)
		}
		fmt.Println(string(jsonData))

	case "csv":
		fmt.Printf("ID,HolderName,Bank,Type,Balance,Currency,MaskedAccount,IFSC,LastUpdate\n")
		for _, account := range accounts {
			lastUpdate := account.LastFetchedAt.Format("2006-01-02T15:04:05Z")

			// Escape CSV fields if they contain commas
			holderName := strings.ReplaceAll(account.HolderName, ",", ";")
			bankName := strings.ReplaceAll(account.FinancialInformationProvider.Name, ",", ";")

			fmt.Printf("%s,%s,%s,%s,%.2f,%s,%s,%s,%s\n",
				account.UUID, holderName, bankName,
				account.Type, account.CurrentBalance, account.Currency,
				account.MaskedAccountNumber, account.IFSCCode, lastUpdate)
		}

	default:
		return fmt.Errorf("unsupported output format: %s. Use table, json, or csv", output)
	}

	fmt.Printf("\n💡 Use account ID with 'fintrack bend transactions --account-id <UUID>' to fetch transactions\n")

	return nil
}
