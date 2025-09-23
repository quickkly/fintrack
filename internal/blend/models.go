package blend

import (
	"time"
)

// =============================================================================
// CORE TRANSACTION MODELS
// =============================================================================

// Transaction represents a transaction from Bend /api/v3/users/{id}/transactions
type Transaction struct {
	// Core transaction data
	UUID         string    `json:"uuid"`
	Amount       float64   `json:"amount"`
	Currency     string    `json:"currency"`
	TxnTimestamp time.Time `json:"txn_timestamp"`
	Type         string    `json:"type"`      // INCOMING, OUTGOING
	Narration    string    `json:"narration"` // Transaction description
	Mode         string    `json:"mode"`      // UPI, FT, CARD, etc.
	Kind         string    `json:"kind"`      // e.g., "NORMAL"

	// Source currency information (for international transactions)
	SourceAmount   float64 `json:"source_amount"`
	SourceCurrency string  `json:"source_currency"`

	// Account and provider information
	AccountID                      string `json:"account_id"` // Account UUID
	FinancialInformationProviderID string `json:"financial_information_provider_id"`

	// Categorization
	Category *TransactionCategory `json:"category"` // Category with ID and subcategory
	Merchant *TransactionMerchant `json:"merchant"` // Detailed merchant info

	// Metadata
	TransactionID string  `json:"transaction_id"`
	Reference     string  `json:"reference"`
	Summary       string  `json:"summary"` // Human-readable summary
	Notes         *string `json:"notes"`

	// Timestamps
	ExtractedTime *time.Time `json:"extracted_time"`

	// Flags and status
	ExcludedFromCashFlow   bool `json:"excluded_from_cash_flow"`
	IsBookmarked           bool `json:"is_bookmarked"`
	IsHidden               bool `json:"is_hidden"`
	IsPossibleDuplicate    bool `json:"is_possible_duplicate"`
	IsCCManualOrBankLinked bool `json:"is_cc_manual_or_bank_linked"`

	// Additional fields
	Via                      *string           `json:"via"`
	AccountIn                *string           `json:"account_in"`
	Refund                   TransactionRefund `json:"refund"`
	Receipts                 []interface{}     `json:"receipts"`
	GroupIDs                 *string           `json:"group_ids"`
	Source                   string            `json:"source"` // e.g., "BANK"
	LinkedCCAccountIDForBill *string           `json:"linked_cc_account_id_for_bill"`
	LinkedCCTransactionID    *string           `json:"linked_cc_transaction_id"`
	UserManualAdded          *bool             `json:"user_manual_added"`
	SplitType                *string           `json:"split_type"`
	RemainingAmount          *float64          `json:"remaining_amount"`
	ParentTransactionID      *string           `json:"parent_transaction_id"`
}

// TransactionCategory represents transaction category information
type TransactionCategory struct {
	ID            *string `json:"id"`
	SubcategoryID *string `json:"subcategory_id"`
}

// TransactionMerchant represents merchant information in transactions
type TransactionMerchant struct {
	ID      *string `json:"id"`
	Name    *string `json:"name"`
	Type    string  `json:"type"`
	Logo    *string `json:"logo"`
	Address *string `json:"address"`
}

// TransactionRefund represents refund status and information
type TransactionRefund struct {
	Status     string     `json:"status"` // e.g., "NONE"
	Notify     bool       `json:"notify"`
	ReceivedOn *time.Time `json:"received_on"`
}

// TransactionCount represents monthly transaction counts and totals
type TransactionCount struct {
	Date          string  `json:"date"` // e.g., "2025-08"
	TotalIncoming float64 `json:"total_incoming"`
	TotalOutgoing float64 `json:"total_outgoing"`
	IncomingCount int     `json:"incoming_count"`
	OutgoingCount int     `json:"outgoing_count"`
	Total         int     `json:"total"`
	BeforeAccount int     `json:"before_account"`
	AfterAccount  int     `json:"after_account"`
}

// =============================================================================
// ACCOUNT MODELS
// =============================================================================

// Account represents a bank account from Bend /api/v1/aa/data
type Account struct {
	// Core account information
	UUID                string `json:"uuid"`
	HolderName          string `json:"holder_name"`
	MaskedAccountNumber string `json:"masked_account_number"`
	Type                string `json:"type"` // e.g., "deposit"

	// Account details
	AccountNumber         *string `json:"account_number"`
	AccountNumberVerified bool    `json:"account_number_verified"`
	IFSCCode              string  `json:"ifsc_code"`
	SwiftCode             string  `json:"swift_code"`
	Nickname              *string `json:"nickname"`
	Track                 string  `json:"track"` // e.g., "ACTIVELY"
	FirstPullCompleted    bool    `json:"first_pull_completed"`

	// Balance and currency
	CurrentBalance float64 `json:"current_balance"`
	Currency       string  `json:"currency"`

	// Timestamps
	LastFetchedAt time.Time `json:"last_fetched_at"`

	// Provider information
	FinancialInformationProvider FinancialInformationProvider `json:"financial_information_provider"`
}

// FinancialInformationProvider represents bank details from /api/v1/aa/data
type FinancialInformationProvider struct {
	UUID         string `json:"uuid"`
	Name         string `json:"name"`
	FIPID        string `json:"fip_id"`
	IsValidTime  bool   `json:"is_valid_time"`
	InvalidTxnID bool   `json:"invalid_txn_id"`
	LogoURL      string `json:"logo_url"`
}

// =============================================================================
// USER MODELS
// =============================================================================

// UserInfo represents user information from Bend
type UserInfo struct {
	// Core user information
	UUID      string `json:"uuid"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	Phone     string `json:"phone"`
	Username  string `json:"username"`

	// Optional fields
	MiddleName *string `json:"middle_name"`
	ProfilePic *string `json:"profile_pic"`

	// Verification status
	EmailVerified bool `json:"email_verified"`
	PhoneVerified bool `json:"phone_verified"`

	// Account linking
	GoogleLinked bool `json:"google_linked"`
	AppleLinked  bool `json:"apple_linked"`

	// User role and access
	Role           string `json:"role"`
	IsInternalUser bool   `json:"is_internal_user"`
	BetaAccess     bool   `json:"beta_access"`
	WebBetaAccess  bool   `json:"web_beta_access"`
	CCEnabled      bool   `json:"cc_enabled"`

	// Metadata
	Timezone  string `json:"timezone"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// GetFullName returns the user's full name
func (u *UserInfo) GetFullName() string {
	name := u.FirstName
	if u.MiddleName != nil && *u.MiddleName != "" {
		name += " " + *u.MiddleName
	}
	if u.LastName != "" {
		name += " " + u.LastName
	}
	return name
}

// =============================================================================
// SESSION MODELS
// =============================================================================

// Session represents authentication session data
type Session struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	TokenType    string    `json:"token_type"`
	MarbleCookie string    `json:"marble_cookie"`
	DeviceHash   string    `json:"device_hash"`
}

// =============================================================================
// API RESPONSE MODELS
// =============================================================================

// APIResponse represents the standard API response structure
type APIResponse struct {
	Meta  APIResponseMeta `json:"meta"`
	Data  interface{}     `json:"data"`
	Error interface{}     `json:"error"`
}

// APIResponseMeta represents metadata in API responses
type APIResponseMeta struct {
	RequestID string `json:"request_id"`
	Timestamp string `json:"timestamp"`
	URI       string `json:"uri"`
}

// TokenData represents token information in API responses
type TokenData struct {
	TokenType    string `json:"token_type"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    string `json:"expires_at"`
}

// RefreshRequest represents token refresh request
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// RefreshResponse represents refresh token response
type RefreshResponse struct {
	Meta  APIResponseMeta `json:"meta"`
	Data  TokenData       `json:"data"`
	Error interface{}     `json:"error"`
}

// =============================================================================
// TRANSACTION API RESPONSE MODELS
// =============================================================================

// TransactionsV3Response represents the complete /api/v3/users/{id}/transactions response
type TransactionsV3Response struct {
	Meta  APIResponseMeta    `json:"meta"`
	Data  TransactionsV3Data `json:"data"`
	Error interface{}        `json:"error"`
}

// TransactionsV3Data represents the data section of /api/v3/users/{id}/transactions response
type TransactionsV3Data struct {
	Transactions       []Transaction      `json:"transactions"`
	Counts             []TransactionCount `json:"counts"`
	Total              int                `json:"total"`
	SearchSummary      *string            `json:"search_summary"`
	After              string             `json:"after"` // Pagination cursor
	ParentTransactions interface{}        `json:"parent_transactions"`
}

// =============================================================================
// ACCOUNT API RESPONSE MODELS
// =============================================================================

// AADataResponse represents the complete /api/v1/aa/data response
type AADataResponse struct {
	Meta  APIResponseMeta `json:"meta"`
	Data  AAData          `json:"data"`
	Error interface{}     `json:"error"`
}

// AAData represents the data section of /api/v1/aa/data response
type AAData struct {
	Accounts []Account `json:"accounts"`
}

// =============================================================================
// USER API RESPONSE MODELS
// =============================================================================

// UserMeResponse represents the complete /api/v2/users/me response
type UserMeResponse struct {
	Meta  APIResponseMeta  `json:"meta"`
	Data  UserDataResponse `json:"data"`
	Error interface{}      `json:"error"`
}

// UserDataResponse represents the complete user data response structure
type UserDataResponse struct {
	User       UserInfo   `json:"user"`
	Settings   Settings   `json:"settings"`
	Onboarding Onboarding `json:"onboarding"`
	Route      string     `json:"route"`
}

// =============================================================================
// LEGACY MODELS (for backward compatibility)
// =============================================================================

// BankAccount represents a bank account from settings (legacy)
type BankAccount struct {
	AccountID           string `json:"account_id"`
	AccountName         string `json:"account_name"`
	MaskedAccountNumber string `json:"masked_account_number"`
	ShowInWidget        bool   `json:"show_in_widget"`
	Order               int    `json:"order"`
}

// ManualAccount represents a manual account (like cash) (legacy)
type ManualAccount struct {
	AccountID    string `json:"account_id"`
	ShowInWidget bool   `json:"show_in_widget"`
	Type         string `json:"type"`
}

// ConvertBankAccountToAccount converts BankAccount from settings to unified Account (legacy)
func (ba *BankAccount) ToAccount() Account {
	return Account{
		UUID:                ba.AccountID,
		HolderName:          "",
		MaskedAccountNumber: ba.MaskedAccountNumber,
		Type:                "Bank",
		Currency:            "INR", // Default for Indian accounts
		CurrentBalance:      0,     // No balance info in settings
		FinancialInformationProvider: FinancialInformationProvider{
			Name: ba.AccountName,
		},
	}
}

// ConvertManualAccountToAccount converts ManualAccount to unified Account (legacy)
func (ma *ManualAccount) ToAccount() Account {
	return Account{
		UUID:           ma.AccountID,
		HolderName:     "",
		Type:           ma.Type,
		Currency:       "",
		CurrentBalance: 0,
		FinancialInformationProvider: FinancialInformationProvider{
			Name: ma.Type,
		},
	}
}

// =============================================================================
// SETTINGS MODELS (for user preferences)
// =============================================================================

// Settings represents user settings from Bend
type Settings struct {
	// Add settings fields as needed
}

// Onboarding represents onboarding status
type Onboarding struct {
	// Add onboarding fields as needed
}

// =============================================================================
// LEGACY RESPONSE MODELS (for backward compatibility)
// =============================================================================

// TransactionResponse represents API response for transactions (legacy)
type TransactionResponse struct {
	Transactions []Transaction `json:"transactions"`
	Count        int           `json:"count"`
	Page         int           `json:"page"`
	TotalPages   int           `json:"total_pages"`
}

// AccountResponse represents API response for accounts (legacy)
type AccountResponse struct {
	Accounts []Account `json:"accounts"`
	Count    int       `json:"count"`
}

// Cookie represents HTTP cookies for session management (legacy)
type Cookie struct {
	Name     string    `json:"name"`
	Value    string    `json:"value"`
	Domain   string    `json:"domain"`
	Path     string    `json:"path"`
	Expires  time.Time `json:"expires"`
	Secure   bool      `json:"secure"`
	HttpOnly bool      `json:"http_only"`
}
