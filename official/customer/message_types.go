package customer

// MessageType identifies a customer service message payload.
type MessageType string

const (
	MessageText            MessageType = "text"
	MessageImage           MessageType = "image"
	MessageVoice           MessageType = "voice"
	MessageVideo           MessageType = "video"
	MessageMusic           MessageType = "music"
	MessageNews            MessageType = "news"
	MessageMPNews          MessageType = "mpnews"
	MessageMPNewsArticle   MessageType = "mpnewsarticle"
	MessageMenu            MessageType = "msgmenu"
	MessageWXCard          MessageType = "wxcard"
	MessageMiniProgramPage MessageType = "miniprogrampage"
	MessageLink            MessageType = "link"
)

// Message contains the payload selected by MsgType and optional sender context.
type Message struct {
	ToUser          string           `json:"touser"`
	MsgType         MessageType      `json:"msgtype"`
	Text            *Text            `json:"text,omitempty"`
	Image           *Media           `json:"image,omitempty"`
	Voice           *Media           `json:"voice,omitempty"`
	Video           *Video           `json:"video,omitempty"`
	Music           *Music           `json:"music,omitempty"`
	News            *News            `json:"news,omitempty"`
	MPNews          *Media           `json:"mpnews,omitempty"`
	MPNewsArticle   *MPNewsArticle   `json:"mpnewsarticle,omitempty"`
	Menu            *Menu            `json:"msgmenu,omitempty"`
	WXCard          *WXCard          `json:"wxcard,omitempty"`
	MiniProgramPage *MiniProgramPage `json:"miniprogrampage,omitempty"`
	Link            *Link            `json:"link,omitempty"`
	CustomService   *CustomService   `json:"customservice,omitempty"`
	AIContext       *AIContext       `json:"aimsgcontext,omitempty"`
	BusinessID      string           `json:"businessid,omitempty"`
}

// Text is a text message, including supported HTML mini program links.
type Text struct {
	Content string `json:"content"`
}

// Media references an uploaded image, voice, or legacy news material.
type Media struct {
	MediaID string `json:"media_id"`
}

// Video references a video and its thumbnail.
type Video struct {
	MediaID      string `json:"media_id"`
	ThumbMediaID string `json:"thumb_media_id"`
	Title        string `json:"title,omitempty"`
	Description  string `json:"description,omitempty"`
}

// Music contains music links and a thumbnail.
type Music struct {
	Title        string `json:"title"`
	Description  string `json:"description"`
	MusicURL     string `json:"musicurl"`
	HQMusicURL   string `json:"hqmusicurl,omitempty"`
	ThumbMediaID string `json:"thumb_media_id"`
}

// News contains external articles; WeChat currently allows at most one.
type News struct {
	Articles []NewsArticle `json:"articles"`
}

// NewsArticle is an external article card.
type NewsArticle struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	PicURL      string `json:"picurl"`
	URL         string `json:"url"`
}

// MPNewsArticle references an article ID from the publishing APIs.
type MPNewsArticle struct {
	ArticleID string `json:"article_id"`
}

// Menu contains selectable text menu entries.
type Menu struct {
	HeadContent string     `json:"head_content,omitempty"`
	List        []MenuItem `json:"list"`
	TailContent string     `json:"tail_content,omitempty"`
}

// MenuItem is one selectable menu entry.
type MenuItem struct {
	ID      string `json:"id"`
	Content string `json:"content"`
}

// WXCard references a WeChat card.
type WXCard struct {
	CardID string `json:"card_id"`
}

// MiniProgramPage is a mini program page card.
type MiniProgramPage struct {
	Title        string `json:"title"`
	AppID        string `json:"appid"`
	PagePath     string `json:"pagepath"`
	ThumbMediaID string `json:"thumb_media_id"`
}

// Link is a link card.
type Link struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	URL         string `json:"url"`
	ThumbURL    string `json:"thumb_url"`
}

// CustomService selects the sending customer service account.
type CustomService struct {
	Account string `json:"kf_account"`
}

// AIContext controls the third-party AI attribution below a message.
type AIContext struct {
	IsAIMessage int `json:"is_ai_msg"`
}

// TypingCommand controls the user-visible typing indicator.
type TypingCommand string

const (
	Typing       TypingCommand = "Typing"
	CancelTyping TypingCommand = "CancelTyping"
)

// TypingRequest changes the typing indicator for a user.
type TypingRequest struct {
	ToUser     string        `json:"touser"`
	Command    TypingCommand `json:"command"`
	BusinessID string        `json:"businessid,omitempty"`
}
