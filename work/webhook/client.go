// Package webhook contains typed enterprise WeChat callback handling.
package webhook

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"

	corewebhook "github.com/goairix/wx/v2/core/webhook"
)

// Client stores enterprise callback credentials.
type Client struct {
	corpID         string
	token          string
	encodingAESKey string
}

// NewClient constructs an enterprise callback adapter.
func NewClient(corpID, token, encodingAESKey string) *Client {
	return &Client{
		corpID:         corpID,
		token:          token,
		encodingAESKey: encodingAESKey,
	}
}

// Event contains the fields shared by supported enterprise callbacks. Fields
// that do not apply to a particular event remain at their zero value.
type Event struct {
	ToUserName     string `xml:"ToUserName" json:"ToUserName"`
	FromUserName   string `xml:"FromUserName" json:"FromUserName"`
	CreateTime     int64  `xml:"CreateTime" json:"CreateTime"`
	MessageType    string `xml:"MsgType" json:"MsgType"`
	Event          string `xml:"Event" json:"Event"`
	ChangeType     string `xml:"ChangeType" json:"ChangeType"`
	AgentID        int64  `xml:"AgentID" json:"AgentID"`
	SuiteID        string `xml:"SuiteId" json:"SuiteId"`
	SuiteTicket    string `xml:"SuiteTicket" json:"SuiteTicket"`
	AuthCode       string `xml:"AuthCode" json:"AuthCode"`
	TimeStamp      int64  `xml:"TimeStamp" json:"TimeStamp"`
	UserID         string `xml:"UserID" json:"UserID"`
	NewUserID      string `xml:"NewUserID" json:"NewUserID"`
	Name           string `xml:"Name" json:"Name"`
	Department     string `xml:"Department" json:"Department"`
	PartyID        int    `xml:"Id" json:"Id"`
	ParentID       int    `xml:"ParentId" json:"ParentId"`
	TagID          int    `xml:"TagId" json:"TagId"`
	ExternalUserID string `xml:"ExternalUserID" json:"ExternalUserID"`
	State          string `xml:"State" json:"State"`
	WelcomeCode    string `xml:"WelcomeCode" json:"WelcomeCode"`
	ChatID         string `xml:"ChatId" json:"ChatId"`
	UpdateDetail   string `xml:"UpdateDetail" json:"UpdateDetail"`
	JoinScene      int    `xml:"JoinScene" json:"JoinScene"`
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

// Handler processes a typed enterprise callback event.
type Handler interface {
	Handle(context.Context, Event) (corewebhook.Response, error)
}

// HandlerFunc adapts a function to Handler.
type HandlerFunc func(context.Context, Event) (corewebhook.Response, error)

// Handle calls f with the decoded event.
func (f HandlerFunc) Handle(
	ctx context.Context,
	event Event,
) (corewebhook.Response, error) {
	return f(ctx, event)
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

// Handler returns an HTTP adapter that verifies and decodes callbacks.
func (c *Client) Handler(next Handler, options ...Option) http.Handler {
	return c.RawHandler(
		corewebhook.HandlerFunc(func(
			ctx context.Context,
			payload corewebhook.Payload,
		) (corewebhook.Response, error) {
			var event Event
			if err := decode(payload, &event); err != nil {
				return corewebhook.Response{}, err
			}
			return next.Handle(ctx, event)
		}),
		options...,
	)
}

// RawHandler returns an HTTP adapter that retains the generic core payload.
func (c *Client) RawHandler(
	next corewebhook.Handler,
	options ...Option,
) http.Handler {
	settings := new(handlerOptions)
	for _, configure := range options {
		if configure != nil {
			configure(settings)
		}
	}
	coreOptions := make([]corewebhook.Option, 0, 1)
	if settings.errorResponse != nil {
		coreOptions = append(
			coreOptions,
			corewebhook.WithErrorResponse(settings.errorResponse),
		)
	}
	return corewebhook.NewCallbackHandler(
		corewebhook.CallbackConfig{
			ReceiverID:     c.corpID,
			Token:          c.token,
			EncodingAESKey: c.encodingAESKey,
		},
		next,
		coreOptions...,
	)
}

func decode(payload corewebhook.Payload, target interface{}) error {
	var err error
	if payload.Format == "json" {
		err = json.Unmarshal(payload.Raw, target)
	} else {
		err = xml.Unmarshal(payload.Raw, target)
	}
	if err != nil {
		return fmt.Errorf("decode work webhook event: %w", err)
	}
	return nil
}
