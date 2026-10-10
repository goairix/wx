package documents

import (
	"context"

	"github.com/goairix/wx/v2/work/aibot"
)

type SheetProperties struct {
	Title string `json:"title,omitempty"`
}
type AddSheetRequest struct {
	DocID      string           `json:"docid"`
	Properties *SheetProperties `json:"properties,omitempty"`
}

func (c *Client) AddSheet(ctx context.Context, r AddSheetRequest) (*ToolResult, error) {
	if r.DocID == "" {
		return nil, aibot.WrapError("documents.smartsheet_add_sheet", "", aibot.ErrInvalidArgument)
	}
	return c.CallTool(ctx, "smartsheet_add_sheet", r)
}
func (c *Client) GetSheets(ctx context.Context, docID string) (*ToolResult, error) {
	if docID == "" {
		return nil, aibot.WrapError("documents.smartsheet_get_sheet", "", aibot.ErrInvalidArgument)
	}
	return c.CallTool(ctx, "smartsheet_get_sheet", struct {
		DocID string `json:"docid"`
	}{docID})
}
