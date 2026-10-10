package documents

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"net/http/httptrace"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	coreerrors "github.com/goairix/wx/v2/core/errors"
	"github.com/goairix/wx/v2/work/aibot"
	"github.com/goairix/wx/v2/work/aibot/internal/settings"
)

type transport struct {
	config             Config
	shared             settings.Settings
	life               context.Context
	cancel             context.CancelCauseFunc
	mu                 sync.Mutex
	sessionID, version string
	tools              bool
	deleteOnce         sync.Once
	deleteErr          error
}
type ResponseMeta struct {
	RequestID  string
	HTTPStatus int
	batched    bool
}

var rpcCounter atomic.Uint64
var errResult = errors.New("MCP result received")

func (t *transport) headers() http.Header {
	t.mu.Lock()
	sid, version := t.sessionID, t.version
	t.mu.Unlock()
	h := t.config.Header.Clone()
	if h == nil {
		h = make(http.Header)
	}
	h.Set("Content-Type", "application/json")
	h.Set("Accept", "application/json, text/event-stream")
	h.Del("Mcp-Session-Id")
	h.Del("MCP-Protocol-Version")
	if sid != "" {
		h.Set("Mcp-Session-Id", sid)
	}
	if version != "" {
		h.Set("MCP-Protocol-Version", version)
	}
	return h
}
func (t *transport) post(ctx context.Context, b []byte, sent *atomic.Bool) (*http.Response, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, t.config.URL, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	request.GetBody = nil
	request.Header = t.headers()
	if sent != nil {
		trace := &httptrace.ClientTrace{WroteRequest: func(info httptrace.WroteRequestInfo) {
			if info.Err == nil {
				sent.Store(true)
			}
		}}
		request = request.WithContext(httptrace.WithClientTrace(request.Context(), trace))
	}
	client := *t.shared.HTTPClient
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return client.Do(request)
}
func (t *transport) rpcError(method string, meta ResponseMeta, cause error) error {
	e := &coreerrors.Error{Platform: "work", Operation: "work.aibot.documents." + method, RequestID: meta.RequestID, HTTPStatus: meta.HTTPStatus, Message: "document MCP operation failed", Err: cause}
	var nested *coreerrors.Error
	if errors.As(cause, &nested) && nested.HTTPStatus != 0 {
		e.HTTPStatus = nested.HTTPStatus
	}
	var rpc *RPCError
	if errors.As(cause, &rpc) {
		e.Code = strconv.Itoa(rpc.Code)
	}
	return e
}

// Session expiry applies to every POST, including notifications and replies to
// server requests. Callers decide whether other HTTP statuses are acceptable.
func (t *transport) checkResponseStatus(status int, accepted bool) error {
	if status == http.StatusNotFound {
		t.mu.Lock()
		hasSession := t.sessionID != ""
		t.mu.Unlock()
		if hasSession {
			t.cancel(ErrSessionExpired)
			return ErrSessionExpired
		}
	}
	if !accepted {
		return aibot.ErrProtocol
	}
	return nil
}
func (t *transport) initialize(ctx context.Context) error {
	raw, meta, err := t.call(ctx, "initialize", map[string]any{"protocolVersion": "2025-06-18", "capabilities": map[string]any{}, "clientInfo": map[string]string{"name": "goairix-wx", "version": "2"}})
	if err != nil {
		return err
	}
	var result struct {
		ProtocolVersion string `json:"protocolVersion"`
		Capabilities    struct {
			Tools json.RawMessage `json:"tools"`
		} `json:"capabilities"`
	}
	if err = json.Unmarshal(raw, &result); err != nil {
		return t.rpcError("initialize", meta, errors.Join(aibot.ErrProtocol, err))
	}
	if (result.ProtocolVersion != "2025-06-18" && result.ProtocolVersion != "2025-03-26") || (meta.batched && result.ProtocolVersion != "2025-03-26") {
		return t.rpcError("initialize", meta, aibot.ErrUnsupported)
	}
	t.mu.Lock()
	t.version = result.ProtocolVersion
	t.tools = len(result.Capabilities.Tools) > 0 && string(result.Capabilities.Tools) != "null"
	t.mu.Unlock()
	return t.notify(ctx, "notifications/initialized", nil)
}
func (t *transport) notify(ctx context.Context, method string, params any) error {
	ctx, cancel := context.WithTimeout(ctx, t.shared.RequestTimeout)
	defer cancel()
	b, err := json.Marshal(struct {
		JSONRPC string `json:"jsonrpc"`
		Method  string `json:"method"`
		Params  any    `json:"params,omitempty"`
	}{"2.0", method, params})
	if err != nil {
		return err
	}
	response, err := t.post(ctx, b, nil)
	if err != nil {
		if response != nil && response.Body != nil {
			_ = response.Body.Close()
		}
		return err
	}
	defer func() { _ = response.Body.Close() }()
	if err := t.checkResponseStatus(response.StatusCode, response.StatusCode == http.StatusAccepted); err != nil {
		return t.rpcError(method, ResponseMeta{HTTPStatus: response.StatusCode}, err)
	}
	return nil
}
func (t *transport) call(ctx context.Context, method string, params any) (result json.RawMessage, meta ResponseMeta, callErr error) {
	meta.RequestID = "wx-" + strconv.FormatUint(rpcCounter.Add(1), 10)
	if ctx.Err() != nil {
		return nil, meta, t.rpcError(method, meta, context.Cause(ctx))
	}
	bound, cancel := context.WithTimeout(ctx, t.shared.RequestTimeout)
	defer cancel()
	merged, stop := context.WithCancelCause(bound)
	after := context.AfterFunc(t.life, func() { stop(context.Cause(t.life)) })
	defer func() { after(); stop(context.Canceled) }()
	if t.life.Err() != nil {
		stop(context.Cause(t.life))
	}
	request := struct {
		JSONRPC string `json:"jsonrpc"`
		ID      string `json:"id"`
		Method  string `json:"method"`
		Params  any    `json:"params,omitempty"`
	}{"2.0", meta.RequestID, method, params}
	b, err := json.Marshal(request)
	if err != nil {
		return nil, meta, t.rpcError(method, meta, err)
	}
	sent := new(atomic.Bool)
	completed := false
	defer func() {
		if merged.Err() != nil {
			callErr = errors.Join(callErr, t.rpcError(method, meta, context.Cause(merged)))
		}
		if !completed && sent.Load() && method != "initialize" && merged.Err() != nil && t.life.Err() == nil {
			notifyCtx, done := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
			defer done()
			_ = t.notify(notifyCtx, "notifications/cancelled", map[string]any{"requestId": meta.RequestID, "reason": "request cancelled"})
		}
	}()
	response, err := t.post(merged, b, sent)
	if err != nil {
		if response != nil && response.Body != nil {
			_ = response.Body.Close()
		}
		return nil, meta, t.rpcError(method, meta, errors.Join(ErrDeliveryUnknown, err))
	}
	defer func() { _ = response.Body.Close() }()
	meta.HTTPStatus = response.StatusCode
	if statusErr := t.checkResponseStatus(response.StatusCode, true); statusErr != nil {
		return nil, meta, t.rpcError(method, meta, statusErr)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 || response.StatusCode == http.StatusAccepted {
		raw, readErr := readJSON(response.Body, t.shared.MaxHTTPResponseBytes)
		var remote struct {
			JSONRPC string          `json:"jsonrpc"`
			ID      json.RawMessage `json:"id"`
			Method  string          `json:"method"`
			Error   *RPCError       `json:"error"`
			Result  json.RawMessage `json:"result"`
		}
		parseErr := json.Unmarshal(raw, &remote)
		var id string
		idErr := json.Unmarshal(remote.ID, &id)
		matched := parseErr == nil && idErr == nil && id == meta.RequestID && remote.JSONRPC == "2.0" && remote.Method == "" && !errors.Is(readErr, aibot.ErrBodyTooLarge)
		cause := aibot.ErrProtocol
		if matched && remote.Error != nil && remote.Result == nil {
			cause = remote.Error
		}
		if matched && remote.Error == nil {
			result = remote.Result
		}
		return result, meta, t.rpcError(method, meta, errors.Join(cause, readErr, parseErr, idErr))
	}
	if method == "initialize" {
		sid := response.Header.Get("Mcp-Session-Id")
		for _, b := range []byte(sid) {
			if b < 0x21 || b > 0x7e {
				return nil, meta, t.rpcError(method, meta, aibot.ErrProtocol)
			}
		}
		t.mu.Lock()
		t.sessionID = sid
		t.mu.Unlock()
	}
	evaluate := func(data []byte) error {
		var messages []json.RawMessage
		trimmed := bytes.TrimSpace(data)
		if len(trimmed) > 0 && trimmed[0] == '[' {
			t.mu.Lock()
			version := t.version
			t.mu.Unlock()
			if version != "" && version != "2025-03-26" {
				return aibot.ErrProtocol
			}
			meta.batched = true
			if decodeErr := json.Unmarshal(trimmed, &messages); decodeErr != nil {
				return errors.Join(aibot.ErrProtocol, decodeErr)
			}
			if len(messages) == 0 {
				return aibot.ErrProtocol
			}
		} else {
			messages = []json.RawMessage{data}
		}
		for _, raw := range messages {
			var message struct {
				JSONRPC string          `json:"jsonrpc"`
				ID      json.RawMessage `json:"id"`
				Method  string          `json:"method"`
				Result  json.RawMessage `json:"result"`
				Error   *RPCError       `json:"error"`
			}
			if decodeErr := json.Unmarshal(raw, &message); decodeErr != nil {
				return errors.Join(aibot.ErrProtocol, decodeErr)
			}
			if message.JSONRPC != "2.0" {
				return aibot.ErrProtocol
			}
			if message.Method != "" {
				if len(message.ID) > 0 && string(message.ID) != "null" {
					if replyErr := t.serverReply(merged, message.ID, message.Method); replyErr != nil {
						return replyErr
					}
				}
				continue
			}
			var id string
			if decodeErr := json.Unmarshal(message.ID, &id); decodeErr != nil || id != meta.RequestID || completed {
				return aibot.ErrProtocol
			}
			if message.Error != nil {
				completed = true
				return message.Error
			}
			if message.Result == nil {
				return aibot.ErrProtocol
			}
			result = append(json.RawMessage(nil), message.Result...)
			completed = true
		}
		if completed {
			return errResult
		}
		return nil
	}
	contentType, _, err := mime.ParseMediaType(response.Header.Get("Content-Type"))
	if err != nil {
		return nil, meta, t.rpcError(method, meta, errors.Join(aibot.ErrProtocol, err))
	}
	readFailed := false
	switch contentType {
	case "application/json":
		raw, readErr := readJSON(response.Body, t.shared.MaxHTTPResponseBytes)
		if errors.Is(readErr, aibot.ErrBodyTooLarge) {
			err = readErr
			break
		}
		err = evaluate(raw)
		if errors.Is(err, errResult) {
			err = nil
		}
		err = errors.Join(err, readErr)
		readFailed = readErr != nil
	case "text/event-stream":
		evaluationFailed := false
		err = readSSE(merged, response.Body, t.shared.MaxHTTPResponseBytes, func(data []byte) error {
			evaluationErr := evaluate(data)
			evaluationFailed = evaluationErr != nil
			return evaluationErr
		})
		readFailed = err != nil && !evaluationFailed && !errors.Is(err, aibot.ErrBodyTooLarge)
	default:
		err = aibot.ErrProtocol
	}
	if errors.Is(err, errResult) {
		return result, meta, nil
	}
	if err == nil && completed {
		return result, meta, nil
	}
	if err == nil {
		err = aibot.ErrProtocol
	}
	if readFailed && !completed {
		err = errors.Join(ErrDeliveryUnknown, err)
	}
	return result, meta, t.rpcError(method, meta, err)
}
func (t *transport) serverReply(ctx context.Context, id json.RawMessage, method string) error {
	payload := map[string]any{"jsonrpc": "2.0", "id": id}
	if method == "ping" {
		payload["result"] = map[string]any{}
	} else {
		payload["error"] = &RPCError{Code: -32601, Message: "method not found"}
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	response, err := t.post(ctx, b, nil)
	if err != nil {
		if response != nil && response.Body != nil {
			_ = response.Body.Close()
		}
		return err
	}
	defer func() { _ = response.Body.Close() }()
	if err := t.checkResponseStatus(response.StatusCode, response.StatusCode == http.StatusAccepted); err != nil {
		return t.rpcError("server_reply", ResponseMeta{HTTPStatus: response.StatusCode}, err)
	}
	return nil
}
func (t *transport) close() error {
	t.cancel(aibot.ErrClosed)
	t.mu.Lock()
	sid := t.sessionID
	t.mu.Unlock()
	if sid == "" {
		return nil
	}
	t.deleteOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(t.life), 2*time.Second)
		defer cancel()
		req, err := http.NewRequestWithContext(ctx, http.MethodDelete, t.config.URL, nil)
		if err != nil {
			t.deleteErr = err
			return
		}
		req.Header = t.headers()
		client := *t.shared.HTTPClient
		client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
		res, err := client.Do(req)
		if err != nil {
			if res != nil && res.Body != nil {
				_ = res.Body.Close()
			}
			t.deleteErr = err
			return
		}
		defer func() { _ = res.Body.Close() }()
		if res.StatusCode < 200 || res.StatusCode >= 300 {
			if res.StatusCode != 405 && res.StatusCode != 404 {
				t.deleteErr = fmt.Errorf("MCP DELETE HTTP status %d", res.StatusCode)
			}
		}
	})
	return aibot.WrapError("documents.close", "", t.deleteErr)
}
