package websocket

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"

	"github.com/goairix/wx/v2/work/aibot"
)

type ReceivedCallback struct {
	Callback                 *aibot.Callback
	Frame                    Frame
	owner                    *Client
	session                  *session
	reqID, eventType, taskID string
}

func (s *session) received(frame Frame) (*ReceivedCallback, error) {
	cb, err := aibot.ParseCallback(frame.Body)
	if err != nil {
		return nil, err
	}
	target := &ReceivedCallback{Callback: cb, Frame: frame, owner: s.owner, session: s, reqID: frame.Headers.ReqID}
	if cb.Event != nil {
		target.eventType = cb.Event.Event.EventType
		if cb.Event.Event.TemplateCardEvent != nil {
			target.taskID = cb.Event.Event.TemplateCardEvent.TaskID
		}
	}
	return target, nil
}
func (c *Client) Reply(ctx context.Context, target *ReceivedCallback, reply *aibot.Reply) (Frame, error) {
	if ctx == nil {
		return Frame{}, aibot.WrapError("websocket.reply", "", aibot.ErrInvalidArgument)
	}
	c.mu.Lock()
	valid := target != nil && target.owner == c && target.session != nil && target.session == c.session && !c.closed && target.session.ctx.Err() == nil && target.reqID != ""
	c.mu.Unlock()
	if !valid {
		return Frame{}, aibot.WrapError("websocket.reply", "", aibot.ErrInvalidTarget)
	}
	if reply == nil {
		return Frame{}, nil
	}
	m := reply.OutgoingMessage
	if m.MsgType == "stream_with_template_card" || (m.Stream != nil && len(m.Stream.MsgItem) > 0) {
		return Frame{}, aibot.WrapError("websocket.reply", target.reqID, aibot.ErrUnsupported)
	}
	if err := aibot.ValidateMessage(m); err != nil {
		return Frame{}, aibot.WrapError("websocket.reply", target.reqID, err)
	}
	bound := context.WithValue(ctx, sessionContextKey{}, target.session)
	body, err := legacyBody(m)
	if err != nil {
		return Frame{}, aibot.WrapError("websocket.reply", target.reqID, err)
	}
	switch target.eventType {
	case "enter_chat":
		return c.replyWelcome(bound, target.reqID, body)
	case "template_card_event":
		if m.MsgType != "template_card" || target.taskID == "" {
			return Frame{}, aibot.WrapError("websocket.reply", target.reqID, aibot.ErrUnsupported)
		}
		card, err := aibot.CloneCardWithTask(m.TemplateCard, target.taskID)
		if err != nil {
			return Frame{}, err
		}
		var wire legacyCard
		decoder := json.NewDecoder(bytes.NewReader(card))
		decoder.UseNumber()
		if err = decoder.Decode(&wire); err != nil {
			return Frame{}, err
		}
		return c.updatelegacyCard(bound, target.reqID, wire, reply.UserIDs)
	case "":
		return c.reply(bound, target.reqID, body)
	default:
		return Frame{}, aibot.WrapError("websocket.reply", target.reqID, aibot.ErrUnsupported)
	}
}
func (c *Client) Send(ctx context.Context, chatID string, m aibot.OutgoingMessage) (Frame, error) {
	if m.MsgType != "markdown" && m.MsgType != "template_card" && m.MsgType != "image" && m.MsgType != "voice" && m.MsgType != "video" && m.MsgType != "file" {
		return Frame{}, aibot.WrapError("websocket.send", "", aibot.ErrUnsupported)
	}
	if err := aibot.ValidateMessage(m); err != nil {
		return Frame{}, aibot.WrapError("websocket.send", "", err)
	}
	body, err := legacyBody(m)
	if err != nil {
		return Frame{}, err
	}
	return c.send(ctx, chatID, body)
}
func legacyBody(m aibot.OutgoingMessage) (MessageBody, error) {
	var body MessageBody
	b, err := json.Marshal(m)
	if err == nil {
		decoder := json.NewDecoder(bytes.NewReader(b))
		decoder.UseNumber()
		err = decoder.Decode(&body)
	}
	if err != nil {
		return body, errors.Join(aibot.ErrInvalidArgument, err)
	}
	return body, nil
}
