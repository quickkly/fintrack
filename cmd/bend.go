package cmd

import (
	"github.com/quickkly/fintrack/cmd/blend"

	"github.com/spf13/cobra"
)

// =============================================================================
// BEND COMMAND DEFINITION
// =============================================================================

// bendCmd represents the bend command
var bendCmd = &cobra.Command{
	Use:   "bend",
	Short: "Bend operations",
	Long: `Commands for interacting with financial data via Bend.

Available operations:
- check: Check session status and validity
- login: Interactive authentication setup with refresh token
- accounts: List all connected bank accounts
- transactions: Fetch transaction data with advanced filtering options

Examples:
  fintrack bend check                    # Check if session is valid
  fintrack bend login                    # Set up authentication
  fintrack bend accounts                 # List all accounts
  fintrack bend transactions --days 7    # Fetch last 7 days of transactions`,
}

func init() {
	setupBendSubcommands()
}

// setupBendSubcommands adds all bend subcommands
func setupBendSubcommands() {
	bendCmd.AddCommand(blend.CheckCmd)
	bendCmd.AddCommand(blend.LoginCmd)
	bendCmd.AddCommand(blend.AccountsCmd)
	bendCmd.AddCommand(blend.TransactionsCmd)
}
