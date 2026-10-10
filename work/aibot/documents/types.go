// Package documents implements the intelligent-bot document MCP tools.
package documents

import (
	"context"
	"encoding/json"
	"errors"
)

type Caller interface {
	Tools(context.Context) ([]Tool, error)
	CallTool(context.Context, string, any) (*ToolResult, error)
}
type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"inputSchema"`
	Raw         json.RawMessage `json:"-"`
}

func (t *Tool) UnmarshalJSON(b []byte) error {
	type alias Tool
	var v alias
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	*t = Tool(v)
	t.Raw = append(json.RawMessage(nil), b...)
	return nil
}

type ContentBlock = json.RawMessage
type ToolResult struct {
	Content           []ContentBlock  `json:"content"`
	StructuredContent json.RawMessage `json:"structuredContent,omitempty"`
	IsError           bool            `json:"isError,omitempty"`
	Raw               json.RawMessage `json:"-"`
	RequestID         string          `json:"-"`
	HTTPStatus        int             `json:"-"`
}
type ToolError struct {
	Tool   string
	Result *ToolResult
}

func (e *ToolError) Error() string { return "MCP tool reported failure" }

type RPCError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

func (e *RPCError) Error() string { return "MCP protocol error" }

var (
	ErrSessionExpired  = errors.New("MCP session expired")
	ErrDeliveryUnknown = errors.New("MCP delivery status unknown")
)
