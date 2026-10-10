package webhook

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
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

// wechatPKCS7BlockSize is the block size mandated by WeChat's encrypted
// message protocol. It differs from AES's 16-byte cipher block size: AES-CBC
// still operates on 16-byte blocks, while the protocol applies PKCS#7
// padding in 32-byte units.
const wechatPKCS7BlockSize = 32

// DecryptMessage decrypts a WeChat encrypted message. encodingAESKey is the
// unpadded base64 key supplied by the platform and receiverID is the expected
// account or enterprise identifier.
func decryptMessage(encodingAESKey, encrypted, receiverID string) ([]byte, error) {
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
	plain, err = unpad(plain, wechatPKCS7BlockSize)
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
	if string(receiver) != receiverID {
		return nil, ErrInvalidReceiver
	}
	return append([]byte(nil), message...), nil
}

// EncryptMessage encrypts a WeChat callback response for receiverID.
func encryptMessage(
	encodingAESKey string,
	message []byte,
	receiverID string,
) (string, error) {
	key, err := decodeAESKey(encodingAESKey)
	if err != nil {
		return "", err
	}
	randomPrefix := make([]byte, 16)
	if _, readErr := rand.Read(randomPrefix); readErr != nil {
		return "", fmt.Errorf("webhook: generate encryption prefix: %w", readErr)
	}
	length := make([]byte, 4)
	binary.BigEndian.PutUint32(length, uint32(len(message)))
	plain := append(randomPrefix, length...)
	plain = append(plain, message...)
	plain = append(plain, receiverID...)
	padding := wechatPKCS7BlockSize - len(plain)%wechatPKCS7BlockSize
	plain = append(
		plain,
		bytes.Repeat([]byte{byte(padding)}, padding)...,
	)
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidAESKey, err)
	}
	ciphertext := make([]byte, len(plain))
	cipher.NewCBCEncrypter(block, key[:aes.BlockSize]).CryptBlocks(
		ciphertext,
		plain,
	)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
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

// CipherConfig explicitly chooses the expected receiver, including bot empty receivers.
type CipherConfig struct {
	EncodingAESKey, ReceiverID string
	AllowEmptyReceiver         bool
}
type MessageCipher struct{ key, receiver string }

func NewMessageCipher(cfg CipherConfig) (*MessageCipher, error) {
	if cfg.ReceiverID == "" && !cfg.AllowEmptyReceiver {
		return nil, ErrInvalidReceiver
	}
	if _, err := decodeAESKey(cfg.EncodingAESKey); err != nil {
		return nil, err
	}
	return &MessageCipher{key: cfg.EncodingAESKey, receiver: cfg.ReceiverID}, nil
}
func (c *MessageCipher) Encrypt(message []byte) (string, error) {
	if uint64(len(message)) > uint64(^uint32(0)) {
		return "", ErrMalformedPayload
	}
	return encryptMessage(c.key, message, c.receiver)
}
func (c *MessageCipher) Decrypt(encrypted string) ([]byte, error) {
	return decryptMessage(c.key, encrypted, c.receiver)
}
func EncryptMessage(key string, message []byte, receiver string) (string, error) {
	c, err := NewMessageCipher(CipherConfig{EncodingAESKey: key, ReceiverID: receiver})
	if err != nil {
		return "", err
	}
	return c.Encrypt(message)
}
func DecryptMessage(key, encrypted, receiver string) ([]byte, error) {
	c, err := NewMessageCipher(CipherConfig{EncodingAESKey: key, ReceiverID: receiver})
	if err != nil {
		return nil, err
	}
	return c.Decrypt(encrypted)
}
