package blend

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/quickkly/fintrack/internal/config"

	"github.com/andybalholm/brotli"
)

// Client represents the Bend client
type Client struct {
	httpClient     *http.Client
	baseURL        string
	session        *Session
	rateLimiter    *time.Ticker
	deviceHash     string
	deviceType     string
	deviceLocation string
	enableLogging  bool
}

// NewClient creates a new Bend financial client
func NewClient(cfg *config.Config) *Client {
	deviceHash := cfg.Bend.DeviceHash
	if deviceHash == "" {
		// Generate a unique device hash if not provided in config
		deviceHash = GenerateDeviceHash()
	}

	return &Client{
		httpClient: &http.Client{
			Timeout: cfg.Bend.Timeout,
		},
		baseURL:        cfg.Bend.BaseURL,
		rateLimiter:    time.NewTicker(cfg.Bend.RateLimit),
		deviceHash:     deviceHash,
		deviceType:     cfg.Bend.DeviceType,
		deviceLocation: cfg.Bend.DeviceLocation,
		enableLogging:  false, // Default to false, can be enabled via SetLogging
	}
}

// SetSession sets the authentication session
func (c *Client) SetSession(session *Session) {
	c.session = session
}

// GetSession returns the current session
func (c *Client) GetSession() *Session {
	return c.session
}

// CheckSession validates the current session
func (c *Client) CheckSession() (*UserInfo, error) {
	if c.session == nil {
		return nil, fmt.Errorf("no session available")
	}

	if time.Now().After(c.session.ExpiresAt) {
		return nil, fmt.Errorf("session expired")
	}

	// Wait for rate limiter
	<-c.rateLimiter.C

	req, err := c.newRequest("GET", "/api/v2/users/me", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	var response UserMeResponse
	if err := c.doRequest(req, &response); err != nil {
		return nil, fmt.Errorf("failed to check session: %w", err)
	}

	if response.Error != nil {
		return nil, fmt.Errorf("session expired or invalid: %v", response.Error)
	}

	// Extract user info from the nested data structure
	userInfo := response.Data.User
	return &userInfo, nil
}

// GetUserID returns the current user's UUID from the session/user info
func (c *Client) GetUserID() (string, error) {
	userInfo, err := c.CheckSession()
	if err != nil {
		return "", fmt.Errorf("failed to get user info: %w", err)
	}
	return userInfo.UUID, nil
}

// RefreshSession refreshes the authentication token
func (c *Client) RefreshSession() error {
	if c.session == nil || c.session.RefreshToken == "" {
		return fmt.Errorf("no refresh token available")
	}

	// Wait for rate limiter
	<-c.rateLimiter.C

	refreshReq := RefreshRequest{
		RefreshToken: c.session.RefreshToken,
	}

	req, err := c.newRequest("POST", "/api/v1/auth/tokens/refresh", refreshReq)
	if err != nil {
		return fmt.Errorf("failed to create refresh request: %w", err)
	}

	var response RefreshResponse
	if err := c.doRequest(req, &response); err != nil {
		return fmt.Errorf("failed to refresh session: %w", err)
	}

	if response.Error != nil {
		return fmt.Errorf("refresh failed: %v", response.Error)
	}

	// Parse expires_at timestamp
	expiresAt, err := time.Parse(time.RFC3339, response.Data.ExpiresAt)
	if err != nil {
		return fmt.Errorf("failed to parse expires_at: %w", err)
	}

	// Update session with new tokens
	c.session.AccessToken = response.Data.AccessToken
	c.session.RefreshToken = response.Data.RefreshToken
	c.session.TokenType = response.Data.TokenType
	c.session.ExpiresAt = expiresAt

	return nil
}

// TransactionFilters represents advanced filtering options for transactions
type TransactionFilters struct {
	Limit      int    `json:"limit,omitempty"`
	After      string `json:"after,omitempty"`
	CountBy    string `json:"count_by,omitempty"`    // e.g., "month"
	TimeFilter string `json:"time_filter,omitempty"` // e.g., "this_month", "last_month"
	Include    string `json:"include,omitempty"`     // e.g., "count_by_totals"

	// Advanced filtering parameters from curl command
	SortBy          string    `json:"sort_by,omitempty"`          // e.g., "txn_timestamp"
	SortOrder       string    `json:"sort_order,omitempty"`       // e.g., "DESC"
	StartDate       time.Time `json:"start_date,omitempty"`       // Start date for filtering
	EndDate         time.Time `json:"end_date,omitempty"`         // End date for filtering
	AccountID       string    `json:"account_id,omitempty"`       // Filter by account ID
	CategoryID      string    `json:"category_id,omitempty"`      // Filter by category ID
	SubcategoryID   string    `json:"subcategory_id,omitempty"`   // Filter by subcategory ID
	IncludeCountBy  bool      `json:"include_count_by,omitempty"` // Include count_by_totals
	IncludeDetailed bool      `json:"include_detailed,omitempty"` // Include detailed_search_summary
	OrCategory      bool      `json:"or_category,omitempty"`      // Use OR logic for category/subcategory
}

// FetchTransactions fetches transactions for a specific user with advanced filtering
func (c *Client) FetchTransactions(userID string, limit int, after string) (*TransactionsV3Data, error) {
	filters := TransactionFilters{
		Limit: limit,
		After: after,
	}
	return c.FetchTransactionsWithFilters(userID, filters)
}

// FetchTransactionsWithFilters fetches transactions with advanced filtering options
func (c *Client) FetchTransactionsWithFilters(userID string, filters TransactionFilters) (*TransactionsV3Data, error) {
	if c.session == nil {
		return nil, fmt.Errorf("no session available")
	}

	// Wait for rate limiter
	<-c.rateLimiter.C

	// Build query parameters
	params := c.buildTransactionQueryParams(filters)

	// Build endpoint URL
	endpoint := fmt.Sprintf("/api/v3/users/%s/transactions", userID)
	if len(params) > 0 {
		endpoint += "?" + params.Encode()
	}

	// Make request
	req, err := c.newRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	var response TransactionsV3Response
	if err := c.doRequest(req, &response); err != nil {
		return nil, fmt.Errorf("failed to fetch transactions: %w", err)
	}

	if response.Error != nil {
		return nil, fmt.Errorf("failed to fetch transactions: %v", response.Error)
	}

	return &response.Data, nil
}

// buildTransactionQueryParams builds URL query parameters from transaction filters
func (c *Client) buildTransactionQueryParams(filters TransactionFilters) url.Values {
	params := url.Values{}

	// Basic parameters
	if filters.Limit > 0 {
		params.Set("limit", fmt.Sprintf("%d", filters.Limit))
	}
	if filters.After != "" {
		params.Set("after", filters.After)
	}
	if filters.CountBy != "" {
		params.Set("count_by", filters.CountBy)
	}
	if filters.TimeFilter != "" {
		params.Set("time_filter", filters.TimeFilter)
	}
	if filters.Include != "" {
		params.Add("include[]", filters.Include)
	}

	// Sorting parameters
	if filters.SortBy != "" {
		params.Set("sort_by", filters.SortBy)
	}
	if filters.SortOrder != "" {
		params.Set("sort_order", filters.SortOrder)
	}

	// Date range parameters
	if !filters.StartDate.IsZero() {
		params.Set("start_date", filters.StartDate.Format(time.RFC3339))
	}
	if !filters.EndDate.IsZero() {
		params.Set("end_date", filters.EndDate.Format(time.RFC3339))
	}

	// Filtering parameters
	if filters.CategoryID != "" {
		params.Set("category_id", filters.CategoryID)
	}
	if filters.AccountID != "" {
		params.Set("account_id[]", filters.AccountID)
	}
	if filters.SubcategoryID != "" {
		params.Set("subcategory_id", filters.SubcategoryID)
	}

	// Include parameters
	if filters.IncludeCountBy {
		params.Add("include[]", "count_by_totals")
	}
	if filters.IncludeDetailed {
		params.Add("include[]", "detailed_search_summary")
	}

	// OR logic parameters
	if filters.OrCategory {
		params.Add("or[]", "subcategory_id")
		params.Add("or[]", "category_id")
	}

	return params
}

// FetchAllTransactions fetches all transactions with pagination support
func (c *Client) FetchAllTransactions(userID string, limit int) ([]Transaction, []TransactionCount, error) {
	var allTransactions []Transaction
	var allCounts []TransactionCount
	after := ""

	for {
		data, err := c.FetchTransactions(userID, limit, after)
		if err != nil {
			return nil, nil, err
		}

		allTransactions = append(allTransactions, data.Transactions...)
		if len(data.Counts) > 0 {
			allCounts = append(allCounts, data.Counts...)
		}

		// Check if there are more pages
		if data.After == "" || len(data.Transactions) < limit {
			break
		}
		after = data.After
	}

	return allTransactions, allCounts, nil
}

// FetchTransactionsWithCurlParams creates filters matching the curl command parameters
func (c *Client) FetchTransactionsWithCurlParams(userID string, startDate, endDate time.Time, categoryID, subcategoryID string) (*TransactionsV3Data, error) {
	filters := TransactionFilters{
		SortBy:          "txn_timestamp",
		SortOrder:       "DESC",
		StartDate:       startDate,
		EndDate:         endDate,
		CountBy:         "month",
		IncludeCountBy:  true,
		IncludeDetailed: true,
		OrCategory:      true,
	}

	if categoryID != "" {
		filters.CategoryID = categoryID
	}
	if subcategoryID != "" {
		filters.SubcategoryID = subcategoryID
	}

	return c.FetchTransactionsWithFilters(userID, filters)
}

// GetAccounts fetches all available accounts with real balances and transactions
func (c *Client) GetAccounts() ([]Account, error) {
	if c.session == nil {
		return nil, fmt.Errorf("no session available")
	}

	// Wait for rate limiter
	<-c.rateLimiter.C

	// Get comprehensive account data from the AA endpoint
	req, err := c.newRequest("GET", "/api/v1/aa/data", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	var response AADataResponse
	if err := c.doRequest(req, &response); err != nil {
		return nil, fmt.Errorf("failed to fetch account data: %w", err)
	}

	if response.Error != nil {
		return nil, fmt.Errorf("failed to get accounts: %v", response.Error)
	}

	return response.Data.Accounts, nil
}

// InitializeFromRefreshToken initializes session from a refresh token
func (c *Client) InitializeFromRefreshToken(refreshToken string) error {
	// Create initial session with refresh token
	session := InitializeSession(refreshToken, c.deviceHash)
	c.SetSession(session)

	// Refresh to get access token
	return c.RefreshSession()
}

// SetLogging enables or disables HTTP request/response logging
func (c *Client) SetLogging(enabled bool) {
	c.enableLogging = enabled
}

// logRequest logs the complete HTTP request details
func (c *Client) logRequest(req *http.Request, body []byte) {
	if !c.enableLogging {
		return
	}

	fmt.Printf("\n=== HTTP REQUEST ===\n")
	fmt.Printf("Method: %s\n", req.Method)
	fmt.Printf("URL: %s\n", req.URL.String())
	fmt.Printf("Headers:\n")
	for name, values := range req.Header {
		for _, value := range values {
			// Mask sensitive headers
			if name == "Authorization" {
				if len(value) > 10 {
					fmt.Printf("  %s: %s...\n", name, value[:10])
				} else {
					fmt.Printf("  %s: [REDACTED]\n", name)
				}
			} else {
				fmt.Printf("  %s: %s\n", name, value)
			}
		}
	}

	if len(body) > 0 {
		fmt.Printf("Body: %s\n", string(body))
	}
	fmt.Printf("==================\n")
}

// logResponse logs the complete HTTP response details
func (c *Client) logResponse(resp *http.Response, body []byte) {
	if !c.enableLogging {
		return
	}

	fmt.Printf("\n=== HTTP RESPONSE ===\n")
	fmt.Printf("Status: %s\n", resp.Status)
	fmt.Printf("Headers:\n")
	for name, values := range resp.Header {
		for _, value := range values {
			fmt.Printf("  %s: %s\n", name, value)
		}
	}

	if len(body) > 0 {
		// Truncate very long responses for readability
		if len(body) > 1000 {
			fmt.Printf("Body (truncated): %s...\n", string(body[:1000]))
		} else {
			fmt.Printf("Body: %s\n", string(body))
		}
	}
	fmt.Printf("===================\n")
}

// newRequest creates a new HTTP request with proper headers
func (c *Client) newRequest(method, endpoint string, body interface{}) (*http.Request, error) {
	url := c.baseURL + endpoint

	var reqBody io.Reader
	var bodyBytes []byte
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyBytes = jsonBody
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	c.setStandardHeaders(req)
	c.setDeviceHeaders(req)
	c.setAuthenticationHeaders(req)

	// Log the request if logging is enabled
	c.logRequest(req, bodyBytes)

	return req, nil
}

// setStandardHeaders sets standard HTTP headers
func (c *Client) setStandardHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Origin", "https://bend.example.com")
}

// setDeviceHeaders sets device-specific headers required by Bend
func (c *Client) setDeviceHeaders(req *http.Request) {
	req.Header.Set("X-Device-Hash", c.deviceHash)
	req.Header.Set("X-Device-Type", c.deviceType)
	req.Header.Set("X-Device-Location", c.deviceLocation)

	// Generate a unique request ID for tracking
	requestID := generateRequestID()
	req.Header.Set("X-Request-ID", requestID)
}

// setAuthenticationHeaders sets authentication headers if session exists
func (c *Client) setAuthenticationHeaders(req *http.Request) {
	if c.session == nil {
		return
	}

	// Set authorization header
	if c.session.AccessToken != "" {
		authHeader := c.session.TokenType + " " + c.session.AccessToken
		if c.session.TokenType == "" {
			authHeader = "Bearer " + c.session.AccessToken
		}
		req.Header.Set("Authorization", authHeader)
	}

	// Add marble-cookie if available
	if c.session.MarbleCookie != "" {
		req.AddCookie(&http.Cookie{
			Name:  "marble-cookie",
			Value: c.session.MarbleCookie,
		})
	}
}

// doRequest executes an HTTP request and decodes the response
func (c *Client) doRequest(req *http.Request, v interface{}) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read and process response body
	body, err := c.readResponseBody(resp)
	if err != nil {
		return err
	}

	// Log the response if logging is enabled
	c.logResponse(resp, body)

	// Handle error responses
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return c.handleErrorResponse(resp, body)
	}

	// Save cookies from response
	c.saveResponseCookies(resp)

	// Decode response if target provided
	if v != nil {
		if err := json.Unmarshal(body, v); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

// readResponseBody reads and decompresses the response body
func (c *Client) readResponseBody(resp *http.Response) ([]byte, error) {
	var reader io.Reader = resp.Body
	contentEncoding := resp.Header.Get("Content-Encoding")

	if strings.Contains(contentEncoding, "gzip") || strings.Contains(contentEncoding, "br") {
		var err error
		reader, err = c.createDecompressionReader(resp.Body, contentEncoding)
		if err != nil {
			return nil, err
		}
	}

	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return body, nil
}

// createDecompressionReader creates a reader for compressed content
func (c *Client) createDecompressionReader(body io.ReadCloser, contentEncoding string) (io.Reader, error) {
	if strings.Contains(contentEncoding, "gzip") {
		gzReader, err := gzip.NewReader(body)
		if err != nil {
			return nil, fmt.Errorf("failed to create gzip reader: %w", err)
		}
		return gzReader, nil
	} else if strings.Contains(contentEncoding, "br") {
		return brotli.NewReader(body), nil
	}
	return body, nil
}

// handleErrorResponse processes error responses and returns appropriate error messages
func (c *Client) handleErrorResponse(resp *http.Response, body []byte) error {
	// Clean error message for common cases
	errorMsg := string(body)
	if !isTextContent(body) {
		errorMsg = "[binary/compressed response]"
	} else if len(errorMsg) > 200 {
		errorMsg = errorMsg[:200] + "..."
	}

	// Try to parse error as JSON for better error messages
	var errorResp APIResponse
	if json.Unmarshal(body, &errorResp) == nil && errorResp.Error != nil {
		return fmt.Errorf("API request failed with status %d: %v", resp.StatusCode, errorResp.Error)
	}

	return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, errorMsg)
}

// saveResponseCookies saves cookies from the response to the session
func (c *Client) saveResponseCookies(resp *http.Response) {
	if c.session == nil || len(resp.Cookies()) == 0 {
		return
	}

	for _, cookie := range resp.Cookies() {
		if cookie.Name == "marble-cookie" {
			c.session.MarbleCookie = cookie.Value
			break
		}
	}
}

// Close cleans up the client resources
func (c *Client) Close() {
	if c.rateLimiter != nil {
		c.rateLimiter.Stop()
	}
}
