package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strconv"

	coreerrors "github.com/goairix/wx/v2/core/errors"
	"github.com/goairix/wx/v2/work/aibot"
)

type ResponseResult struct {
	HTTPStatus int
	ErrCode    int             `json:"errcode"`
	ErrMsg     string          `json:"errmsg"`
	Raw        json.RawMessage `json:"-"`
}

func (c *Client) ReplyByURL(ctx context.Context, responseURL string, message aibot.OutgoingMessage) (*ResponseResult, error) {
	const operation = "webhook.reply_by_url"
	if ctx == nil {
		return nil, aibot.WrapError(operation, "", aibot.ErrInvalidArgument)
	}
	u, err := url.Parse(responseURL)
	if err != nil || u.Host == "" || u.User != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return nil, aibot.WrapError(operation, "", aibot.ErrInvalidArgument)
	}
	if message.MsgType != "markdown" && message.MsgType != "template_card" {
		return nil, aibot.WrapError(operation, "", aibot.ErrUnsupported)
	}
	if !withinBudget(reflect.ValueOf(message), c.config.MaxPlainReplyBytes) {
		return nil, aibot.WrapError(operation, "", aibot.ErrBodyTooLarge)
	}
	if err = aibot.ValidateMessage(message); err != nil {
		return nil, aibot.WrapError(operation, "", err)
	}
	b, err := json.Marshal(message)
	if err != nil {
		return nil, aibot.WrapError(operation, "", err)
	}
	if int64(len(b)) > c.config.MaxPlainReplyBytes {
		return nil, aibot.WrapError(operation, "", aibot.ErrBodyTooLarge)
	}
	requestCtx, cancel := context.WithTimeout(ctx, c.shared.RequestTimeout)
	defer cancel()
	request, err := http.NewRequestWithContext(requestCtx, http.MethodPost, responseURL, bytes.NewReader(b))
	if err != nil {
		return nil, aibot.WrapError(operation, "", err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.GetBody = nil
	client := *c.shared.HTTPClient
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	response, err := client.Do(request)
	if err != nil {
		if response != nil && response.Body != nil {
			_ = response.Body.Close()
		}
		return nil, aibot.WrapError(operation, "", err)
	}
	defer func() { _ = response.Body.Close() }()
	result := &ResponseResult{HTTPStatus: response.StatusCode}
	data, readErr := io.ReadAll(io.LimitReader(response.Body, c.shared.MaxHTTPResponseBytes+1))
	if int64(len(data)) > c.shared.MaxHTTPResponseBytes {
		data = data[:c.shared.MaxHTTPResponseBytes]
		readErr = errors.Join(readErr, aibot.ErrBodyTooLarge)
	}
	result.Raw = append(json.RawMessage(nil), data...)
	parseErr := json.Unmarshal(data, result)
	var fields map[string]json.RawMessage
	if parseErr == nil {
		parseErr = json.Unmarshal(data, &fields)
		if fields == nil || fields["errcode"] == nil || bytes.Equal(bytes.TrimSpace(fields["errcode"]), []byte("null")) {
			parseErr = aibot.ErrProtocol
		}
	}
	if parseErr != nil {
		parseErr = errors.Join(aibot.ErrProtocol, parseErr)
	}
	cause := errors.Join(readErr, parseErr)
	if result.ErrCode != 0 || response.StatusCode < 200 || response.StatusCode >= 300 {
		cause = errors.Join(cause, aibot.ErrProtocol)
	}
	if cause != nil {
		return result, &coreerrors.Error{Platform: "work", Operation: "work.aibot." + operation, HTTPStatus: response.StatusCode, Code: strconv.Itoa(result.ErrCode), Message: result.ErrMsg, Err: cause}
	}
	return result, nil
}
