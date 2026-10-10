package websocket

import (
	"encoding/json"

	"github.com/goairix/wx/v2/work/aibot"
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
	ErrClosed               = aibot.ErrClosed
	ErrNotReady             = aibot.ErrNotReady
	ErrDisconnected         = aibot.ErrDisconnected
	ErrReplaced             = aibot.ErrReplaced
	ErrNotificationOverflow = aibot.ErrNotificationOverflow
	ErrInvalidArgument      = aibot.ErrInvalidArgument
	ErrProtocol             = aibot.ErrProtocol
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
	return aibot.WrapError("websocket."+operation, reqID, cause)
}
