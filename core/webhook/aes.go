package webhook

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
)

var (
	ErrInvalidSignature = errors.New("webhook: invalid signature")
	ErrInvalidAESKey    = errors.New("webhook: invalid AES key")
	ErrInvalidBase64    = errors.New("webhook: invalid base64")
	ErrInvalidPadding   = errors.New("webhook: invalid PKCS#7 padding")
	ErrMalformedPayload = errors.New("webhook: malformed encrypted payload")
	ErrInvalidReceiver  = errors.New("webhook: invalid receiver id")
)

// DecryptMessage decrypts a WeChat encrypted message. encodingAESKey is the
// unpadded base64 key supplied by the platform and receiverID is the expected
// account or enterprise identifier.
func DecryptMessage(encodingAESKey, encrypted, receiverID string) ([]byte, error) {
	key, err := decodeAESKey(encodingAESKey)
	if err != nil {
		return nil, err
	}
	ciphertext, err := decodeBase64(encrypted)
	if err != nil {
		return nil, err
	}
	if len(ciphertext) == 0 || len(ciphertext)%aes.BlockSize != 0 {
		return nil, ErrMalformedPayload
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidAESKey, err)
	}
	plain := make([]byte, len(ciphertext))
	cipher.NewCBCDecrypter(block, key[:aes.BlockSize]).CryptBlocks(plain, ciphertext)
	plain, err = unpad(plain, aes.BlockSize)
	if err != nil {
		return nil, err
	}
	if len(plain) < 20 {
		return nil, ErrMalformedPayload
	}
	length := binary.BigEndian.Uint32(plain[16:20])
	start, end := 20, uint64(20)+uint64(length)
	if end > uint64(len(plain)) {
		return nil, ErrMalformedPayload
	}
	receiverStart := int(end)
	if receiverStart > len(plain) {
		return nil, ErrMalformedPayload
	}
	message := plain[start:receiverStart]
	receiver := plain[receiverStart:]
	if receiverID != "" && string(receiver) != receiverID {
		return nil, ErrInvalidReceiver
	}
	return append([]byte(nil), message...), nil
}

func decodeAESKey(value string) ([]byte, error) {
	if value == "" {
		return nil, ErrInvalidAESKey
	}
	value += "==="[:(4-len(value)%4)%4]
	key, err := base64.StdEncoding.DecodeString(value)
	if err != nil || len(key) != 32 {
		return nil, ErrInvalidAESKey
	}
	return key, nil
}

func decodeBase64(value string) ([]byte, error) {
	decoded, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return nil, ErrInvalidBase64
	}
	return decoded, nil
}

func unpad(data []byte, blockSize int) ([]byte, error) {
	if len(data) == 0 || len(data)%blockSize != 0 {
		return nil, ErrInvalidPadding
	}
	padding := int(data[len(data)-1])
	if padding < 1 || padding > blockSize || padding > len(data) {
		return nil, ErrInvalidPadding
	}
	if !bytes.Equal(data[len(data)-padding:], bytes.Repeat([]byte{byte(padding)}, padding)) {
		return nil, ErrInvalidPadding
	}
	return data[:len(data)-padding], nil
}
