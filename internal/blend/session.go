package blend

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// SessionManager handles session persistence and management
type SessionManager struct {
	sessionFile string
}

// NewSessionManager creates a new session manager
func NewSessionManager(sessionFile string) *SessionManager {
	return &SessionManager{
		sessionFile: sessionFile,
	}
}

// SaveSession saves the session to disk
func (sm *SessionManager) SaveSession(session *Session) error {
	// Ensure directory exists
	dir := filepath.Dir(sm.sessionFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create session directory: %w", err)
	}

	// Marshal session to JSON
	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	// Write to file with secure permissions
	if err := os.WriteFile(sm.sessionFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write session file: %w", err)
	}

	return nil
}

// LoadSession loads the session from disk
func (sm *SessionManager) LoadSession() (*Session, error) {
	// Check if file exists
	if _, err := os.Stat(sm.sessionFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("session file does not exist")
	}

	// Read file
	data, err := os.ReadFile(sm.sessionFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read session file: %w", err)
	}

	// Unmarshal session
	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	return &session, nil
}

// IsSessionValid checks if the session is still valid
func (sm *SessionManager) IsSessionValid(session *Session) bool {
	if session == nil {
		return false
	}

	if session.AccessToken == "" {
		return false
	}

	// Check if token is expired (with 5 minute buffer)
	if time.Now().Add(5 * time.Minute).After(session.ExpiresAt) {
		return false
	}

	return true
}

// DeleteSession removes the session file
func (sm *SessionManager) DeleteSession() error {
	if _, err := os.Stat(sm.sessionFile); os.IsNotExist(err) {
		return nil // File doesn't exist, nothing to do
	}

	if err := os.Remove(sm.sessionFile); err != nil {
		return fmt.Errorf("failed to delete session file: %w", err)
	}

	return nil
}

// GetSessionInfo returns information about the current session
func (sm *SessionManager) GetSessionInfo() (*SessionInfo, error) {
	session, err := sm.LoadSession()
	if err != nil {
		return &SessionInfo{
			Exists: false,
			Valid:  false,
		}, nil
	}

	info := &SessionInfo{
		Exists:          true,
		Valid:           sm.IsSessionValid(session),
		ExpiresAt:       session.ExpiresAt,
		HasRefreshToken: session.RefreshToken != "",
	}

	if info.Valid {
		info.TimeRemaining = time.Until(session.ExpiresAt)
	}

	return info, nil
}

// SessionInfo contains information about a session
type SessionInfo struct {
	Exists          bool          `json:"exists"`
	Valid           bool          `json:"valid"`
	ExpiresAt       time.Time     `json:"expires_at,omitempty"`
	TimeRemaining   time.Duration `json:"time_remaining,omitempty"`
	HasRefreshToken bool          `json:"has_refresh_token"`
}
