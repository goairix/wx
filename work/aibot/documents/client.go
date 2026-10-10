package documents

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"sync"

	"github.com/goairix/wx/v2/work/aibot"
	"github.com/goairix/wx/v2/work/aibot/internal/settings"
)

type Config struct {
	URL    string
	Header http.Header
}
type Client struct {
	mu                        sync.Mutex
	config                    Config
	shared                    settings.Settings
	borrowed                  Caller
	transport                 *transport
	life                      context.Context
	cancel                    context.CancelCauseFunc
	connecting, ready, closed bool
}

func New(cfg Config, opts ...aibot.Option) (*Client, error) {
	u, err := url.Parse(cfg.URL)
	if err != nil || u.Host == "" || u.User != nil || u.Fragment != "" || (u.Scheme != "http" && u.Scheme != "https") {
		return nil, aibot.WrapError("documents.new", "", aibot.ErrInvalidArgument)
	}
	shared, err := settings.Resolve(opts...)
	if err != nil {
		return nil, aibot.WrapError("documents.new", "", aibot.ErrInvalidArgument)
	}
	cfg.Header = cfg.Header.Clone()
	return &Client{config: cfg, shared: shared}, nil
}
func NewWithCaller(caller Caller, opts ...aibot.Option) (*Client, error) {
	if settings.IsNil(caller) {
		return nil, aibot.WrapError("documents.new", "", aibot.ErrInvalidArgument)
	}
	shared, err := settings.Resolve(opts...)
	if err != nil {
		return nil, aibot.WrapError("documents.new", "", aibot.ErrInvalidArgument)
	}
	return &Client{borrowed: caller, shared: shared}, nil
}
func (c *Client) Connect(ctx context.Context) error {
	if ctx == nil {
		return aibot.WrapError("documents.connect", "", aibot.ErrInvalidArgument)
	}
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return aibot.WrapError("documents.connect", "", aibot.ErrClosed)
	}
	if c.connecting {
		c.mu.Unlock()
		return aibot.WrapError("documents.connect", "", aibot.ErrNotReady)
	}
	if c.life != nil && c.life.Err() == nil && c.ready {
		c.mu.Unlock()
		return nil
	}
	if c.life != nil && c.life.Err() != nil && !errors.Is(context.Cause(c.life), ErrSessionExpired) {
		c.mu.Unlock()
		return aibot.WrapError("documents.connect", "", aibot.ErrClosed)
	}
	life, cancel := context.WithCancelCause(ctx)
	c.life = life
	c.cancel = cancel
	c.connecting = true
	c.ready = false
	t := &transport{config: c.config, shared: c.shared, life: life, cancel: cancel}
	if c.borrowed == nil {
		c.transport = t
	}
	c.mu.Unlock()
	var err error
	if c.borrowed == nil {
		err = t.initialize(life)
	} else {
		err = life.Err()
	}
	c.mu.Lock()
	c.connecting = false
	if c.closed {
		err = errors.Join(err, aibot.ErrClosed)
	} else if life.Err() != nil {
		err = errors.Join(err, context.Cause(life))
	}
	c.ready = err == nil
	if err != nil {
		c.life = nil
		c.transport = nil
		cancel(err)
	}
	c.mu.Unlock()
	if err != nil {
		if c.borrowed == nil {
			_ = t.close()
		}
		return aibot.WrapError("documents.connect", "", err)
	}
	return nil
}
func (c *Client) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	c.ready = false
	t := c.transport
	if c.cancel != nil {
		c.cancel(aibot.ErrClosed)
	}
	c.mu.Unlock()
	if t != nil {
		return t.close()
	}
	return nil
}
func (c *Client) bound(ctx context.Context) (context.Context, func(), Caller, error) {
	if ctx == nil {
		return nil, nil, nil, aibot.ErrInvalidArgument
	}
	c.mu.Lock()
	life, ready, closed, t, borrowed := c.life, c.ready, c.closed, c.transport, c.borrowed
	c.mu.Unlock()
	if closed {
		return nil, nil, nil, aibot.ErrClosed
	}
	if !ready || life == nil {
		return nil, nil, nil, aibot.ErrNotReady
	}
	if life.Err() != nil {
		return nil, nil, nil, context.Cause(life)
	}
	bound, cancel := context.WithTimeout(ctx, c.shared.RequestTimeout)
	merged, stop := context.WithCancelCause(bound)
	after := context.AfterFunc(life, func() { stop(context.Cause(life)) })
	if life.Err() != nil {
		stop(context.Cause(life))
	}
	cleanup := func() { after(); stop(context.Canceled); cancel() }
	var caller Caller = t
	if borrowed != nil {
		caller = borrowed
	}
	return merged, cleanup, caller, nil
}
func (c *Client) Tools(ctx context.Context) ([]Tool, error) {
	bound, cancel, caller, err := c.bound(ctx)
	if err != nil {
		return nil, aibot.WrapError("documents.tools.list", "", err)
	}
	defer cancel()
	result, err := caller.Tools(bound)
	if bound.Err() != nil {
		err = errors.Join(err, context.Cause(bound))
	}
	return result, wrapError("tools.list", nil, err)
}
func (c *Client) CallTool(ctx context.Context, name string, args any) (*ToolResult, error) {
	if name == "" {
		return nil, aibot.WrapError("documents.call_tool", "", aibot.ErrInvalidArgument)
	}
	bound, cancel, caller, err := c.bound(ctx)
	if err != nil {
		return nil, aibot.WrapError("documents."+name, "", err)
	}
	defer cancel()
	result, err := caller.CallTool(bound, name, args)
	if bound.Err() != nil {
		err = errors.Join(err, context.Cause(bound))
	}
	if result != nil {
		if result.IsError {
			err = errors.Join(err, &ToolError{Tool: name, Result: result})
		}
	}
	return result, wrapError(name, result, err)
}
