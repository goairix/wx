package aibot

import (
	"context"
	"fmt"
	"unicode/utf8"
)

// TemplateCard retains all documented and future template-card parameters.
// The server validates the fields appropriate to each card_type.
type TemplateCard map[string]any

// Feedback supplies an identifier returned in feedback callbacks.
type Feedback struct {
	ID string `json:"id"`
}

// MediaContent references a temporary uploaded image, file, voice or video.
// Title and Description apply to video messages.
type MediaContent struct {
	MediaID     string `json:"media_id"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
}

// StreamImage carries an inline image and the MD5 of its decoded data.
type StreamImage struct {
	Base64 string `json:"base64"`
	MD5    string `json:"md5"`
}

// StreamItem describes an inline image in the final stream update.
type StreamItem struct {
	MsgType string      `json:"msgtype"`
	Image   StreamImage `json:"image"`
}

// Stream contains the full current Markdown text, rather than a text delta.
// MsgItem is retained for protocol compatibility; current long-connection
// servers may reject inline images even in a final update.
type Stream struct {
	ID       string       `json:"id"`
	Finish   bool         `json:"finish"`
	Content  string       `json:"content"`
	MsgItem  []StreamItem `json:"msg_item,omitempty"`
	Feedback *Feedback    `json:"feedback,omitempty"`
}

// MessageBody is an outgoing body. Set MsgType and its matching payload.
// ChatType selects userid (1), group (2), or automatic (0) resolution for Send.
type MessageBody struct {
	MsgType      string        `json:"msgtype"`
	ChatType     int           `json:"chat_type,omitempty"`
	Markdown     *TextContent  `json:"markdown,omitempty"`
	Text         *TextContent  `json:"text,omitempty"`
	TemplateCard TemplateCard  `json:"template_card,omitempty"`
	File         *MediaContent `json:"file,omitempty"`
	Image        *MediaContent `json:"image,omitempty"`
	Voice        *MediaContent `json:"voice,omitempty"`
	Video        *MediaContent `json:"video,omitempty"`
	Stream       *Stream       `json:"stream,omitempty"`
}

// Reply sends a callback reply and waits for its acknowledgment. reqID must be
// the callback's headers.req_id. Stream updates also serialize by stream ID.
func (c *Client) Reply(ctx context.Context, reqID string, body MessageBody) (Frame, error) {
	return c.replyBody(ctx, CmdRespond, reqID, body)
}

// ReplyMarkdown replies with Markdown text.
func (c *Client) ReplyMarkdown(ctx context.Context, reqID, content string) (Frame, error) {
	return c.Reply(ctx, reqID, MessageBody{MsgType: "markdown", Markdown: &TextContent{Content: content}})
}

// ReplyTemplateCard replies with an extensible template card.
func (c *Client) ReplyTemplateCard(ctx context.Context, reqID string, card TemplateCard) (Frame, error) {
	return c.Reply(ctx, reqID, MessageBody{MsgType: "template_card", TemplateCard: card})
}

// ReplyMedia replies with an uploaded file, image, voice or video.
func (c *Client) ReplyMedia(ctx context.Context, reqID, mediaType string, media MediaContent) (Frame, error) {
	body := MessageBody{MsgType: mediaType}
	switch mediaType {
	case "file":
		body.File = &media
	case "image":
		body.Image = &media
	case "voice":
		body.Voice = &media
	case "video":
		body.Video = &media
	}
	return c.Reply(ctx, reqID, body)
}

// ReplyStream sends or refreshes a stream using its full current content.
func (c *Client) ReplyStream(ctx context.Context, reqID string, stream Stream) (Frame, error) {
	return c.Reply(ctx, reqID, MessageBody{MsgType: "stream", Stream: &stream})
}

// ReplyStreamWithTemplateCard sends a combined stream and optional card. This
// SDK-compatible payload is currently unsupported by long-connection servers.
func (c *Client) ReplyStreamWithTemplateCard(ctx context.Context, reqID string, stream Stream, card TemplateCard) (Frame, error) {
	return c.Reply(ctx, reqID, MessageBody{MsgType: "stream_with_template_card", Stream: &stream, TemplateCard: card})
}

// UpdateTemplateCard replaces the card for the listed users. An empty userIDs
// list updates the card for every user associated with the original message.
// card.task_id must reuse the task ID of the original card.
func (c *Client) UpdateTemplateCard(ctx context.Context, reqID string, card TemplateCard, userIDs []string) (Frame, error) {
	if reqID == "" || len(card) == 0 {
		return Frame{}, invalidMessage(CmdUpdate, reqID, "req_id and template_card are required")
	}
	if taskID, ok := card["task_id"].(string); !ok || taskID == "" {
		return Frame{}, invalidMessage(CmdUpdate, reqID, "template_card.task_id must match the original card")
	}
	body := struct {
		ResponseType string       `json:"response_type"`
		UserIDs      []string     `json:"userids,omitempty"`
		TemplateCard TemplateCard `json:"template_card"`
	}{ResponseType: "update_template_card", UserIDs: userIDs, TemplateCard: card}
	return c.request(ctx, CmdUpdate, reqID, body)
}

// ReplyWelcome replies to an enter_chat callback with text or a template card.
func (c *Client) ReplyWelcome(ctx context.Context, reqID string, body MessageBody) (Frame, error) {
	return c.replyBody(ctx, CmdWelcome, reqID, body)
}

// Send proactively sends Markdown, a template card or uploaded media. The
// request uses a fresh req_id; the server interprets chatID using body.ChatType.
func (c *Client) Send(ctx context.Context, chatID string, body MessageBody) (Frame, error) {
	if chatID == "" {
		return Frame{}, invalidMessage(CmdSend, "", "chatid is required")
	}
	if err := validateMessage(CmdSend, body); err != nil {
		return Frame{}, invalidMessage(CmdSend, "", err.Error())
	}
	payload := struct {
		ChatID string `json:"chatid"`
		MessageBody
	}{ChatID: chatID, MessageBody: body}
	return c.request(ctx, CmdSend, "", payload)
}

func (c *Client) replyBody(ctx context.Context, command, reqID string, body MessageBody) (Frame, error) {
	if ctx == nil {
		return Frame{}, invalidMessage(command, reqID, "context is required")
	}
	if reqID == "" {
		return Frame{}, invalidMessage(command, reqID, "req_id is required")
	}
	if err := validateMessage(command, body); err != nil {
		return Frame{}, invalidMessage(command, reqID, err.Error())
	}
	bound, bindErr := c.bindSession(ctx)
	if bindErr != nil {
		return Frame{}, operationError(command, reqID, bindErr)
	}
	deadline, cancel := context.WithTimeout(bound, c.options.requestTimeout)
	defer cancel()
	started := false
	request := func() (Frame, error) { started = true; return c.request(deadline, command, reqID, body) }
	if body.Stream != nil {
		frame, err := c.withKey(deadline, "stream:"+body.Stream.ID, request)
		if err != nil && !started {
			return frame, operationError(command, reqID, err)
		}
		return frame, err
	}
	return request()
}

func invalidMessage(operation, reqID, message string) error {
	return operationError(operation, reqID, fmt.Errorf("%w: %s", ErrInvalidArgument, message))
}

func validateMessage(command string, body MessageBody) error {
	if command == CmdWelcome && body.MsgType != "text" && body.MsgType != "template_card" {
		return fmt.Errorf("welcome requires text or template_card")
	}
	if command == CmdSend && (body.MsgType == "text" || body.MsgType == "stream" || body.MsgType == "stream_with_template_card") {
		return fmt.Errorf("unsupported proactive message type")
	}
	if command != CmdWelcome && body.MsgType == "text" {
		return fmt.Errorf("text is only supported for welcome replies")
	}
	if body.ChatType < 0 || body.ChatType > 2 {
		return fmt.Errorf("chat_type must be 0, 1 or 2")
	}
	switch body.MsgType {
	case "text":
		if body.Text == nil || !utf8.ValidString(body.Text.Content) {
			return fmt.Errorf("UTF-8 text is required")
		}
	case "markdown":
		if body.Markdown == nil {
			return fmt.Errorf("markdown is required")
		}
		if err := validateContent(body.Markdown.Content); err != nil {
			return err
		}
		return validateFeedback(body.Markdown.Feedback)
	case "template_card":
		if len(body.TemplateCard) == 0 {
			return fmt.Errorf("template_card is required")
		}
	case "file", "image", "voice", "video":
		var media *MediaContent
		switch body.MsgType {
		case "file":
			media = body.File
		case "image":
			media = body.Image
		case "voice":
			media = body.Voice
		case "video":
			media = body.Video
		}
		if media == nil || media.MediaID == "" {
			return fmt.Errorf("media_id is required")
		}
	case "stream", "stream_with_template_card":
		if body.Stream == nil || body.Stream.ID == "" {
			return fmt.Errorf("stream.id is required")
		}
		if err := validateContent(body.Stream.Content); err != nil {
			return err
		}
		if len(body.Stream.MsgItem) > 10 {
			return fmt.Errorf("stream supports at most 10 images")
		}
		if len(body.Stream.MsgItem) > 0 && !body.Stream.Finish {
			return fmt.Errorf("stream images require finish=true")
		}
		for _, item := range body.Stream.MsgItem {
			if item.MsgType != "image" || item.Image.Base64 == "" || item.Image.MD5 == "" {
				return fmt.Errorf("stream images require image type, base64 and md5")
			}
		}
		return validateFeedback(body.Stream.Feedback)
	default:
		return fmt.Errorf("unsupported message type")
	}
	return nil
}

func validateContent(content string) error {
	if len(content) > 20480 || !utf8.ValidString(content) {
		return fmt.Errorf("content must be UTF-8 and at most 20480 bytes")
	}
	return nil
}

func validateFeedback(feedback *Feedback) error {
	if feedback != nil && (len(feedback.ID) > 256 || !utf8.ValidString(feedback.ID)) {
		return fmt.Errorf("feedback.id must be UTF-8 and at most 256 bytes")
	}
	return nil
}
