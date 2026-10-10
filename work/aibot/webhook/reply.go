package webhook

import (
	"encoding/json"
	"reflect"
	"strconv"
	"time"

	corehook "github.com/goairix/wx/v2/core/webhook"
	"github.com/goairix/wx/v2/work/aibot"
)

func (c *Client) encodeReply(cb *aibot.Callback, reply *aibot.Reply, nonce string) ([]byte, error) {
	if cb == nil || reply == nil || len(nonce) > 256 {
		return nil, aibot.ErrInvalidArgument
	}
	// Check bulk strings and base64 before validation/JSON copies.
	if !withinBudget(reflect.ValueOf(reply), c.config.MaxPlainReplyBytes) {
		return nil, aibot.ErrBodyTooLarge
	}
	m := reply.OutgoingMessage
	if err := aibot.ValidateMessage(m); err != nil {
		return nil, err
	}
	var payload any = m
	if cb.Event != nil {
		switch cb.Event.Event.EventType {
		case "enter_chat":
			if m.MsgType != "text" && m.MsgType != "template_card" {
				return nil, aibot.ErrUnsupported
			}
		case "template_card_event":
			event := cb.Event.Event.TemplateCardEvent
			if m.MsgType != "template_card" || event == nil || event.TaskID == "" {
				return nil, aibot.ErrUnsupported
			}
			card, err := aibot.CloneCardWithTask(m.TemplateCard, event.TaskID)
			if err != nil {
				return nil, err
			}
			payload = struct {
				ResponseType string                `json:"response_type"`
				UserIDs      []string              `json:"userids,omitempty"`
				TemplateCard aibot.RawTemplateCard `json:"template_card"`
			}{"update_template_card", reply.UserIDs, card}
		default:
			return nil, aibot.ErrUnsupported
		}
	} else if cb.Message != nil {
		if m.MsgType != "stream" && m.MsgType != "template_card" && m.MsgType != "stream_with_template_card" {
			return nil, aibot.ErrUnsupported
		}
		if cb.Message.MsgType == "stream" && (m.Stream == nil || cb.Message.Stream == nil || m.Stream.ID != cb.Message.Stream.ID) {
			return nil, aibot.ErrInvalidTarget
		}
	} else {
		return nil, aibot.ErrInvalidArgument
	}
	plain, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	if int64(len(plain)) > c.config.MaxPlainReplyBytes {
		return nil, aibot.ErrBodyTooLarge
	}
	encrypted, err := c.cipher.Encrypt(plain)
	if err != nil {
		return nil, err
	}
	timestamp := time.Now().Unix()
	wire, err := json.Marshal(struct {
		Encrypt   string `json:"encrypt"`
		Signature string `json:"msgsignature"`
		Timestamp int64  `json:"timestamp"`
		Nonce     string `json:"nonce"`
	}{encrypted, corehook.MessageSignature(c.config.Token, strconv.FormatInt(timestamp, 10), nonce, encrypted), timestamp, nonce})
	if int64(len(wire)) > c.config.MaxWireReplyBytes {
		return nil, aibot.ErrBodyTooLarge
	}
	return wire, err
}
func withinBudget(v reflect.Value, budget int64) bool {
	seen := make(map[uintptr]bool)
	var visit func(reflect.Value, int) bool
	visit = func(v reflect.Value, depth int) bool {
		if !v.IsValid() {
			return true
		}
		if depth > 64 {
			return false
		}
		switch v.Kind() {
		case reflect.Interface:
			if !v.IsNil() {
				return visit(v.Elem(), depth+1)
			}
		case reflect.Pointer:
			if !v.IsNil() {
				p := v.Pointer()
				if seen[p] {
					return false
				}
				seen[p] = true
				defer delete(seen, p)
				return visit(v.Elem(), depth+1)
			}
		case reflect.String:
			budget -= int64(v.Len())
		case reflect.Slice:
			if v.Type().Elem().Kind() == reflect.Uint8 {
				budget -= int64(v.Len())
				break
			}
			fallthrough
		case reflect.Array:
			for i := 0; i < v.Len(); i++ {
				if !visit(v.Index(i), depth+1) {
					return false
				}
			}
		case reflect.Struct:
			for i := 0; i < v.NumField(); i++ {
				if !visit(v.Field(i), depth+1) {
					return false
				}
			}
		case reflect.Map:
			iter := v.MapRange()
			for iter.Next() {
				if !visit(iter.Key(), depth+1) || !visit(iter.Value(), depth+1) {
					return false
				}
			}
		}
		return budget >= 0
	}
	return visit(v, 0)
}
