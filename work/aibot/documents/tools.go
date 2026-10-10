package documents

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/goairix/wx/v2/work/aibot"
)

func (t *transport) Tools(ctx context.Context) ([]Tool, error) {
	t.mu.Lock()
	supported := t.tools
	t.mu.Unlock()
	if !supported {
		return nil, aibot.ErrUnsupported
	}
	var tools []Tool
	cursor := ""
	seen := map[string]bool{}
	var total int64
	for {
		params := map[string]any{}
		if cursor != "" {
			params["cursor"] = cursor
		}
		raw, meta, err := t.call(ctx, "tools/list", params)
		if err != nil {
			return tools, err
		}
		total += int64(len(raw))
		if total > t.shared.MaxHTTPResponseBytes {
			return tools, t.rpcError("tools/list", meta, aibot.ErrBodyTooLarge)
		}
		var result struct {
			Tools      []Tool `json:"tools"`
			NextCursor string `json:"nextCursor"`
		}
		if err = json.Unmarshal(raw, &result); err != nil || result.Tools == nil {
			return tools, t.rpcError("tools/list", meta, errors.Join(aibot.ErrProtocol, err))
		}
		tools = append(tools, result.Tools...)
		if result.NextCursor == "" {
			return tools, nil
		}
		if seen[result.NextCursor] {
			return tools, t.rpcError("tools/list", meta, aibot.ErrProtocol)
		}
		seen[result.NextCursor] = true
		cursor = result.NextCursor
	}
}
func (t *transport) CallTool(ctx context.Context, name string, args any) (*ToolResult, error) {
	t.mu.Lock()
	supported := t.tools
	t.mu.Unlock()
	if !supported {
		return nil, aibot.ErrUnsupported
	}
	if args == nil {
		args = map[string]any{}
	}
	raw, meta, err := t.call(ctx, "tools/call", map[string]any{"name": name, "arguments": args})
	if raw == nil {
		return nil, err
	}
	result := &ToolResult{Raw: raw, RequestID: meta.RequestID, HTTPStatus: meta.HTTPStatus}
	decodeErr := json.Unmarshal(raw, result)
	if decodeErr != nil || result.Content == nil {
		decodeErr = errors.Join(aibot.ErrProtocol, decodeErr)
	}
	return result, errors.Join(err, t.rpcErrorIf("tools/call", meta, decodeErr))
}
func (t *transport) rpcErrorIf(method string, meta ResponseMeta, err error) error {
	if err == nil {
		return nil
	}
	return t.rpcError(method, meta, err)
}
