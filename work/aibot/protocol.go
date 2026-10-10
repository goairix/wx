package aibot

import (
	"encoding/json"
	"errors"

	coreerrors "github.com/goairix/wx/v2/core/errors"
)

const (
	CmdSubscribe       = "aibot_subscribe"
	CmdPing            = "ping"
	CmdRespond         = "aibot_respond_msg"
	CmdWelcome         = "aibot_respond_welcome_msg"
	CmdUpdate          = "aibot_respond_update_msg"
	CmdSend            = "aibot_send_msg"
	CmdUploadInit      = "aibot_upload_media_init"
	CmdUploadChunk     = "aibot_upload_media_chunk"
	CmdUploadFinish    = "aibot_upload_media_finish"
	CmdMessageCallback = "aibot_msg_callback"
	CmdEventCallback   = "aibot_event_callback"
)

var (
	ErrClosed               = errors.New("aibot client closed")
	ErrNotReady             = errors.New("aibot client not ready")
	ErrDisconnected         = errors.New("aibot connection lost")
	ErrReplaced             = errors.New("aibot connection replaced")
	ErrNotificationOverflow = errors.New("aibot notification buffer full")
	ErrInvalidArgument      = errors.New("invalid aibot argument")
	ErrProtocol             = errors.New("invalid aibot protocol frame")
)

// Headers identifies a request or callback. Additional headers remain in Frame.Raw.
type Headers struct {
	ReqID string `json:"req_id"`
}

// Frame preserves both the typed envelope and the original wire bytes.
type Frame struct {
	Command string          `json:"cmd,omitempty"`
	Headers Headers         `json:"headers"`
	Body    json.RawMessage `json:"body,omitempty"`
	ErrCode *int            `json:"errcode,omitempty"`
	ErrMsg  string          `json:"errmsg,omitempty"`
	Raw     json.RawMessage `json:"-"`
}

func operationError(operation, reqID string, cause error) error {
	wrapped := &coreerrors.Error{Platform: "work", Operation: "aibot." + operation, RequestID: reqID, Message: "intelligent bot operation failed", Err: cause}
	var inner *coreerrors.Error
	if errors.As(cause, &inner) {
		wrapped.Code = inner.Code
		wrapped.HTTPStatus = inner.HTTPStatus
		if reqID == "" {
			wrapped.RequestID = inner.RequestID
		}
	}
	return wrapped
}
