// Package webhook contains enterprise WeChat callback handling.
package webhook

import (
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	corewebhook "github.com/goairix/wx/v2/core/webhook"
)

// Client stores enterprise callback credentials.
type Client struct {
	corpID         string
	token          string
	encodingAESKey string
}

func NewClient(corpID, token, encodingAESKey string) *Client {
	return &Client{
		corpID:         corpID,
		token:          token,
		encodingAESKey: encodingAESKey,
	}
}

// Event contains common enterprise callback fields.
type Event struct {
	ToUserName   string `xml:"ToUserName" json:"ToUserName"`
	FromUserName string `xml:"FromUserName" json:"FromUserName"`
	CreateTime   int64  `xml:"CreateTime" json:"CreateTime"`
	MessageType  string `xml:"MsgType" json:"MsgType"`
	Event        string `xml:"Event" json:"Event"`
	ChangeType   string `xml:"ChangeType" json:"ChangeType"`
	AgentID      int64  `xml:"AgentID" json:"AgentID"`
}

// SuiteTicketEvent represents a third-party suite ticket callback.
type SuiteTicketEvent struct {
	Event
	SuiteID     string `xml:"SuiteId" json:"SuiteId"`
	SuiteTicket string `xml:"SuiteTicket" json:"SuiteTicket"`
}

// AuthorizationEvent represents authorization lifecycle callbacks.
type AuthorizationEvent struct {
	Event
	AuthCode  string `xml:"AuthCode" json:"AuthCode"`
	SuiteID   string `xml:"SuiteId" json:"SuiteId"`
	TimeStamp int64  `xml:"TimeStamp" json:"TimeStamp"`
}

// ContactChangeEvent represents member, department, and tag changes.
type ContactChangeEvent struct {
	Event
	UserID     string `xml:"UserID" json:"UserID"`
	NewUserID  string `xml:"NewUserID" json:"NewUserID"`
	Name       string `xml:"Name" json:"Name"`
	Department string `xml:"Department" json:"Department"`
	PartyID    int    `xml:"Id" json:"Id"`
	ParentID   int    `xml:"ParentId" json:"ParentId"`
	TagID      int    `xml:"TagId" json:"TagId"`
}

// ExternalContactChangeEvent represents external contact lifecycle changes.
type ExternalContactChangeEvent struct {
	Event
	UserID         string `xml:"UserID" json:"UserID"`
	ExternalUserID string `xml:"ExternalUserID" json:"ExternalUserID"`
	State          string `xml:"State" json:"State"`
	WelcomeCode    string `xml:"WelcomeCode" json:"WelcomeCode"`
}

// GroupChatChangeEvent represents customer group chat lifecycle changes.
type GroupChatChangeEvent struct {
	Event
	ChatID       string `xml:"ChatId" json:"ChatId"`
	UpdateDetail string `xml:"UpdateDetail" json:"UpdateDetail"`
	JoinScene    int    `xml:"JoinScene" json:"JoinScene"`
}

// BatchJobEvent represents completion of an asynchronous contact job.
type BatchJobEvent struct {
	Event
	BatchJob struct {
		JobID   string `xml:"JobId" json:"JobId"`
		JobType string `xml:"JobType" json:"JobType"`
		ErrCode int    `xml:"ErrCode" json:"ErrCode"`
		ErrMsg  string `xml:"ErrMsg" json:"ErrMsg"`
	} `xml:"BatchJob" json:"BatchJob"`
}

type handlerOptions struct {
	errorResponse corewebhook.ErrorResponse
}

// Option configures the enterprise callback adapter.
type Option func(*handlerOptions)

// WithErrorResponse configures callback error responses.
func WithErrorResponse(policy corewebhook.ErrorResponse) Option {
	return func(options *handlerOptions) {
		options.errorResponse = policy
	}
}

// Handler returns an HTTP adapter for URL verification and callback delivery.
func (c *Client) Handler(next corewebhook.Handler, options ...Option) http.Handler {
	settings := handlerOptions{
		errorResponse: func(error) corewebhook.Response {
			return corewebhook.Response{Status: http.StatusBadRequest}
		},
	}
	for _, configure := range options {
		if configure != nil {
			configure(&settings)
		}
	}
	return &callbackHandler{
		client:        c,
		next:          next,
		errorResponse: settings.errorResponse,
		configError:   c.validate(),
	}
}

type callbackHandler struct {
	client        *Client
	next          corewebhook.Handler
	errorResponse corewebhook.ErrorResponse
	configError   error
}

func (h *callbackHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if h.configError != nil {
		h.writeError(writer, h.configError)
		return
	}
	if request.Method == http.MethodGet {
		h.verifyURL(writer, request)
		return
	}
	if request.Method != http.MethodPost {
		writer.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	body, err := io.ReadAll(request.Body)
	if err != nil {
		h.writeError(writer, err)
		return
	}
	body, err = h.decode(request, body)
	if err != nil {
		h.writeError(writer, err)
		return
	}
	request.Body = io.NopCloser(bytes.NewReader(body))
	delegate := corewebhook.NewHandler(
		h.next,
		corewebhook.WithErrorResponse(h.errorResponse),
	)
	delegate.ServeHTTP(writer, request)
}

func (c *Client) validate() error {
	if c == nil || strings.TrimSpace(c.token) == "" {
		return errors.New("work webhook: token is required")
	}
	if c.encodingAESKey == "" {
		return nil
	}
	encodedKey := c.encodingAESKey
	encodedKey += "==="[:(4-len(encodedKey)%4)%4]
	key, err := base64.StdEncoding.DecodeString(encodedKey)
	if err != nil || len(key) != 32 {
		return errors.New("work webhook: invalid encoding AES key")
	}
	return nil
}

func (h *callbackHandler) verifyURL(writer http.ResponseWriter, request *http.Request) {
	query := request.URL.Query()
	timestamp := query.Get("timestamp")
	nonce := query.Get("nonce")
	echo := query.Get("echostr")
	signature := query.Get("msg_signature")
	if signature != "" {
		err := corewebhook.VerifyMessageSignature(
			h.client.token,
			timestamp,
			nonce,
			echo,
			signature,
		)
		if err == nil {
			var decoded []byte
			decoded, err = corewebhook.DecryptMessage(
				h.client.encodingAESKey,
				echo,
				h.client.corpID,
			)
			if err == nil {
				echo = string(decoded)
			}
		}
		if err != nil {
			h.writeError(writer, err)
			return
		}
	} else if err := corewebhook.VerifySignature(
		h.client.token,
		timestamp,
		nonce,
		query.Get("signature"),
	); err != nil {
		h.writeError(writer, err)
		return
	}
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write([]byte(echo))
}

func (h *callbackHandler) decode(request *http.Request, body []byte) ([]byte, error) {
	var envelope struct {
		Encrypted string `xml:"Encrypt"`
	}
	if err := xml.Unmarshal(body, &envelope); err != nil || envelope.Encrypted == "" {
		err := corewebhook.VerifySignature(
			h.client.token,
			request.URL.Query().Get("timestamp"),
			request.URL.Query().Get("nonce"),
			request.URL.Query().Get("signature"),
		)
		return body, err
	}
	query := request.URL.Query()
	err := corewebhook.VerifyMessageSignature(
		h.client.token,
		query.Get("timestamp"),
		query.Get("nonce"),
		envelope.Encrypted,
		query.Get("msg_signature"),
	)
	if err != nil {
		return nil, err
	}
	plain, err := corewebhook.DecryptMessage(
		h.client.encodingAESKey,
		envelope.Encrypted,
		h.client.corpID,
	)
	if err != nil {
		return nil, fmt.Errorf("decrypt work callback: %w", err)
	}
	return plain, nil
}

func (h *callbackHandler) writeError(writer http.ResponseWriter, err error) {
	response := h.errorResponse(err)
	for key, values := range response.Header {
		for _, value := range values {
			writer.Header().Add(key, value)
		}
	}
	status := response.Status
	if status == 0 {
		status = http.StatusBadRequest
	}
	writer.WriteHeader(status)
	_, _ = writer.Write(response.Body)
}
