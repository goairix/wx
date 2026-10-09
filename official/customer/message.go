package customer

import (
	"context"

	wxerrors "github.com/goairix/wx/v2/core/errors"
)

func inputError(operation, message string) error {
	return &wxerrors.Error{Platform: "official", Operation: operation, Message: message}
}

// Send sends a typed customer service message.
func (c *Client) Send(ctx context.Context, input Message) error {
	const operation = "official.customer.message.send"
	if input.ToUser == "" {
		return inputError(operation, "touser is required")
	}
	valid := false
	switch input.MsgType {
	case MessageText:
		valid = input.Text != nil
	case MessageImage:
		valid = input.Image != nil
	case MessageVoice:
		valid = input.Voice != nil
	case MessageVideo:
		valid = input.Video != nil
	case MessageMusic:
		valid = input.Music != nil
	case MessageNews:
		valid = input.News != nil
	case MessageMPNews:
		valid = input.MPNews != nil
	case MessageMPNewsArticle:
		valid = input.MPNewsArticle != nil
	case MessageMenu:
		valid = input.Menu != nil
	case MessageWXCard:
		valid = input.WXCard != nil
	case MessageMiniProgramPage:
		valid = input.MiniProgramPage != nil
	case MessageLink:
		valid = input.Link != nil
	}
	if !valid {
		return inputError(operation, "msgtype must select a supported nonnil payload")
	}
	return c.api.Post(ctx, operation, "cgi-bin/message/custom/send", input, nil)
}

// SetTyping changes the user-visible typing indicator.
func (c *Client) SetTyping(ctx context.Context, input TypingRequest) error {
	const operation = "official.customer.message.typing"
	if input.ToUser == "" || input.Command != Typing && input.Command != CancelTyping {
		return inputError(operation, "touser and a valid typing command are required")
	}
	return c.api.Post(ctx, operation, "cgi-bin/message/custom/typing", input, nil)
}

// Records returns one page of chat records within a window of at most 24 hours.
func (c *Client) Records(ctx context.Context, input RecordRequest) (*RecordList, error) {
	const operation = "official.customer.message.records"
	if input.StartTime < 0 || input.EndTime < input.StartTime || input.EndTime-input.StartTime > 86400 || input.MessageID < 0 || input.Number < 1 || input.Number > 10000 {
		return nil, inputError(operation, "records require a window within 24 hours, a nonnegative cursor and number between 1 and 10000")
	}
	result := new(RecordList)
	err := c.api.Post(ctx, operation, "customservice/msgrecord/getmsglist", input, result)
	return result, err
}
