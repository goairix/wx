package webhook

import (
	"crypto/sha1"
	"crypto/subtle"
	"encoding/hex"
	"sort"
)

// Signature returns the WeChat URL verification signature for token, timestamp and nonce.
func Signature(token, timestamp, nonce string) string {
	return signature(token, timestamp, nonce)
}

// VerifySignature validates a plain webhook URL signature.
func VerifySignature(token, timestamp, nonce, expected string) error {
	if !equalSignature(signature(token, timestamp, nonce), expected) {
		return ErrInvalidSignature
	}
	return nil
}

// MessageSignature returns the WeChat encrypted-message signature.
func MessageSignature(token, timestamp, nonce, encrypted string) string {
	return signature(token, timestamp, nonce, encrypted)
}

// VerifyMessageSignature validates an encrypted-message signature.
func VerifyMessageSignature(token, timestamp, nonce, encrypted, expected string) error {
	if !equalSignature(signature(token, timestamp, nonce, encrypted), expected) {
		return ErrInvalidSignature
	}
	return nil
}

func signature(parts ...string) string {
	values := append([]string(nil), parts...)
	sort.Strings(values)
	h := sha1.Sum([]byte(join(values)))
	return hex.EncodeToString(h[:])
}

func join(values []string) string {
	var out string
	for _, value := range values {
		out += value
	}
	return out
}

func equalSignature(got, want string) bool {
	if len(got) != len(want) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(want)) == 1
}
