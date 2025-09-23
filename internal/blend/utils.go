package blend

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
)

// generateRequestID generates a UUID-like request ID for API calls
func generateRequestID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

// GenerateDeviceHash generates a unique device hash (UUID v4 format)
func GenerateDeviceHash() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		// Fallback to a deterministic but unique hash based on hostname and user
		hostname, _ := os.Hostname()
		user := os.Getenv("USER")
		if user == "" {
			user = os.Getenv("USERNAME") // Windows fallback
		}
		fallback := fmt.Sprintf("fintrack-%s-%s", hostname, user)
		// Convert to UUID format
		return fmt.Sprintf("%x-%x-%x-%x-%x",
			[]byte(fallback)[:4],
			[]byte(fallback)[4:6],
			[]byte(fallback)[6:8],
			[]byte(fallback)[8:10],
			[]byte(fallback)[10:16])
	}

	// Set version (4) and variant bits
	b[6] = (b[6] & 0x0f) | 0x40 // Version 4
	b[8] = (b[8] & 0x3f) | 0x80 // Variant 10

	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

// GetOrCreateDeviceHash returns existing device hash from config dir or creates a new one
func GetOrCreateDeviceHash(configDir string) (string, error) {
	deviceHashFile := filepath.Join(configDir, "device_hash")

	// Try to read existing device hash
	if data, err := os.ReadFile(deviceHashFile); err == nil {
		deviceHash := string(data)
		if len(deviceHash) > 0 {
			return deviceHash, nil
		}
	}

	// Generate new device hash
	deviceHash := GenerateDeviceHash()

	// Ensure config directory exists
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return deviceHash, fmt.Errorf("failed to create config directory: %w", err)
	}

	// Save device hash to file
	if err := os.WriteFile(deviceHashFile, []byte(deviceHash), 0644); err != nil {
		return deviceHash, fmt.Errorf("failed to save device hash: %w", err)
	}

	return deviceHash, nil
}

// InitializeSession creates a session from a refresh token
func InitializeSession(refreshToken, deviceHash string) *Session {
	return &Session{
		RefreshToken: refreshToken,
		DeviceHash:   deviceHash,
		TokenType:    "Bearer",
	}
}
