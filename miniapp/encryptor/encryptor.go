// Package encryptor decrypts encrypted miniapp session payloads.
package encryptor

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"

	wxerrors "github.com/goairix/wx/v2/core/errors"
)

type Encryptor struct{}

func New() *Encryptor {
	return &Encryptor{}
}
func (Encryptor) Decrypt(sessionKey, iv, encryptedData string) (map[string]interface{}, error) {
	key, err := base64.StdEncoding.DecodeString(sessionKey)
	if err != nil {
		return nil, err
	}
	ivb, err := base64.StdEncoding.DecodeString(iv)
	if err != nil {
		return nil, err
	}
	data, err := base64.StdEncoding.DecodeString(encryptedData)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 || len(data)%block.BlockSize() != 0 || len(ivb) != block.BlockSize() {
		return nil, wxerrors.New("miniapp encryptor: invalid encrypted payload")
	}
	out := make([]byte, len(data))
	cipher.NewCBCDecrypter(block, ivb).CryptBlocks(out, data)
	pad := int(out[len(out)-1])
	if pad == 0 || pad > block.BlockSize() || pad > len(out) {
		return nil, wxerrors.New("miniapp encryptor: invalid encrypted payload")
	}
	out = out[:len(out)-pad]
	var result map[string]interface{}
	if err := json.Unmarshal(out, &result); err != nil {
		return nil, wxerrors.New("miniapp encryptor: invalid payload")
	}
	return result, nil
}
