package aibot

// MessageFrom identifies the user who sent a message or triggered an event.
type MessageFrom struct {
	UserID string `json:"userid"`
	CorpID string `json:"corpid,omitempty"`
}

// TextContent is plain text or Markdown. Feedback is supported on outgoing Markdown.
type TextContent struct {
	Content  string    `json:"content"`
	Feedback *Feedback `json:"feedback,omitempty"`
}

// DownloadContent carries a short-lived encrypted media URL and its AES key.
type DownloadContent struct {
	URL    string `json:"url"`
	AESKey string `json:"aeskey,omitempty"`
}

// VoiceContent contains the speech-to-text result of an incoming voice message.
type VoiceContent struct {
	Content string `json:"content"`
}

// MixedItem is one text or image item in an incoming mixed message.
type MixedItem struct {
	MsgType string           `json:"msgtype"`
	Text    *TextContent     `json:"text,omitempty"`
	Image   *DownloadContent `json:"image,omitempty"`
}

// MixedContent preserves the order of incoming text and image items.
type MixedContent struct {
	MsgItem []MixedItem `json:"msg_item"`
}

// QuoteContent describes a message quoted by the sender.
type QuoteContent struct {
	MsgType string           `json:"msgtype"`
	Text    *TextContent     `json:"text,omitempty"`
	Image   *DownloadContent `json:"image,omitempty"`
	Mixed   *MixedContent    `json:"mixed,omitempty"`
	Voice   *VoiceContent    `json:"voice,omitempty"`
	File    *DownloadContent `json:"file,omitempty"`
	Video   *DownloadContent `json:"video,omitempty"`
}

// Message is a typed message callback body. Frame.Raw retains unknown fields.
type Message struct {
	MsgID       string           `json:"msgid"`
	AIBotID     string           `json:"aibotid"`
	ChatID      string           `json:"chatid,omitempty"`
	ChatType    string           `json:"chattype"`
	From        MessageFrom      `json:"from"`
	CreateTime  int64            `json:"create_time,omitempty"`
	ResponseURL string           `json:"response_url,omitempty"`
	MsgType     string           `json:"msgtype"`
	Stream      *StreamRefresh   `json:"stream,omitempty"`
	Text        *TextContent     `json:"text,omitempty"`
	Image       *DownloadContent `json:"image,omitempty"`
	Voice       *VoiceContent    `json:"voice,omitempty"`
	File        *DownloadContent `json:"file,omitempty"`
	Video       *DownloadContent `json:"video,omitempty"`
	Mixed       *MixedContent    `json:"mixed,omitempty"`
	Quote       *QuoteContent    `json:"quote,omitempty"`
}

// EventContent covers enter_chat, template_card_event, feedback_event and
// disconnected_event. Other event fields remain available through Frame.Raw.
type EventContent struct {
	EventType         string             `json:"eventtype"`
	TemplateCardEvent *TemplateCardEvent `json:"template_card_event,omitempty"`
	FeedbackEvent     *FeedbackEvent     `json:"feedback_event,omitempty"`
	// Flat fields are retained for compatibility with earlier SDK event models.
	EventKey string `json:"event_key,omitempty"`
	TaskID   string `json:"task_id,omitempty"`
}

// TemplateCardEvent contains the submitted card action and selected answers.
type TemplateCardEvent struct {
	CardType      string         `json:"card_type"`
	EventKey      string         `json:"event_key"`
	TaskID        string         `json:"task_id"`
	SelectedItems *SelectedItems `json:"selected_items,omitempty"`
}

// SelectedItems wraps submitted card questions in the callback wire format.
type SelectedItems struct {
	SelectedItem []SelectedItem `json:"selected_item"`
}

// SelectedItem associates one card question with the selected option IDs.
type SelectedItem struct {
	QuestionKey string    `json:"question_key"`
	OptionIDs   OptionIDs `json:"option_ids"`
}

// OptionIDs preserves all selected answers for single and multiple choice.
type OptionIDs struct {
	OptionID []string `json:"option_id"`
}

// FeedbackEvent contains the user's rating and optional explanation.
type FeedbackEvent struct {
	ID                   string `json:"id"`
	Type                 int    `json:"type"`
	Content              string `json:"content,omitempty"`
	InaccurateReasonList []int  `json:"inaccurate_reason_list,omitempty"`
}

// EventMessage is a typed event callback body.
type EventMessage struct {
	MsgID       string       `json:"msgid"`
	AIBotID     string       `json:"aibotid"`
	ChatID      string       `json:"chatid,omitempty"`
	ChatType    string       `json:"chattype,omitempty"`
	From        MessageFrom  `json:"from"`
	CreateTime  int64        `json:"create_time"`
	ResponseURL string       `json:"response_url,omitempty"`
	MsgType     string       `json:"msgtype"`
	Event       EventContent `json:"event"`
}
