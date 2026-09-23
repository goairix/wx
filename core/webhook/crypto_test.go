package webhook

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"strings"
	"testing"
)

func TestSignatureVerification(t *testing.T) {
	const token, timestamp, nonce = "secret", "12345", "nonce"
	const valid = ""
	_ = valid
	signature := Signature(token, timestamp, nonce)
	if err := VerifySignature(token, timestamp, nonce, signature); err != nil {
		t.Fatal(err)
	}
	if err := VerifySignature(token, timestamp, nonce, "wrong"); err == nil {
		t.Fatal("accepted invalid signature")
	}
	messageSignature := MessageSignature(token, timestamp, nonce, "ciphertext")
	if err := VerifyMessageSignature(token, timestamp, nonce, "ciphertext", messageSignature); err != nil {
		t.Fatal(err)
	}
	if err := VerifyMessageSignature(token, timestamp, nonce, "tampered", messageSignature); err == nil {
		t.Fatal("accepted tampered message")
	}
}

func TestDecryptMessage(t *testing.T) {
	key := bytes.Repeat([]byte("k"), 32)
	encodedKey := strings.TrimRight(base64.StdEncoding.EncodeToString(key), "=")
	message := []byte(`<xml><Content>Hello</Content></xml>`)
	ciphertext := fixtureCiphertext(t, key, message, "receiver", 32)
	decrypted, err := DecryptMessage(encodedKey, ciphertext, "receiver")
	if err != nil || !bytes.Equal(decrypted, message) {
		t.Fatalf("message=%q err=%v", decrypted, err)
	}
	if _, err := DecryptMessage(encodedKey, ciphertext, "other"); err == nil {
		t.Fatal("accepted wrong receiver")
	}
	if _, err := DecryptMessage(encodedKey, "%%%", "receiver"); err == nil {
		t.Fatal("accepted invalid base64")
	}
	if _, err := DecryptMessage("bad", ciphertext, "receiver"); err == nil {
		t.Fatal("accepted invalid key")
	}
}

func TestDecryptMessageAcceptsProtocolPaddingLengths(t *testing.T) {
	key := bytes.Repeat([]byte("k"), 32)
	encodedKey := strings.TrimRight(base64.StdEncoding.EncodeToString(key), "=")

	for padding := 17; padding <= 32; padding++ {
		padding := padding
		t.Run(fmt.Sprintf("padding-%d", padding), func(t *testing.T) {
			// The encrypted payload has 28 bytes of framing and receiver data.
			// Choose the message length so that the
			// protocol's 32-byte PKCS#7 padding is exactly padding bytes.
			message := bytes.Repeat([]byte("m"), 36-padding)
			ciphertext := fixtureCiphertext(t, key, message, "receiver", 32)
			decrypted, err := DecryptMessage(encodedKey, ciphertext, "receiver")
			if err != nil {
				t.Fatalf("padding=%d: decrypt failed: %v", padding, err)
			}
			if !bytes.Equal(decrypted, message) {
				t.Fatalf("padding=%d: message=%q, want %q", padding, decrypted, message)
			}
		})
	}
}

func TestDecryptMessageRejectsMalformedPaddingAndPayload(t *testing.T) {
	key := bytes.Repeat([]byte("k"), 32)
	encodedKey := strings.TrimRight(base64.StdEncoding.EncodeToString(key), "=")
	for name, plaintext := range map[string][]byte{
		"short":            []byte("short"),
		"oversized length": append(append(bytes.Repeat([]byte("r"), 16), 0, 0, 1, 0), []byte("datareceiver")...),
	} {
		t.Run(name, func(t *testing.T) {
			ciphertext := fixtureCiphertext(t, key, plaintext, "", 32)
			if _, err := DecryptMessage(encodedKey, ciphertext, "receiver"); err == nil {
				t.Fatal("accepted malformed payload")
			}
		})
	}
	badPadding := fixtureCiphertext(t, key, []byte("payload"), "", 32)
	decoded, _ := base64.StdEncoding.DecodeString(badPadding)
	block, _ := aes.NewCipher(key)
	plain := make([]byte, len(decoded))
	cipher.NewCBCDecrypter(block, key[:aes.BlockSize]).CryptBlocks(plain, decoded)
	plain[len(plain)-2] ^= 1
	cipher.NewCBCEncrypter(block, key[:aes.BlockSize]).CryptBlocks(decoded, plain)
	if _, err := DecryptMessage(encodedKey, base64.StdEncoding.EncodeToString(decoded), "receiver"); err == nil {
		t.Fatal("accepted invalid padding")
	}
}

func fixtureCiphertext(t *testing.T, key, message []byte, receiver string, padBlock int) string {
	t.Helper()
	plain := append([]byte{}, bytes.Repeat([]byte("r"), 16)...)
	length := make([]byte, 4)
	binary.BigEndian.PutUint32(length, uint32(len(message)))
	plain = append(plain, length...)
	plain = append(plain, message...)
	plain = append(plain, receiver...)
	padding := padBlock - len(plain)%padBlock
	plain = append(plain, bytes.Repeat([]byte{byte(padding)}, padding)...)
	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatal(err)
	}
	ciphertext := make([]byte, len(plain))
	cipher.NewCBCEncrypter(block, key[:aes.BlockSize]).CryptBlocks(ciphertext, plain)
	return base64.StdEncoding.EncodeToString(ciphertext)
}
