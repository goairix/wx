// Package transport executes platform-independent HTTP requests.
package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	wxerrors "github.com/goairix/wx/v2/core/errors"
	"github.com/goairix/wx/v2/core/observability"
	"github.com/goairix/wx/v2/core/request"
)

// Client is an injectable HTTP transport with request retries.
type Client struct {
	HTTPClient *http.Client
	BaseURL    string
	Retry      RetryPolicy
	Hook       observability.Hook
}

// New constructs a transport client. A nil HTTP client gets a bounded default.
func New(httpClient *http.Client, baseURL string, retry RetryPolicy) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &Client{HTTPClient: httpClient, BaseURL: strings.TrimRight(baseURL, "/"), Retry: retry.normalized()}
}

// Do executes req, decoding a successful response into req.Result.
func (c *Client) Do(ctx context.Context, req request.Request) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	policy := c.Retry.normalized()
	client := c.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}

	endpoint, err := resolveURL(c.BaseURL, req.Path, req.Query)
	if err != nil {
		return err
	}
	body, err := encodeBody(req.Body)
	if err != nil {
		return fmt.Errorf("encode %s request body: %w", req.Operation, err)
	}

	var lastErr error
	for attempt := 1; attempt <= policy.MaxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		httpReq, err := http.NewRequestWithContext(ctx, req.Method, endpoint, bytes.NewReader(body))
		if err != nil {
			return err
		}
		httpReq.Header = cloneHeader(req.Header)
		if req.Body != nil && httpReq.Header.Get("Content-Type") == "" {
			httpReq.Header.Set("Content-Type", "application/json")
		}
		if c.Hook != nil {
			c.Hook.OnRequest(observability.Event{Operation: req.Operation, Platform: req.Platform})
		}
		started := time.Now()
		response, doErr := client.Do(httpReq)
		if doErr != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return ctxErr
			}
			lastErr = doErr
			if attempt < policy.MaxAttempts {
				if err := waitBackoff(ctx, policy.Backoff(attempt)); err != nil {
					return err
				}
				continue
			}
			break
		}
		responseBody, readErr := io.ReadAll(response.Body)
		_ = response.Body.Close()
		requestID := response.Header.Get("X-Request-Id")
		if requestID == "" {
			requestID = response.Header.Get("Request-Id")
		}
		if req.Meta != nil {
			req.Meta.StatusCode = response.StatusCode
			req.Meta.Header = response.Header.Clone()
			req.Meta.RequestID = requestID
		}
		if readErr != nil {
			lastErr = readErr
			break
		}
		if c.Hook != nil {
			c.Hook.OnResponse(observability.Event{Operation: req.Operation, Platform: req.Platform, StatusCode: response.StatusCode, RequestID: requestID, Duration: time.Since(started)})
		}
		if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
			lastErr = wxerrors.ParsePlatformError(req.Platform, req.Operation, response.StatusCode, responseBody, requestID)
			if attempt < policy.MaxAttempts && policy.RetryStatus[response.StatusCode] {
				if err := waitBackoff(ctx, policy.Backoff(attempt)); err != nil {
					return err
				}
				continue
			}
			return lastErr
		}
		if req.Result == nil || len(responseBody) == 0 || response.StatusCode == http.StatusNoContent {
			return nil
		}
		if err := json.Unmarshal(responseBody, req.Result); err != nil {
			return fmt.Errorf("decode %s response: %w", req.Operation, err)
		}
		return nil
	}
	if lastErr != nil {
		return lastErr
	}
	return fmt.Errorf("request %s failed", req.Operation)
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
