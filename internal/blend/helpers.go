package blend

import "unicode/utf8"

// isTextContent checks if byte slice contains text (not binary)
func isTextContent(data []byte) bool {
	if len(data) == 0 {
		return true
	}

	// Check if it's valid UTF-8 and doesn't contain too many control characters
	if !utf8.Valid(data) {
		return false
	}

	controlChars := 0
	for i, r := range string(data) {
		if i > 1000 { // Don't check entire large responses
			break
		}
		if r < 32 && r != '\n' && r != '\r' && r != '\t' {
			controlChars++
		}
	}

	// If more than 10% control characters, probably binary
	return float64(controlChars)/float64(len(data)) < 0.1
}
