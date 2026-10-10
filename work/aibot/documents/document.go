package documents

import (
	"context"

	"github.com/goairix/wx/v2/work/aibot"
)

type CreateDocumentRequest struct {
	DocType int    `json:"doc_type"`
	DocName string `json:"doc_name"`
}
type EditDocumentContentRequest struct {
	DocID       string `json:"docid"`
	Content     string `json:"content"`
	ContentType int    `json:"content_type"`
}

func (c *Client) CreateDocument(ctx context.Context, r CreateDocumentRequest) (*ToolResult, error) {
	if (r.DocType != 3 && r.DocType != 10) || r.DocName == "" {
		return nil, aibot.WrapError("documents.create_doc", "", aibot.ErrInvalidArgument)
	}
	return c.CallTool(ctx, "create_doc", r)
}
func (c *Client) EditDocumentContent(ctx context.Context, r EditDocumentContentRequest) (*ToolResult, error) {
	if r.DocID == "" || r.ContentType != 1 {
		return nil, aibot.WrapError("documents.edit_doc_content", "", aibot.ErrInvalidArgument)
	}
	return c.CallTool(ctx, "edit_doc_content", r)
}
