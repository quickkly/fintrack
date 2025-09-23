package blend

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"fintrack/internal/blend"
	"fintrack/internal/config"

	"github.com/spf13/cobra"
)

// TransactionsCmd represents the blend transactions command
var TransactionsCmd = &cobra.Command{
	Use:   "transactions",
	Short: "Fetch transactions from Bend with basic and advanced filtering",
	Long: `Fetch transaction data from Bend with comprehensive filtering options.

Basic filtering:
- Date ranges (YYYY-MM-DD or RFC3339 format)
- Account filtering
- Time-based filters (this_month, last_month, etc.)

Advanced filtering (matching curl parameters):
- Category and subcategory filtering
- Custom sorting (amount, txn_timestamp, etc.)
- Detailed search summaries
- OR logic for category/subcategory combinations
- Aggregated totals and counts

Data is saved to the staging directory for further processing.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runTransactions(cmd)
	},
}

var (
	fromDate      string
	toDate        string
	accountID     string
	days          int
	stagingDir    string
	timeFilter    string
	countBy       string
	includeTotals bool

	// Advanced filtering options
	categoryID      string
	subcategoryID   string
	sortBy          string
	sortOrder       string
	includeDetailed bool
	orCategory      bool

	// Debug options
	enableLogging bool
)

func init() {
	// Basic filtering options
	TransactionsCmd.Flags().StringVar(&fromDate, "from", "", "Start date (YYYY-MM-DD or RFC3339 format). If only --from is provided, fetches from that date to now")
	TransactionsCmd.Flags().StringVar(&toDate, "to", "", "End date (YYYY-MM-DD or RFC3339 format). If only --to is provided, fetches --days back from that date")
	TransactionsCmd.Flags().StringVar(&accountID, "account-id", "", "Specific account UUID")
	TransactionsCmd.Flags().IntVar(&days, "days", 30, "Number of days to fetch (default: 30, used when dates not fully specified)")

	TransactionsCmd.Flags().StringVar(&stagingDir, "staging-dir", "", "Staging directory (default: from config)")
	TransactionsCmd.Flags().StringVar(&timeFilter, "time-filter", "", "Predefined time filter (this_month, last_month, this_year, etc.)")
	TransactionsCmd.Flags().StringVar(&countBy, "count-by", "", "Aggregation period (month, week, day)")
	TransactionsCmd.Flags().BoolVar(&includeTotals, "include-totals", false, "Include aggregated totals in response")

	// Advanced filtering options
	TransactionsCmd.Flags().StringVar(&categoryID, "category-id", "", "Filter by category ID")
	TransactionsCmd.Flags().StringVar(&subcategoryID, "subcategory-id", "", "Filter by subcategory ID")
	TransactionsCmd.Flags().StringVar(&sortBy, "sort-by", "txn_timestamp", "Sort field (default: txn_timestamp)")
	TransactionsCmd.Flags().StringVar(&sortOrder, "sort-order", "DESC", "Sort order (ASC/DESC, default: DESC)")
	TransactionsCmd.Flags().BoolVar(&includeDetailed, "include-detailed", false, "Include detailed search summary")
	TransactionsCmd.Flags().BoolVar(&orCategory, "or-category", false, "Use OR logic for category/subcategory filtering")

	// Debug options
	TransactionsCmd.Flags().BoolVar(&enableLogging, "log-http", false, "Enable HTTP request/response logging")
}

func runTransactions(cmd *cobra.Command) error {
	cfg, err := config.GetFromContext(cmd)
	if err != nil {
		return fmt.Errorf("failed to get configuration: %w", err)
	}

	// Setup client and session
	client, _, err := setupClientAndSession(cfg)
	if err != nil {
		return err
	}

	// Parse date range
	from, to, err := parseDateRange(fromDate, toDate, days)
	if err != nil {
		return err
	}

	fmt.Printf("🔄 Fetching transactions from %s to %s\n",
		from.Format("2006-01-02"), to.Format("2006-01-02"))

	// Setup staging directory
	stagingDir, err := setupStagingDirectory(stagingDir)
	if err != nil {
		return err
	}

	// Get user ID
	userID, err := client.GetUserID()
	if err != nil {
		return fmt.Errorf("failed to get user ID: %w", err)
	}

	fmt.Printf("👤 Fetching transactions for user: %s\n", userID)

	// Prepare filters
	filters := prepareTransactionFilters(from, to, countBy, timeFilter, sortBy, sortOrder,
		accountID, categoryID, subcategoryID, includeTotals, includeDetailed, orCategory)

	// Check if using advanced filtering
	hasAdvancedOptions := hasAdvancedFilteringOptions(timeFilter, accountID, categoryID, subcategoryID,
		sortBy, sortOrder, includeDetailed, orCategory)

	if hasAdvancedOptions {
		return handleAdvancedTransactions(client, userID, filters, stagingDir, from, to)
	}

	return handleBasicTransactions(client, userID, filters, stagingDir, from, to)
}

// setupClientAndSession initializes the client and validates the session
func setupClientAndSession(cfg *config.Config) (*blend.Client, *blend.Session, error) {
	client := blend.NewClient(cfg)
	client.SetLogging(enableLogging)

	sessionManager := blend.NewSessionManager(cfg.Bend.SessionFile)

	session, err := sessionManager.LoadSession()
	if err != nil {
		return nil, nil, fmt.Errorf("no session found. Run 'fintrack bend login' first")
	}

	sessionInfo, err := sessionManager.GetSessionInfo()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get session info: %w", err)
	}

	if !sessionInfo.Valid {
		fmt.Println("Session expired, attempting to refresh...")
		client.SetSession(session)
		if err := client.RefreshSession(); err != nil {
			return nil, nil, fmt.Errorf("session refresh failed. Run 'fintrack bend login' to re-authenticate: %w", err)
		}
		if err := sessionManager.SaveSession(client.GetSession()); err != nil {
			return nil, nil, fmt.Errorf("failed to save refreshed session: %w", err)
		}
		fmt.Println("✅ Session refreshed")
	} else {
		client.SetSession(session)
	}

	return client, session, nil
}

// parseDateRange handles all date parsing logic with support for multiple formats
func parseDateRange(fromDate, toDate string, days int) (from, to time.Time, err error) {
	parseDate := func(dateStr string, fieldName string) (time.Time, error) {
		// Try RFC3339 format first (for advanced usage)
		if t, err := time.Parse(time.RFC3339, dateStr); err == nil {
			return t, nil
		}
		// Try alternative RFC3339 format
		if t, err := time.Parse("2006-01-02T15:04:05Z", dateStr); err == nil {
			return t, nil
		}
		// Try YYYY-MM-DD format (for basic usage)
		if t, err := time.Parse("2006-01-02", dateStr); err == nil {
			return t, nil
		}
		return time.Time{}, fmt.Errorf("invalid %s date format (use YYYY-MM-DD or RFC3339): %s", fieldName, dateStr)
	}

	// Handle different date input scenarios
	if fromDate != "" && toDate != "" {
		// Both dates provided
		from, err = parseDate(fromDate, "from")
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
		to, err = parseDate(toDate, "to")
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
		// Ensure from is before to
		if from.After(to) {
			return time.Time{}, time.Time{}, fmt.Errorf("from date (%s) cannot be after to date (%s)", fromDate, toDate)
		}
	} else if fromDate != "" {
		// Only from date provided, use from date to now
		from, err = parseDate(fromDate, "from")
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
		to = time.Now()
	} else if toDate != "" {
		// Only to date provided, use days parameter back from to date
		to, err = parseDate(toDate, "to")
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
		from = to.AddDate(0, 0, -days)
	} else {
		// No dates provided, use days parameter from now
		to = time.Now()
		from = to.AddDate(0, 0, -days)
	}

	return from, to, nil
}

// setupStagingDirectory ensures the staging directory exists
func setupStagingDirectory(stagingDir string) (string, error) {
	if stagingDir == "" {
		stagingDir = "./staging"
	}

	if err := os.MkdirAll(stagingDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create staging directory: %w", err)
	}

	return stagingDir, nil
}

// prepareTransactionFilters creates the transaction filters struct
func prepareTransactionFilters(from, to time.Time, countBy, timeFilter, sortBy, sortOrder,
	accountID, categoryID, subcategoryID string, includeTotals, includeDetailed, orCategory bool) blend.TransactionFilters {
	return blend.TransactionFilters{
		Limit:           50, // Default limit per page
		CountBy:         countBy,
		TimeFilter:      timeFilter,
		SortBy:          sortBy,
		SortOrder:       sortOrder,
		StartDate:       from,
		EndDate:         to,
		AccountID:       accountID,
		CategoryID:      categoryID,
		SubcategoryID:   subcategoryID,
		IncludeCountBy:  includeTotals,
		IncludeDetailed: includeDetailed,
		OrCategory:      orCategory,
	}
}

// hasAdvancedFilteringOptions checks if any advanced filtering is being used
func hasAdvancedFilteringOptions(timeFilter, accountID, categoryID, subcategoryID,
	sortBy, sortOrder string, includeDetailed, orCategory bool) bool {
	return timeFilter != "" || accountID != "" || categoryID != "" || subcategoryID != "" ||
		sortBy != "txn_timestamp" || sortOrder != "DESC" || includeDetailed || orCategory
}

// handleAdvancedTransactions processes transactions with advanced filtering
func handleAdvancedTransactions(client *blend.Client, userID string, filters blend.TransactionFilters,
	stagingDir string, from, to time.Time) error {

	// Log advanced filtering options
	logAdvancedFilteringOptions(filters)

	data, err := client.FetchTransactionsWithFilters(userID, filters)
	if err != nil {
		return fmt.Errorf("failed to fetch transactions with filters: %w", err)
	}

	if len(data.Transactions) == 0 {
		fmt.Println("📭 No transactions found")
		return nil
	}

	// Display summary
	fmt.Printf("📊 Found %d transactions (Total in API: %d)\n", len(data.Transactions), data.Total)

	// Generate filename and save
	filename := generateAdvancedFilename(filters)
	filepath := filepath.Join(stagingDir, filename)

	if err := saveTransactionsV3(filepath, data.Transactions, data.Counts, from, to); err != nil {
		return fmt.Errorf("failed to save transactions: %w", err)
	}

	fmt.Printf("✅ Saved %d transactions to %s\n", len(data.Transactions), filename)

	// Display counts if available
	if len(data.Counts) > 0 {
		displayTransactionCounts(data.Counts)
	}

	fmt.Printf("📁 Staging directory: %s\n", stagingDir)
	return nil
}

// handleBasicTransactions processes transactions with basic filtering
func handleBasicTransactions(client *blend.Client, userID string, filters blend.TransactionFilters,
	stagingDir string, from, to time.Time) error {

	// Use the standard v3 transactions API with pagination
	// If account filtering is specified, use API filtering instead of local filtering
	if filters.AccountID != "" {
		fmt.Printf("🏦 Account filter: %s\n", filters.AccountID)
		data, err := client.FetchTransactionsWithFilters(userID, filters)
		if err != nil {
			return fmt.Errorf("failed to fetch transactions with account filter: %w", err)
		}

		if len(data.Transactions) == 0 {
			fmt.Println("📭 No transactions found")
			return nil
		}

		fmt.Printf("📊 Found %d transactions (Total in API: %d)\n", len(data.Transactions), data.Total)

		filename := fmt.Sprintf("transactions_%s_to_%s_account_%s.json",
			from.Format("2006-01-02"), to.Format("2006-01-02"), filters.AccountID)
		filepath := filepath.Join(stagingDir, filename)

		if err := saveTransactionsV3(filepath, data.Transactions, data.Counts, from, to); err != nil {
			return fmt.Errorf("failed to save transactions: %w", err)
		}

		fmt.Printf("✅ Saved %d transactions to %s\n", len(data.Transactions), filename)
		fmt.Printf("📁 Staging directory: %s\n", stagingDir)
		return nil
	}

	// Basic fetching without account filtering
	data, err := client.FetchTransactions(userID, 50, "")
	if err != nil {
		return fmt.Errorf("failed to fetch transactions: %w", err)
	}

	if len(data.Transactions) == 0 {
		fmt.Println("📭 No transactions found")
		return nil
	}

	fmt.Printf("📊 Found %d transactions (Total in API: %d)\n", len(data.Transactions), data.Total)

	filename := fmt.Sprintf("transactions_%s_to_%s.json",
		from.Format("2006-01-02"), to.Format("2006-01-02"))
	filepath := filepath.Join(stagingDir, filename)

	if err := saveTransactionsV3(filepath, data.Transactions, data.Counts, from, to); err != nil {
		return fmt.Errorf("failed to save transactions: %w", err)
	}

	fmt.Printf("✅ Saved %d transactions to %s\n", len(data.Transactions), filename)
	fmt.Printf("📁 Staging directory: %s\n", stagingDir)
	return nil
}

// logAdvancedFilteringOptions logs which advanced filtering options are being used
func logAdvancedFilteringOptions(filters blend.TransactionFilters) {
	if filters.TimeFilter != "" {
		fmt.Printf("📅 Using time filter: %s\n", filters.TimeFilter)
	}
	if filters.AccountID != "" {
		fmt.Printf("🏦 Account filter: %s\n", filters.AccountID)
	}
	if filters.CategoryID != "" {
		fmt.Printf("🏷️  Category filter: %s\n", filters.CategoryID)
	}
	if filters.SubcategoryID != "" {
		fmt.Printf("🏷️  Subcategory filter: %s\n", filters.SubcategoryID)
	}
	if filters.SortBy != "txn_timestamp" || filters.SortOrder != "DESC" {
		fmt.Printf("📊 Sorting: %s %s\n", filters.SortBy, filters.SortOrder)
	}
	if filters.IncludeDetailed {
		fmt.Printf("📋 Including detailed search summary\n")
	}
	if filters.OrCategory {
		fmt.Printf("🔗 Using OR logic for category/subcategory\n")
	}
}

// displayTransactionCounts displays transaction count summaries
func displayTransactionCounts(counts []blend.TransactionCount) {
	for _, count := range counts {
		fmt.Printf("📈 %s: %.2f INR in (%d txns), %.2f INR out (%d txns)\n",
			count.Date, count.TotalIncoming, count.IncomingCount,
			count.TotalOutgoing, count.OutgoingCount)
	}
}

// generateAdvancedFilename creates a descriptive filename based on the filters used
func generateAdvancedFilename(filters blend.TransactionFilters) string {
	parts := []string{"blend_transactions"}

	if filters.TimeFilter != "" {
		parts = append(parts, filters.TimeFilter)
	} else {
		parts = append(parts, "advanced")
	}

	if filters.AccountID != "" {
		parts = append(parts, "account-"+filters.AccountID)
	}
	if filters.CategoryID != "" {
		parts = append(parts, "cat-"+filters.CategoryID)
	}
	if filters.SubcategoryID != "" {
		parts = append(parts, "subcat-"+filters.SubcategoryID)
	}
	if filters.SortBy != "txn_timestamp" {
		parts = append(parts, "sort-"+filters.SortBy)
	}
	if filters.SortOrder != "DESC" {
		parts = append(parts, filters.SortOrder)
	}

	parts = append(parts, time.Now().Format("20060102_150405"))
	return strings.Join(parts, "_") + ".json"
}

// TransactionFileV3 represents the structure for saving fetched v3 transaction data
type TransactionFileV3 struct {
	Transactions []blend.Transaction      `json:"transactions"`
	Counts       []blend.TransactionCount `json:"counts"`
	FetchedAt    time.Time                `json:"fetched_at"`
	DateRange    DateRange                `json:"date_range"`
	TotalCount   int                      `json:"total_count"`
}

// DateRange represents the date range for fetched transactions
type DateRange struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

func saveTransactionsV3(filepath string, transactions []blend.Transaction, counts []blend.TransactionCount, from, to time.Time) error {
	data := TransactionFileV3{
		Transactions: transactions,
		Counts:       counts,
		FetchedAt:    time.Now(),
		DateRange: DateRange{
			From: from,
			To:   to,
		},
		TotalCount: len(transactions),
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal transaction data: %w", err)
	}

	return os.WriteFile(filepath, jsonData, 0644)
}
