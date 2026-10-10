package aibot

import "encoding/json"

type StreamRefresh struct {
	ID string `json:"id"`
}
type Callback struct {
	Message *Message
	Event   *EventMessage
	Raw     json.RawMessage
}

func ParseCallback(data []byte) (*Callback, error) {
	var envelope struct {
		MsgType string `json:"msgtype"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, WrapError("parse_callback", "", err)
	}
	if envelope.MsgType == "" {
		return nil, WrapError("parse_callback", "", ErrProtocol)
	}
	cb := &Callback{Raw: append(json.RawMessage(nil), data...)}
	if envelope.MsgType == "event" {
		cb.Event = new(EventMessage)
		if err := json.Unmarshal(data, cb.Event); err != nil {
			return nil, WrapError("parse_callback", "", err)
		}
	} else {
		cb.Message = new(Message)
		if err := json.Unmarshal(data, cb.Message); err != nil {
			return nil, WrapError("parse_callback", "", err)
		}
	}
	return cb, nil
}
