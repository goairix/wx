package websocket

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/goairix/wx/v2/work/aibot"
)

type ReceivedCallback struct {
	Callback                 *aibot.Callback
	Frame                    Frame
	owner                    *Client
	session                  *session
	reqID, eventType, taskID string
	state                    *replyState
}

// ReceivedAt is captured before parsing or queuing the incoming frame.
func (r *ReceivedCallback) ReceivedAt() time.Time {
	if r == nil || r.state == nil {
		return time.Time{}
	}
	return r.state.receivedAt
}

// ReplyDeadline is the local protocol send deadline; a zero value means that
// this event cannot be replied to. Server-side deadlines remain authoritative.
func (r *ReceivedCallback) ReplyDeadline() time.Time {
	if r == nil || r.state == nil {
		return time.Time{}
	}
	r.state.mu.Lock()
	defer r.state.mu.Unlock()
	return r.state.deadline
}

func (s *session) received(frame Frame, receivedAt time.Time) (*ReceivedCallback, error) {
	cb, err := aibot.ParseCallback(frame.Body)
	if err != nil {
		return nil, err
	}
	state := &replyState{receivedAt: receivedAt, streams: make(map[string]*streamState)}
	target := &ReceivedCallback{Callback: cb, Frame: frame, owner: s.owner, session: s, reqID: frame.Headers.ReqID, state: state}
	if cb.Message != nil {
		state.deadline = receivedAt.Add(24 * time.Hour)
		state.chatType = cb.Message.ChatType
		state.chatID = cb.Message.From.UserID
		if state.chatType == "group" {
			state.chatID = cb.Message.ChatID
		}
	}
	if cb.Event != nil {
		state.chatType = cb.Event.ChatType
		state.chatID = cb.Event.From.UserID
		if state.chatType == "group" {
			state.chatID = cb.Event.ChatID
		}
		target.eventType = cb.Event.Event.EventType
		if target.eventType == "enter_chat" || target.eventType == "template_card_event" {
			state.deadline = receivedAt.Add(5 * time.Second)
		}
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
	valid := target != nil && target.owner == c && target.session != nil && target.session == c.session && !c.closed && target.session.ctx.Err() == nil && target.reqID != "" && target.state != nil
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
	policy := &sendPolicy{state: target.state}
	if m.Stream != nil {
		policy.streamID = m.Stream.ID
		policy.finish = m.Stream.Finish
	}
	bound = context.WithValue(bound, sendPolicyKey{}, policy)
	if err := policy.check(); err != nil {
		return Frame{}, aibot.WrapError("websocket.reply", target.reqID, err)
	}
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
	if ctx == nil {
		return Frame{}, aibot.WrapError("websocket.send", "", aibot.ErrInvalidArgument)
	}
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
	chatType := "auto"
	switch m.ChatType {
	case 1:
		chatType = "single"
	case 2:
		chatType = "group"
	}
	ctx = context.WithValue(ctx, limitTargetKey{}, LimitRequest{ConversationID: chatID, ConversationType: chatType})
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
