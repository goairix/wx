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
	observer         observability.Observer
	logger           logging.Logger
	maxResponseBytes int64
}

// Option configures a transport client during construction.
type Option interface {
	apply(*clientOptions)
}

type clientOptions struct {
	hook             observability.Hook
	observer         observability.Observer
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

// WithObserver instruments each HTTP attempt and may derive its context.
func WithObserver(observer observability.Observer) Option {
	return optionFunc(func(options *clientOptions) { options.observer = observer })
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
		observer:         settings.observer,
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

	body, encodeErr := encodeBody(req.Body)
	if encodeErr != nil {
		encodedErr := fmt.Errorf("encode %s request body: %w", req.Operation, encodeErr)
		c.logFailed(ctx, req, 0, 0, policy.MaxAttempts, 0, "", encodedErr)
		return encodedErr
	}

	for attempt := 1; attempt <= policy.MaxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			c.logFailed(ctx, req, 0, attempt, policy.MaxAttempts, 0, "", err)
			return err
		}
		outcome := c.doAttempt(ctx, client, req, body, attempt, policy.MaxAttempts)
		event := outcome.event
		if event.Err == nil {
			c.logCompleted(event.Context, req, event.StatusCode, attempt, policy.MaxAttempts, event.Duration, event.RequestID)
			return nil
		}
		if attempt < policy.MaxAttempts && retryableRequest(req) && outcome.retryable &&
			(event.StatusCode == 0 || policy.RetryStatus[event.StatusCode]) {
			delay := policy.Backoff(attempt)
			c.logRetrying(event.Context, req, event.StatusCode, attempt, policy.MaxAttempts, event.Duration, delay, event.RequestID, event.Err)
			if err := waitBackoff(ctx, delay); err != nil {
				c.logFailed(event.Context, req, event.StatusCode, attempt, policy.MaxAttempts, event.Duration, event.RequestID, err)
				return err
			}
			continue
		}
		c.logFailed(event.Context, req, event.StatusCode, attempt, policy.MaxAttempts, event.Duration, event.RequestID, event.Err)
		return event.Err
	}
	return fmt.Errorf("request %s failed", req.Operation)
}

type attemptOutcome struct {
	event     observability.Event
	retryable bool
}

// doAttempt owns the observation lifecycle so every exit completes exactly once.
func (c *Client) doAttempt(
	ctx context.Context,
	client *http.Client,
	req request.Request,
	body []byte,
	attempt int,
	maxAttempts int,
) (outcome attemptOutcome) {
	outcome.event = observability.Event{
		Context:     ctx,
		Operation:   req.Operation,
		Platform:    req.Platform,
		Attempt:     attempt,
		MaxAttempts: maxAttempts,
	}
	var finish func(observability.Event)
	if c.observer != nil {
		derived, end := c.observer.Start(ctx, outcome.event)
		if derived != nil {
			ctx = derived
		}
		finish = end
	}
	outcome.event.Context = ctx
	started := time.Now()
	hookStarted := false
	defer func() {
		if err := ctx.Err(); err != nil {
			outcome.event.Err = err
			outcome.retryable = false
		}
		outcome.event.Duration = time.Since(started)
		if hookStarted && c.hook != nil {
			c.hook.OnResponse(outcome.event)
		}
		if finish != nil {
			event := outcome.event
			// URL wrappers may contain credentials in their query string. Hooks
			// retain legacy errors; new observations use the same safe error as logs.
			event.Err = safeLogError(event.Err)
			finish(event)
		}
	}()
	if err := ctx.Err(); err != nil {
		outcome.event.Err = err
		return
	}
	endpoint, err := resolveURL(c.baseURL, req.Path, req.Query)
	if err != nil {
		outcome.event.Err = err
		return
	}
	httpReq, err := http.NewRequestWithContext(ctx, req.Method, endpoint, bytes.NewReader(body))
	if err != nil {
		outcome.event.Err = err
		return
	}
	if req.RetryMode == request.RetryNever {
		// net/http can replay GETs on a reused connection independently of our
		// retry loop. An opaque, non-rewindable body prevents that replay, even
		// for an empty GET. For empty GETs, the standard transport probes the
		// reader and sends no body or chunked encoding.
		httpReq.Body = io.NopCloser(bytes.NewReader(body))
		httpReq.GetBody = nil
	}
	httpReq.Header = cloneHeader(req.Header)
	if req.Body != nil && httpReq.Header.Get("Content-Type") == "" {
		httpReq.Header.Set("Content-Type", "application/json")
	}
	c.logStarted(ctx, req, attempt, maxAttempts)
	if c.hook != nil {
		c.hook.OnRequest(outcome.event)
		hookStarted = true
	}
	started = time.Now()
	response, err := client.Do(httpReq)
	if err != nil {
		outcome.event.Err = err
		outcome.retryable = true
		return
	}
	defer func() { _ = response.Body.Close() }()
	outcome.event.StatusCode = response.StatusCode
	outcome.event.RequestID = response.Header.Get("X-Request-Id")
	if outcome.event.RequestID == "" {
		outcome.event.RequestID = response.Header.Get("Request-Id")
	}
	meta := request.ResponseMeta{
		StatusCode: response.StatusCode,
		Header:     response.Header.Clone(),
		RequestID:  outcome.event.RequestID,
	}
	if req.Meta != nil {
		*req.Meta = meta
	}
	success := response.StatusCode >= http.StatusOK && response.StatusCode < http.StatusMultipleChoices
	if req.ResponseWriter != nil && success && !strings.Contains(strings.ToLower(response.Header.Get("Content-Type")), "json") {
		_, outcome.event.Err = io.Copy(req.ResponseWriter, response.Body)
		return
	}
	responseBody, err := readAllLimited(response.Body, c.maxResponseBytes)
	if err != nil {
		outcome.event.Err = err
		return
	}
	if !success {
		outcome.event.Err = wxerrors.ParsePlatformError(req.Platform, req.Operation, response.StatusCode, responseBody, meta.RequestID)
		outcome.retryable = true
		return
	}
	outcome.event.Err = decodeResponse(req, responseBody, meta)
	return
}

func decodeResponse(req request.Request, body []byte, meta request.ResponseMeta) error {
	if decoder, ok := req.Result.(request.ResponseDecoder); ok {
		if meta.StatusCode == http.StatusNoContent {
			return nil
		}
		if len(bytes.TrimSpace(body)) == 0 {
			return fmt.Errorf("decode %s response: %w", req.Operation, io.ErrUnexpectedEOF)
		}
		return decoder.DecodeResponse(body, meta)
	}
	if err := wxerrors.ParseResponseError(req.Platform, req.Operation, meta.StatusCode, body, meta.RequestID); err != nil {
		return err
	}
	if req.ResponseWriter != nil {
		_, err := req.ResponseWriter.Write(body)
		return err
	}
	if req.Result == nil || meta.StatusCode == http.StatusNoContent {
		return nil
	}
	if result, ok := req.Result.(*[]byte); ok {
		*result = append((*result)[:0], body...)
		return nil
	}
	if err := json.Unmarshal(body, req.Result); err != nil {
		return fmt.Errorf("decode %s response: %w", req.Operation, err)
	}
	return nil
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
