package aibot

import (
	"encoding/json"
	"unicode/utf8"
)

func ValidateCard(c CardPayload) error {
	if c == nil {
		return ErrInvalidArgument
	}
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}
	var card struct {
		Feedback *Feedback    `json:"feedback,omitempty"`
		CardType string       `json:"card_type"`
		TaskID   string       `json:"task_id"`
		JumpList []JumpAction `json:"jump_list"`
	}
	if err = json.Unmarshal(b, &card); err != nil || card.CardType == "" {
		return ErrInvalidArgument
	}
	if card.Feedback != nil {
		if err := validateText(&TextContent{Feedback: card.Feedback}); err != nil {
			return err
		}
	}
	if len(card.TaskID) > 128 {
		return ErrInvalidArgument
	}
	for _, j := range card.JumpList {
		if j.Type == 3 && (j.Question == "" || len(j.Question) > 200 || !utf8.ValidString(j.Question)) {
			return ErrInvalidArgument
		}
	}
	return nil
}
func isImage(b []byte) bool {
	return (len(b) >= 8 && string(b[:8]) == "\x89PNG\r\n\x1a\n") || (len(b) >= 3 && b[0] == 0xff && b[1] == 0xd8 && b[2] == 0xff)
}
