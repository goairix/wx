package webhook

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// CallbackConfig contains the credentials used to verify a WeChat callback.
type CallbackConfig struct {
	ReceiverID     string
	Token          string
	EncodingAESKey string
}

// NewCallbackHandler verifies URL handshakes and callback signatures, decrypts
// encrypted messages, and delegates the decoded body to next.
func NewCallbackHandler(
	config CallbackConfig,
	next Handler,
	options ...Option,
) http.Handler {
	delegate := newHandler(next, options...)
	return &callbackHandler{
		config:      config,
		delegate:    delegate,
		configError: validateCallbackConfig(config),
	}
}

type callbackHandler struct {
	config      CallbackConfig
	delegate    *handler
	configError error
}

func (h *callbackHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if h.configError != nil {
		h.delegate.write(writer, h.delegate.errorResponse(h.configError))
		return
	}

	switch request.Method {
	case http.MethodGet:
		h.verifyURL(writer, request)
	case http.MethodPost:
		h.deliver(writer, request)
	default:
		writer.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (h *callbackHandler) verifyURL(writer http.ResponseWriter, request *http.Request) {
	query := request.URL.Query()
	timestamp := query.Get("timestamp")
	nonce := query.Get("nonce")
	echo := query.Get("echostr")

	var err error
	if signature := query.Get("msg_signature"); signature != "" {
		err = VerifyMessageSignature(h.config.Token, timestamp, nonce, echo, signature)
		if err == nil {
			var plain []byte
			plain, err = DecryptMessage(
				h.config.EncodingAESKey,
				echo,
				h.config.ReceiverID,
			)
			echo = string(plain)
		}
	} else {
		err = VerifySignature(
			h.config.Token,
			timestamp,
			nonce,
			query.Get("signature"),
		)
	}
	if err != nil {
		h.delegate.write(writer, h.delegate.errorResponse(err))
		return
	}

	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write([]byte(echo))
}

func (h *callbackHandler) deliver(writer http.ResponseWriter, request *http.Request) {
	body, err := io.ReadAll(request.Body)
	if err != nil {
		h.delegate.write(writer, h.delegate.errorResponse(err))
		return
	}
	body, encrypted, err := encryptedValue(body)
	if err != nil {
		h.delegate.write(writer, h.delegate.errorResponse(err))
		return
	}

	query := request.URL.Query()
	if encrypted == "" {
		err = VerifySignature(
			h.config.Token,
			query.Get("timestamp"),
			query.Get("nonce"),
			query.Get("signature"),
		)
	} else {
		err = VerifyMessageSignature(
			h.config.Token,
			query.Get("timestamp"),
			query.Get("nonce"),
			encrypted,
			query.Get("msg_signature"),
		)
		if err == nil {
			body, err = DecryptMessage(
				h.config.EncodingAESKey,
				encrypted,
				h.config.ReceiverID,
			)
		}
	}
	if err != nil {
		h.delegate.write(writer, h.delegate.errorResponse(err))
		return
	}

	request.Body = io.NopCloser(bytes.NewReader(body))
	h.delegate.ServeHTTP(writer, request)
}

func validateCallbackConfig(config CallbackConfig) error {
	if strings.TrimSpace(config.Token) == "" {
		return errors.New("webhook: token is required")
	}
	if config.EncodingAESKey == "" {
		return nil
	}
	value := config.EncodingAESKey
	value += "==="[:(4-len(value)%4)%4]
	key, err := base64.StdEncoding.DecodeString(value)
	if err != nil || len(key) != 32 {
		return ErrInvalidAESKey
	}
	return nil
}

func encryptedValue(body []byte) ([]byte, string, error) {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		return body, "", nil
	}
	if trimmed[0] == '{' {
		var envelope struct {
			Encrypted string `json:"Encrypt"`
		}
		if err := json.Unmarshal(body, &envelope); err != nil {
			return nil, "", fmt.Errorf("parse encrypted webhook json: %w", err)
		}
		return body, envelope.Encrypted, nil
	}
	var envelope struct {
		Encrypted string `xml:"Encrypt"`
	}
	if err := xml.Unmarshal(body, &envelope); err != nil {
		return nil, "", fmt.Errorf("parse encrypted webhook xml: %w", err)
	}
	return body, envelope.Encrypted, nil
}
