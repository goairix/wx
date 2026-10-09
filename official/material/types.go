package material

// Type identifies an uploaded material or a material list category.
type Type string

const (
	TypeImage Type = "image"
	TypeVoice Type = "voice"
	TypeVideo Type = "video"
	TypeThumb Type = "thumb"
	TypeNews  Type = "news"
)

// UploadResult describes a temporary upload. Thumbnail uploads may return
// thumb_media_id instead of media_id.
type UploadResult struct {
	Type         Type   `json:"type"`
	MediaID      string `json:"media_id"`
	ThumbMediaID string `json:"thumb_media_id"`
	CreatedAt    int64  `json:"created_at"`
}

// VideoDescription is the multipart description field required for videos.
type VideoDescription struct {
	Title        string `json:"title"`
	Introduction string `json:"introduction"`
}

// UploadRequest adds a permanent media file.
type UploadRequest struct {
	Type        Type
	Filename    string
	Data        []byte
	Description *VideoDescription
}

// AddResult contains the permanent ID and the image URL when available.
type AddResult struct {
	MediaID string `json:"media_id"`
	URL     string `json:"url"`
}

// ImageResult contains the URL of an image uploaded for article content.
type ImageResult struct {
	URL string `json:"url"`
}

// DownloadResult holds a temporary file or a video's download URL.
// A video URL is returned as metadata; the SDK does not fetch it automatically.
type DownloadResult struct {
	VideoURL    string `json:"video_url,omitempty"`
	Data        []byte `json:"-"`
	ContentType string `json:"-"`
}

// NewsItem describes one article in a permanent news material.
type NewsItem struct {
	Title            string `json:"title"`
	ThumbMediaID     string `json:"thumb_media_id"`
	ShowCoverPic     int    `json:"show_cover_pic"`
	Author           string `json:"author"`
	Digest           string `json:"digest"`
	Content          string `json:"content"`
	URL              string `json:"url"`
	ContentSourceURL string `json:"content_source_url"`
	ThumbURL         string `json:"thumb_url,omitempty"`
}

// GetResult holds news articles, video metadata, or binary file bytes.
// Binary responses set Data; JSON responses populate the corresponding fields.
type GetResult struct {
	NewsItem    []NewsItem `json:"news_item,omitempty"`
	Title       string     `json:"title,omitempty"`
	Description string     `json:"description,omitempty"`
	DownURL     string     `json:"down_url,omitempty"`
	Data        []byte     `json:"-"`
	ContentType string     `json:"-"`
}

// Counts contains the stored permanent material totals by category.
type Counts struct {
	VoiceCount int `json:"voice_count"`
	VideoCount int `json:"video_count"`
	ImageCount int `json:"image_count"`
	NewsCount  int `json:"news_count"`
}

// ListRequest selects a category, offset and page size between 1 and 20.
type ListRequest struct {
	Type   Type `json:"type"`
	Offset int  `json:"offset"`
	Count  int  `json:"count"`
}

// NewsContent groups the articles in one news material.
type NewsContent struct {
	NewsItem []NewsItem `json:"news_item"`
}

// Item describes a permanent file or a news material in a list.
type Item struct {
	MediaID    string       `json:"media_id"`
	Name       string       `json:"name,omitempty"`
	URL        string       `json:"url,omitempty"`
	UpdateTime int64        `json:"update_time"`
	Content    *NewsContent `json:"content,omitempty"`
}

// ListResult contains one page and its category's total count.
type ListResult struct {
	TotalCount int    `json:"total_count"`
	ItemCount  int    `json:"item_count"`
	Items      []Item `json:"item"`
}
