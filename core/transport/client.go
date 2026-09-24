// Package transport executes platform-independent HTTP requests.
package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	wxerrors "github.com/goairix/wx/v2/core/errors"
	"github.com/goairix/wx/v2/core/logging"
	"github.com/goairix/wx/v2/core/observability"
	"github.com/goairix/wx/v2/core/request"
)

const defaultMaxResponseBytes int64 = 32 << 20

// ErrResponseTooLarge is returned when a buffered response exceeds its limit.
var ErrResponseTooLarge = errors.New("transport: response body exceeds configured limit")

// Client is an injectable HTTP transport with request retries.
type Client struct {
	httpClient       *http.Client
	baseURL          string
	retry            RetryPolicy
	hook             observability.Hook
	logger           logging.Logger
	maxResponseBytes int64
}

// Option configures a transport client during construction.
type Option interface {
	apply(*clientOptions)
}

type clientOptions struct {
	hook             observability.Hook
	logger           logging.Logger
	maxResponseBytes int64
}

type optionFunc func(*clientOptions)

func (configure optionFunc) apply(options *clientOptions) {
	configure(options)
}

// WithMaxResponseBytes sets the maximum buffered response size. A negative
// value disables the limit.
func WithMaxResponseBytes(limit int64) Option {
	return optionFunc(func(options *clientOptions) {
		options.maxResponseBytes = limit
	})
}

// WithHook observes every HTTP attempt and its final transport or platform
// result.
func WithHook(hook observability.Hook) Option {
	return optionFunc(func(options *clientOptions) {
		options.hook = hook
	})
}

// WithLogger configures the structured logger used for request lifecycle
// events. A nil logger disables logging.
func WithLogger(logger logging.Logger) Option {
	return optionFunc(func(options *clientOptions) {
		options.logger = logger
	})
}

// New constructs a transport client. A nil HTTP client gets a bounded default.
func New(
	httpClient *http.Client,
	baseURL string,
	retry RetryPolicy,
	options ...Option,
) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	settings := clientOptions{maxResponseBytes: defaultMaxResponseBytes}
	for _, option := range options {
		if option != nil {
			option.apply(&settings)
		}
	}
	return &Client{
		httpClient:       httpClient,
		baseURL:          strings.TrimRight(baseURL, "/"),
		retry:            retry.normalized(),
		hook:             settings.hook,
		logger:           normalizeLogger(settings.logger),
		maxResponseBytes: settings.maxResponseBytes,
	}
}

// Do executes req, decoding a successful response into req.Result.
func (c *Client) Do(ctx context.Context, req request.Request) error {
	if ctx == nil {
		ctx = context.Background()
	}
	policy := c.retry.normalized()
	if err := ctx.Err(); err != nil {
		c.logFailed(ctx, req, 0, 0, policy.MaxAttempts, 0, "", err)
		return err
	}
	if req.Result != nil && req.ResponseWriter != nil {
		err := fmt.Errorf("transport: Result and ResponseWriter cannot both be set")
		c.logFailed(ctx, req, 0, 0, policy.MaxAttempts, 0, "", err)
		return err
	}
	client := c.httpClient
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}

	endpoint, resolveErr := resolveURL(c.baseURL, req.Path, req.Query)
	if resolveErr != nil {
		c.logFailed(ctx, req, 0, 0, policy.MaxAttempts, 0, "", resolveErr)
		return resolveErr
	}
	body, encodeErr := encodeBody(req.Body)
	if encodeErr != nil {
		encodedErr := fmt.Errorf("encode %s request body: %w", req.Operation, encodeErr)
		c.logFailed(ctx, req, 0, 0, policy.MaxAttempts, 0, "", encodedErr)
		return encodedErr
	}

	var lastErr error
	for attempt := 1; attempt <= policy.MaxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			c.logFailed(ctx, req, 0, attempt, policy.MaxAttempts, 0, "", err)
			return err
		}
		httpReq, err := http.NewRequestWithContext(ctx, req.Method, endpoint, bytes.NewReader(body))
		if err != nil {
			c.logFailed(ctx, req, 0, attempt, policy.MaxAttempts, 0, "", err)
			return err
		}
		httpReq.Header = cloneHeader(req.Header)
		if req.Body != nil && httpReq.Header.Get("Content-Type") == "" {
			httpReq.Header.Set("Content-Type", "application/json")
		}
		c.logStarted(ctx, req, attempt, policy.MaxAttempts)
		if c.hook != nil {
			c.hook.OnRequest(observability.Event{
				Context:   ctx,
				Operation: req.Operation,
				Platform:  req.Platform,
			})
		}
		started := time.Now()
		response, doErr := client.Do(httpReq)
		if doErr != nil {
			duration := time.Since(started)
			if ctxErr := ctx.Err(); ctxErr != nil {
				c.observeResponse(ctx, req, 0, "", duration, ctxErr)
				c.logFailed(ctx, req, 0, attempt, policy.MaxAttempts, duration, "", ctxErr)
				return ctxErr
			}
			lastErr = doErr
			c.observeResponse(ctx, req, 0, "", duration, doErr)
			if attempt < policy.MaxAttempts && retryableRequest(req) {
				delay := policy.Backoff(attempt)
				c.logRetrying(ctx, req, 0, attempt, policy.MaxAttempts, duration, delay, "", doErr)
				if err := waitBackoff(ctx, delay); err != nil {
					c.logFailed(ctx, req, 0, attempt, policy.MaxAttempts, duration, "", err)
					return err
				}
				continue
			}
			c.logFailed(ctx, req, 0, attempt, policy.MaxAttempts, duration, "", doErr)
			break
		}
		requestID := response.Header.Get("X-Request-Id")
		if requestID == "" {
			requestID = response.Header.Get("Request-Id")
		}
		if req.Meta != nil {
			req.Meta.StatusCode = response.StatusCode
			req.Meta.Header = response.Header.Clone()
			req.Meta.RequestID = requestID
		}
		if req.ResponseWriter != nil &&
			response.StatusCode >= http.StatusOK &&
			response.StatusCode < http.StatusMultipleChoices &&
			!strings.Contains(strings.ToLower(response.Header.Get("Content-Type")), "json") {
			_, copyErr := io.Copy(req.ResponseWriter, response.Body)
			_ = response.Body.Close()
			duration := time.Since(started)
			c.observeResponse(
				ctx,
				req,
				response.StatusCode,
				requestID,
				duration,
				copyErr,
			)
			if copyErr != nil {
				c.logFailed(ctx, req, response.StatusCode, attempt, policy.MaxAttempts, duration, requestID, copyErr)
			} else {
				c.logCompleted(ctx, req, response.StatusCode, attempt, policy.MaxAttempts, duration, requestID)
			}
			return copyErr
		}

		responseBody, readErr := readAllLimited(response.Body, c.maxResponseBytes)
		_ = response.Body.Close()
		if readErr != nil {
			lastErr = readErr
			duration := time.Since(started)
			c.observeResponse(
				ctx,
				req,
				response.StatusCode,
				requestID,
				duration,
				readErr,
			)
			c.logFailed(ctx, req, response.StatusCode, attempt, policy.MaxAttempts, duration, requestID, readErr)
			break
		}
		if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
			lastErr = wxerrors.ParsePlatformError(req.Platform, req.Operation, response.StatusCode, responseBody, requestID)
			duration := time.Since(started)
			c.observeResponse(
				ctx,
				req,
				response.StatusCode,
				requestID,
				duration,
				lastErr,
			)
			if attempt < policy.MaxAttempts &&
				retryableRequest(req) &&
				policy.RetryStatus[response.StatusCode] {
				delay := policy.Backoff(attempt)
				c.logRetrying(ctx, req, response.StatusCode, attempt, policy.MaxAttempts, duration, delay, requestID, lastErr)
				if err := waitBackoff(ctx, delay); err != nil {
					c.logFailed(ctx, req, response.StatusCode, attempt, policy.MaxAttempts, duration, requestID, err)
					return err
				}
				continue
			}
			c.logFailed(ctx, req, response.StatusCode, attempt, policy.MaxAttempts, duration, requestID, lastErr)
			return lastErr
		}
		if platformErr := wxerrors.ParseResponseError(
			req.Platform,
			req.Operation,
			response.StatusCode,
			responseBody,
			requestID,
		); platformErr != nil {
			duration := time.Since(started)
			c.observeResponse(
				ctx,
				req,
				response.StatusCode,
				requestID,
				duration,
				platformErr,
			)
			c.logFailed(ctx, req, response.StatusCode, attempt, policy.MaxAttempts, duration, requestID, platformErr)
			return platformErr
		}
		if req.ResponseWriter != nil {
			_, writeErr := req.ResponseWriter.Write(responseBody)
			duration := time.Since(started)
			c.observeResponse(
				ctx,
				req,
				response.StatusCode,
				requestID,
				duration,
				writeErr,
			)
			if writeErr != nil {
				c.logFailed(ctx, req, response.StatusCode, attempt, policy.MaxAttempts, duration, requestID, writeErr)
			} else {
				c.logCompleted(ctx, req, response.StatusCode, attempt, policy.MaxAttempts, duration, requestID)
			}
			return writeErr
		}
		if req.Result == nil || len(responseBody) == 0 || response.StatusCode == http.StatusNoContent {
			duration := time.Since(started)
			c.observeResponse(ctx, req, response.StatusCode, requestID, duration, nil)
			c.logCompleted(ctx, req, response.StatusCode, attempt, policy.MaxAttempts, duration, requestID)
			return nil
		}
		if bytesResult, ok := req.Result.(*[]byte); ok {
			*bytesResult = append((*bytesResult)[:0], responseBody...)
			duration := time.Since(started)
			c.observeResponse(ctx, req, response.StatusCode, requestID, duration, nil)
			c.logCompleted(ctx, req, response.StatusCode, attempt, policy.MaxAttempts, duration, requestID)
			return nil
		}
		if err := json.Unmarshal(responseBody, req.Result); err != nil {
			decodeErr := fmt.Errorf("decode %s response: %w", req.Operation, err)
			duration := time.Since(started)
			c.observeResponse(
				ctx,
				req,
				response.StatusCode,
				requestID,
				duration,
				decodeErr,
			)
			c.logFailed(ctx, req, response.StatusCode, attempt, policy.MaxAttempts, duration, requestID, decodeErr)
			return decodeErr
		}
		duration := time.Since(started)
		c.observeResponse(ctx, req, response.StatusCode, requestID, duration, nil)
		c.logCompleted(ctx, req, response.StatusCode, attempt, policy.MaxAttempts, duration, requestID)
		return nil
	}
	if lastErr != nil {
		return lastErr
	}
	requestErr := fmt.Errorf("request %s failed", req.Operation)
	c.logFailed(ctx, req, 0, 0, policy.MaxAttempts, 0, "", requestErr)
	return requestErr
}

func readAllLimited(reader io.Reader, limit int64) ([]byte, error) {
	if limit < 0 {
		return io.ReadAll(reader)
	}
	data, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, ErrResponseTooLarge
	}
	return data, nil
}

func (c *Client) observeResponse(
	ctx context.Context,
	req request.Request,
	statusCode int,
	requestID string,
	duration time.Duration,
	err error,
) {
	if c.hook == nil {
		return
	}
	c.hook.OnResponse(observability.Event{
		Context:    ctx,
		Operation:  req.Operation,
		Platform:   req.Platform,
		StatusCode: statusCode,
		RequestID:  requestID,
		Duration:   duration,
		Err:        err,
	})
}

func retryableRequest(req request.Request) bool {
	switch req.RetryMode {
	case request.RetryNever:
		return false
	case request.RetryAlways:
		return true
	}

	switch req.Method {
	case http.MethodGet,
		http.MethodHead,
		http.MethodOptions,
		http.MethodPut,
		http.MethodDelete:
		return true
	default:
		return false
	}
}

func resolveURL(base, path string, query url.Values) (string, error) {
	baseURL, err := url.Parse(base)
	if err != nil {
		return "", fmt.Errorf("parse base URL: %w", err)
	}
	pathURL, err := url.Parse(path)
	if err != nil {
		return "", fmt.Errorf("parse request path: %w", err)
	}
	pathQuery := pathURL.Query()
	if pathURL.IsAbs() {
		baseURL = pathURL
		pathQuery = nil
	} else {
		// ResolveReference treats a leading slash as an absolute path and drops
		// the base path. APIs commonly use a base prefix such as /v1, so join
		// escaped paths explicitly while preserving encoded path segments.
		basePath := strings.TrimRight(baseURL.EscapedPath(), "/")
		requestPath := pathURL.EscapedPath()
		if requestPath == "" {
			requestPath = "/"
		}
		if !strings.HasPrefix(requestPath, "/") {
			requestPath = "/" + requestPath
		}
		joinedPath := basePath + requestPath
		baseURL.Path, err = url.PathUnescape(joinedPath)
		if err != nil {
			return "", fmt.Errorf("unescape request path: %w", err)
		}
		if escaped := baseURL.EscapedPath(); escaped != joinedPath {
			baseURL.RawPath = joinedPath
		} else {
			baseURL.RawPath = ""
		}
	}
	merged := baseURL.Query()
	for key, values := range pathQuery {
		for _, value := range values {
			merged.Add(key, value)
		}
	}
	for key, values := range query {
		for _, value := range values {
			merged.Add(key, value)
		}
	}
	if len(merged) > 0 {
		baseURL.RawQuery = merged.Encode()
	} else {
		baseURL.RawQuery = ""
	}
	return baseURL.String(), nil
}

func encodeBody(body interface{}) ([]byte, error) {
	if body == nil {
		return nil, nil
	}
	if raw, ok := body.([]byte); ok {
		return append([]byte(nil), raw...), nil
	}
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(body); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func cloneHeader(header http.Header) http.Header {
	if header == nil {
		return make(http.Header)
	}
	return header.Clone()
}

func waitBackoff(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return ctx.Err()
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-timer.C:
		return ctx.Err()
	case <-ctx.Done():
		return ctx.Err()
	}
}
