// Package random provides cryptographically secure random identifiers.
package random

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

// String returns a URL-safe random string with at least length characters.
func String(length int) (string, error) {
	if length <= 0 {
		return "", nil
	}
	data := make([]byte, (length*3+3)/4)
	if _, err := rand.Read(data); err != nil {
		return "", fmt.Errorf("random string: %w", err)
	}
	encoded := base64.RawURLEncoding.EncodeToString(data)
	if len(encoded) > length {
		encoded = encoded[:length]
	}
	return encoded, nil
}
