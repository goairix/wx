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
	"strconv"
	"strings"
	"time"

	"github.com/goairix/wx/v2/core/random"
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
	query := request.URL.Query()
	isEncrypted := query.Get("msg_signature") != ""
	if !isEncrypted {
		err := VerifySignature(
			h.config.Token,
			query.Get("timestamp"),
			query.Get("nonce"),
			query.Get("signature"),
		)
		if err != nil {
			h.delegate.write(writer, h.delegate.errorResponse(err))
			return
		}
	}

	body, err := readBody(request.Body)
	if err != nil {
		h.delegate.write(writer, h.delegate.errorResponse(err))
		return
	}
	body, encrypted, err := encryptedValue(body)
	if err != nil {
		h.delegate.write(writer, h.delegate.errorResponse(err))
		return
	}

	if encrypted == "" {
		if isEncrypted {
			err = ErrInvalidSignature
		}
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
	if encrypted == "" {
		h.delegate.ServeHTTP(writer, request)
		return
	}
	h.deliverEncryptedResponse(writer, request)
}

func validateCallbackConfig(config CallbackConfig) error {
	if strings.TrimSpace(config.Token) == "" {
		return errors.New("webhook: token is required")
	}
	if config.EncodingAESKey == "" {
		return nil
	}
	if strings.TrimSpace(config.ReceiverID) == "" {
		return ErrInvalidReceiver
	}
	value := config.EncodingAESKey
	value += "==="[:(4-len(value)%4)%4]
	key, err := base64.StdEncoding.DecodeString(value)
	if err != nil || len(key) != 32 {
		return ErrInvalidAESKey
	}
	return nil
}

func (h *callbackHandler) deliverEncryptedResponse(
	writer http.ResponseWriter,
	request *http.Request,
) {
	buffer := newResponseBuffer()
	h.delegate.ServeHTTP(buffer, request)
	if buffer.body.Len() == 0 {
		buffer.writeTo(writer)
		return
	}

	encrypted, err := EncryptMessage(
		h.config.EncodingAESKey,
		buffer.body.Bytes(),
		h.config.ReceiverID,
	)
	if err != nil {
		h.delegate.write(writer, h.delegate.errorResponse(err))
		return
	}
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	nonce, err := random.String(16)
	if err != nil {
		h.delegate.write(writer, h.delegate.errorResponse(err))
		return
	}
	envelope := encryptedResponse{
		Encrypt: cdata{Value: encrypted},
		Signature: cdata{Value: MessageSignature(
			h.config.Token,
			timestamp,
			nonce,
			encrypted,
		)},
		Timestamp: timestamp,
		Nonce:     cdata{Value: nonce},
	}
	body, err := xml.Marshal(envelope)
	if err != nil {
		h.delegate.write(writer, h.delegate.errorResponse(err))
		return
	}
	copyHeader(writer.Header(), buffer.header)
	writer.Header().Set("Content-Type", "application/xml; charset=utf-8")
	writer.Header().Set("Content-Length", strconv.Itoa(len(body)))
	writer.WriteHeader(buffer.statusCode())
	_, _ = writer.Write(body)
}

type encryptedResponse struct {
	XMLName   xml.Name `xml:"xml"`
	Encrypt   cdata    `xml:"Encrypt"`
	Signature cdata    `xml:"MsgSignature"`
	Timestamp string   `xml:"TimeStamp"`
	Nonce     cdata    `xml:"Nonce"`
}

type cdata struct {
	Value string `xml:",cdata"`
}

type responseBuffer struct {
	header http.Header
	status int
	body   bytes.Buffer
}

func newResponseBuffer() *responseBuffer {
	return &responseBuffer{header: make(http.Header)}
}

func (r *responseBuffer) Header() http.Header {
	return r.header
}

func (r *responseBuffer) WriteHeader(status int) {
	if r.status == 0 {
		r.status = status
	}
}

func (r *responseBuffer) Write(body []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	return r.body.Write(body)
}

func (r *responseBuffer) statusCode() int {
	if r.status == 0 {
		return http.StatusOK
	}
	return r.status
}

func (r *responseBuffer) writeTo(writer http.ResponseWriter) {
	copyHeader(writer.Header(), r.header)
	writer.WriteHeader(r.statusCode())
	if r.body.Len() > 0 {
		_, _ = writer.Write(r.body.Bytes())
	}
}

func copyHeader(destination, source http.Header) {
	for key, values := range source {
		for _, value := range values {
			destination.Add(key, value)
		}
	}
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
