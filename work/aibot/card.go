package aibot

import "encoding/json"

// CardPayload preserves the typed card or the complete raw platform object.
type CardPayload interface {
	json.Marshaler
	cardPayload()
}
type RawTemplateCard json.RawMessage

func (c RawTemplateCard) MarshalJSON() ([]byte, error) {
	if !json.Valid(c) {
		return nil, ErrInvalidArgument
	}
	return c, nil
}

type TemplateCard struct {
	Feedback              *Feedback           `json:"feedback,omitempty"`
	CardType              string              `json:"card_type"`
	Source                *Source             `json:"source,omitempty"`
	ActionMenu            *ActionMenu         `json:"action_menu,omitempty"`
	MainTitle             *MainTitle          `json:"main_title,omitempty"`
	EmphasisContent       *EmphasisContent    `json:"emphasis_content,omitempty"`
	QuoteArea             *QuoteArea          `json:"quote_area,omitempty"`
	SubTitleText          string              `json:"sub_title_text,omitempty"`
	HorizontalContentList []HorizontalContent `json:"horizontal_content_list,omitempty"`
	JumpList              []JumpAction        `json:"jump_list,omitempty"`
	CardAction            *CardAction         `json:"card_action,omitempty"`
	VerticalContentList   []VerticalContent   `json:"vertical_content_list,omitempty"`
	CardImage             *CardImage          `json:"card_image,omitempty"`
	ImageTextArea         *ImageTextArea      `json:"image_text_area,omitempty"`
	ButtonSelection       *SelectionItem      `json:"button_selection,omitempty"`
	ButtonList            []Button            `json:"button_list,omitempty"`
	Checkbox              *Checkbox           `json:"checkbox,omitempty"`
	SelectList            []SelectionItem     `json:"select_list,omitempty"`
	SubmitButton          *SubmitButton       `json:"submit_button,omitempty"`
	TaskID                string              `json:"task_id,omitempty"`
}

func (c *TemplateCard) MarshalJSON() ([]byte, error) {
	type plain TemplateCard
	return json.Marshal((*plain)(c))
}

type Source struct {
	IconURL   string `json:"icon_url,omitempty"`
	Desc      string `json:"desc,omitempty"`
	DescColor int    `json:"desc_color,omitempty"`
}
type ActionMenu struct {
	Desc       string   `json:"desc"`
	ActionList []Action `json:"action_list"`
}
type Action struct {
	Text string `json:"text"`
	Key  string `json:"key"`
}
type MainTitle struct {
	Title string `json:"title,omitempty"`
	Desc  string `json:"desc,omitempty"`
}
type EmphasisContent struct {
	Title string `json:"title,omitempty"`
	Desc  string `json:"desc,omitempty"`
}
type QuoteArea struct {
	Type      int    `json:"type,omitempty"`
	URL       string `json:"url,omitempty"`
	AppID     string `json:"appid,omitempty"`
	PagePath  string `json:"pagepath,omitempty"`
	Title     string `json:"title,omitempty"`
	QuoteText string `json:"quote_text,omitempty"`
}
type HorizontalContent struct {
	Type    int    `json:"type,omitempty"`
	KeyName string `json:"keyname"`
	Value   string `json:"value,omitempty"`
	URL     string `json:"url,omitempty"`
	UserID  string `json:"userid,omitempty"`
}
type JumpAction struct {
	Type     int    `json:"type,omitempty"`
	Question string `json:"question,omitempty"`
	Title    string `json:"title"`
	URL      string `json:"url,omitempty"`
	AppID    string `json:"appid,omitempty"`
	PagePath string `json:"pagepath,omitempty"`
}
type CardAction struct {
	Type     int    `json:"type"`
	URL      string `json:"url,omitempty"`
	AppID    string `json:"appid,omitempty"`
	PagePath string `json:"pagepath,omitempty"`
}
type VerticalContent struct {
	Title string `json:"title"`
	Desc  string `json:"desc,omitempty"`
}
type CardImage struct {
	URL         string  `json:"url"`
	AspectRatio float64 `json:"aspect_ratio,omitempty"`
}
type ImageTextArea struct {
	Type     int    `json:"type,omitempty"`
	URL      string `json:"url,omitempty"`
	AppID    string `json:"appid,omitempty"`
	PagePath string `json:"pagepath,omitempty"`
	Title    string `json:"title,omitempty"`
	Desc     string `json:"desc,omitempty"`
	ImageURL string `json:"image_url"`
}
type SubmitButton struct {
	Text string `json:"text"`
	Key  string `json:"key"`
}
type SelectionItem struct {
	QuestionKey string            `json:"question_key"`
	Title       string            `json:"title,omitempty"`
	Disable     *bool             `json:"disable,omitempty"`
	SelectedID  string            `json:"selected_id,omitempty"`
	OptionList  []SelectionOption `json:"option_list"`
}
type SelectionOption struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}
type Button struct {
	Text  string `json:"text"`
	Style int    `json:"style,omitempty"`
	Key   string `json:"key"`
}
type Checkbox struct {
	QuestionKey string           `json:"question_key"`
	Disable     *bool            `json:"disable,omitempty"`
	Mode        int              `json:"mode,omitempty"`
	OptionList  []CheckboxOption `json:"option_list"`
}
type CheckboxOption struct {
	ID        string `json:"id"`
	Text      string `json:"text"`
	IsChecked *bool  `json:"is_checked,omitempty"`
}

func (*TemplateCard) cardPayload()   {}
func (RawTemplateCard) cardPayload() {}
