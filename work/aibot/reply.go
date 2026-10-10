package aibot

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"unicode/utf8"
)

type Feedback struct {
	ID string `json:"id"`
}
type MediaContent struct {
	MediaID     string `json:"media_id"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
}
type StreamImage struct {
	Base64 string `json:"base64"`
	MD5    string `json:"md5"`
}
type StreamItem struct {
	MsgType string      `json:"msgtype"`
	Image   StreamImage `json:"image"`
}
type Stream struct {
	ID       string       `json:"id"`
	Finish   bool         `json:"finish"`
	Content  string       `json:"content"`
	MsgItem  []StreamItem `json:"msg_item,omitempty"`
	Feedback *Feedback    `json:"feedback,omitempty"`
}
type OutgoingMessage struct {
	MsgType      string        `json:"msgtype"`
	ChatType     int           `json:"chat_type,omitempty"`
	Markdown     *TextContent  `json:"markdown,omitempty"`
	Text         *TextContent  `json:"text,omitempty"`
	TemplateCard CardPayload   `json:"template_card,omitempty"`
	File         *MediaContent `json:"file,omitempty"`
	Image        *MediaContent `json:"image,omitempty"`
	Voice        *MediaContent `json:"voice,omitempty"`
	Video        *MediaContent `json:"video,omitempty"`
	Stream       *Stream       `json:"stream,omitempty"`
}
type Reply struct {
	OutgoingMessage
	UserIDs []string `json:"userids,omitempty"`
}

func ValidateMessage(m OutgoingMessage) error {
	if m.ChatType < 0 || m.ChatType > 2 {
		return ErrInvalidArgument
	}
	present := 0
	for _, v := range []bool{m.Markdown != nil, m.Text != nil, m.TemplateCard != nil, m.File != nil, m.Image != nil, m.Voice != nil, m.Video != nil, m.Stream != nil} {
		if v {
			present++
		}
	}
	want := 1
	if m.MsgType == "stream_with_template_card" && m.TemplateCard != nil {
		want = 2
	}
	if present != want {
		return fmt.Errorf("%w: payload must match msgtype", ErrInvalidArgument)
	}
	switch m.MsgType {
	case "markdown":
		if m.Markdown == nil {
			return ErrInvalidArgument
		}
		return validateText(m.Markdown)
	case "text":
		if m.Text == nil {
			return ErrInvalidArgument
		}
		return validateText(m.Text)
	case "template_card":
		return ValidateCard(m.TemplateCard)
	case "file", "image", "voice", "video":
		media := map[string]*MediaContent{"file": m.File, "image": m.Image, "voice": m.Voice, "video": m.Video}[m.MsgType]
		if media == nil || media.MediaID == "" {
			return ErrInvalidArgument
		}
	case "stream", "stream_with_template_card":
		s := m.Stream
		if s == nil || s.ID == "" || len(s.MsgItem) > 10 || (!s.Finish && len(s.MsgItem) > 0) {
			return ErrInvalidArgument
		}
		if err := validateText(&TextContent{Content: s.Content, Feedback: s.Feedback}); err != nil {
			return err
		}
		for _, item := range s.MsgItem {
			if item.MsgType != "image" || len(item.Image.Base64) > base64.StdEncoding.EncodedLen(10<<20) {
				return ErrInvalidArgument
			}
			b, err := base64.StdEncoding.DecodeString(item.Image.Base64)
			if err != nil || len(b) > 10<<20 {
				return ErrInvalidArgument
			}
			sum := md5.Sum(b)
			if hex.EncodeToString(sum[:]) != item.Image.MD5 || !isImage(b) {
				return ErrInvalidArgument
			}
		}
		if m.TemplateCard != nil {
			return ValidateCard(m.TemplateCard)
		}
	default:
		return ErrUnsupported
	}
	return nil
}
func validateText(t *TextContent) error {
	if !utf8.ValidString(t.Content) || len(t.Content) > 20480 {
		return ErrInvalidArgument
	}
	if t.Feedback != nil && (!utf8.ValidString(t.Feedback.ID) || len(t.Feedback.ID) > 256) {
		return ErrInvalidArgument
	}
	return nil
}
func CloneCardWithTask(card CardPayload, task string) (RawTemplateCard, error) {
	b, err := json.Marshal(card)
	if err != nil {
		return nil, err
	}
	var data map[string]json.RawMessage
	if err = json.Unmarshal(b, &data); err != nil || data == nil {
		return nil, ErrInvalidArgument
	}
	data["task_id"], err = json.Marshal(task)
	if err != nil {
		return nil, err
	}
	b, err = json.Marshal(data)
	return RawTemplateCard(b), err
}
