package transport

import (
	"context"
	"errors"
	"net/url"
	"time"

	wxerrors "github.com/goairix/wx/v2/core/errors"
	"github.com/goairix/wx/v2/core/logging"
	"github.com/goairix/wx/v2/core/request"
)

const (
	requestStartedEvent   = "wx.request.started"
	requestCompletedEvent = "wx.request.completed"
	requestRetryingEvent  = "wx.request.retrying"
	requestFailedEvent    = "wx.request.failed"
)

type requestLog struct {
	statusCode  int
	attempt     int
	maxAttempts int
	nextAttempt int
	duration    time.Duration
	retryDelay  time.Duration
	requestID   string
	err         error
}

func normalizeLogger(logger logging.Logger) logging.Logger {
	if logger == nil {
		return logging.Nop()
	}
	return logger
}

func (c *Client) logStarted(
	ctx context.Context,
	req request.Request,
	attempt int,
	maxAttempts int,
) {
	c.logRequest(ctx, logging.LevelDebug, requestStartedEvent, req, requestLog{
		attempt:     attempt,
		maxAttempts: maxAttempts,
	})
}

func (c *Client) logCompleted(
	ctx context.Context,
	req request.Request,
	statusCode int,
	attempt int,
	maxAttempts int,
	duration time.Duration,
	requestID string,
) {
	c.logRequest(ctx, logging.LevelDebug, requestCompletedEvent, req, requestLog{
		statusCode:  statusCode,
		attempt:     attempt,
		maxAttempts: maxAttempts,
		duration:    duration,
		requestID:   requestID,
	})
}

func (c *Client) logRetrying(
	ctx context.Context,
	req request.Request,
	statusCode int,
	attempt int,
	maxAttempts int,
	duration time.Duration,
	retryDelay time.Duration,
	requestID string,
	err error,
) {
	c.logRequest(ctx, logging.LevelWarn, requestRetryingEvent, req, requestLog{
		statusCode:  statusCode,
		attempt:     attempt,
		maxAttempts: maxAttempts,
		nextAttempt: attempt + 1,
		duration:    duration,
		retryDelay:  retryDelay,
		requestID:   requestID,
		err:         err,
	})
}

func (c *Client) logFailed(
	ctx context.Context,
	req request.Request,
	statusCode int,
	attempt int,
	maxAttempts int,
	duration time.Duration,
	requestID string,
	err error,
) {
	c.logRequest(ctx, logging.LevelError, requestFailedEvent, req, requestLog{
		statusCode:  statusCode,
		attempt:     attempt,
		maxAttempts: maxAttempts,
		duration:    duration,
		requestID:   requestID,
		err:         err,
	})
}

func (c *Client) logRequest(
	ctx context.Context,
	level logging.Level,
	event string,
	req request.Request,
	entry requestLog,
) {
	var platformErr *wxerrors.Error
	if entry.err != nil && errors.As(entry.err, &platformErr) {
		if entry.statusCode == 0 {
			entry.statusCode = platformErr.HTTPStatus
		}
		if entry.requestID == "" {
			entry.requestID = platformErr.RequestID
		}
	}

	attrs := make([]logging.Attr, 0, 12)
	if req.Platform != "" {
		attrs = append(attrs, logging.String("platform", req.Platform))
	}
	if req.Operation != "" {
		attrs = append(attrs, logging.String("operation", req.Operation))
	}
	if req.Method != "" {
		attrs = append(attrs, logging.String("method", req.Method))
	}
	if entry.statusCode != 0 {
		attrs = append(attrs, logging.Int("status", entry.statusCode))
	}
	if platformErr != nil && platformErr.Code != "" {
		attrs = append(attrs, logging.String("code", platformErr.Code))
	}
	if entry.attempt != 0 {
		attrs = append(attrs, logging.Int("attempt", entry.attempt))
	}
	if entry.maxAttempts != 0 {
		attrs = append(attrs, logging.Int("max_attempts", entry.maxAttempts))
	}
	if entry.nextAttempt != 0 {
		attrs = append(attrs, logging.Int("next_attempt", entry.nextAttempt))
	}
	if entry.duration != 0 {
		attrs = append(attrs, logging.Duration("duration", entry.duration))
	}
	if entry.retryDelay != 0 {
		attrs = append(attrs, logging.Duration("retry_delay", entry.retryDelay))
	}
	if entry.requestID != "" {
		attrs = append(attrs, logging.String("request_id", entry.requestID))
	}
	if entry.err != nil {
		attrs = append(attrs, logging.Error(safeLogError(entry.err)))
	}
	normalizeLogger(c.logger).Log(ctx, level, event, attrs...)
}

func safeLogError(err error) error {
	var platformErr *wxerrors.Error
	if errors.As(err, &platformErr) {
		return err
	}

	var urlErr *url.Error
	if errors.As(err, &urlErr) && urlErr.Err != nil {
		return urlErr.Err
	}
	return err
}
