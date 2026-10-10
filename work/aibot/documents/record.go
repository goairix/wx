package documents

import (
	"context"

	"github.com/goairix/wx/v2/work/aibot"
)

// Record.Values keys are field titles, not field IDs. Numeric zero and false are retained.
type Record struct {
	Values map[string]any `json:"values"`
}
type AddRecordsRequest struct {
	DocID   string   `json:"docid"`
	SheetID string   `json:"sheet_id"`
	Records []Record `json:"records"`
}
type TextValue struct {
	Type string `json:"type"`
	Text string `json:"text"`
	Link string `json:"link,omitempty"`
}
type UserValue struct {
	UserID string `json:"user_id"`
}
type URLValue struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
	Link string `json:"link"`
}
type ImageValue struct {
	ID       string `json:"id,omitempty"`
	Title    string `json:"title,omitempty"`
	ImageURL string `json:"image_url,omitempty"`
	Width    *int   `json:"width,omitempty"`
	Height   *int   `json:"height,omitempty"`
}
type OptionValue struct {
	ID    string `json:"id,omitempty"`
	Style *int   `json:"style,omitempty"`
	Text  string `json:"text,omitempty"`
}
type LocationValue struct {
	SourceType int    `json:"source_type"`
	ID         string `json:"id"`
	Latitude   string `json:"latitude"`
	Longitude  string `json:"longitude"`
	Title      string `json:"title"`
}

func (c *Client) AddRecords(ctx context.Context, r AddRecordsRequest) (*ToolResult, error) {
	if r.DocID == "" || r.SheetID == "" || len(r.Records) == 0 {
		return nil, aibot.WrapError("documents.smartsheet_add_records", "", aibot.ErrInvalidArgument)
	}
	for _, record := range r.Records {
		if len(record.Values) == 0 {
			return nil, aibot.WrapError("documents.smartsheet_add_records", "", aibot.ErrInvalidArgument)
		}
		for title := range record.Values {
			if title == "" {
				return nil, aibot.WrapError("documents.smartsheet_add_records", "", aibot.ErrInvalidArgument)
			}
		}
	}
	return c.CallTool(ctx, "smartsheet_add_records", r)
}
