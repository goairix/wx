package encryptor

import (
	"encoding/base64"
	"encoding/json"

	"github.com/goairix/wx/v2/kernel/contracts"
	"github.com/goairix/wx/v2/support/aes"
	"github.com/pkg/errors"
)

// Encryptor 小程序加密数据的解密
type Encryptor struct {
	account contracts.AccountInterface
}

func New(account contracts.AccountInterface) *Encryptor {
	return &Encryptor{account: account}
}

func (e Encryptor) Decrypt(sessionKey, iv, encryptedData string) (map[string]interface{}, error) {
	encryptedDataByte, _ := base64.StdEncoding.DecodeString(encryptedData)
	ivByte, _ := base64.StdEncoding.DecodeString(iv)
	sessionKeyByte, _ := base64.StdEncoding.DecodeString(sessionKey)

	decrypted, err := aes.Decrypt(
		encryptedDataByte,
		sessionKeyByte,
		ivByte,
	)
	if err != nil {
		return nil, err
	}

	var decryptedData map[string]interface{}
	err = json.Unmarshal(decrypted, &decryptedData)
	if err != nil {
		return nil, errors.New("The given payload is invalid.")
	}

	return decryptedData, nil
}
