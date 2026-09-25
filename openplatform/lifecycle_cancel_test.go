package openplatform

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

type checkedContext struct {
	context.Context
	checked chan struct{}
	once    sync.Once
}

func (c *checkedContext) Err() error {
	err := c.Context.Err()
	c.once.Do(func() { close(c.checked) })
	return err
}

func TestAuthorizerLifecycleCanCancelWhileAnotherUpdateIsSaving(t *testing.T) {
	for _, action := range []string{"update", "revoke"} {
		t.Run(action, func(t *testing.T) {
			started := make(chan struct{})
			release := make(chan struct{})
			firstDone := make(chan error, 1)
			client, err := NewClient(Config{AppID: "component", AppSecret: "secret"},
				WithRefreshTokenStore(RefreshTokenStoreFunc(func(_ context.Context, _, token string) error {
					if token == "holding" {
						close(started)
						<-release
					}
					return nil
				})),
			)
			if err != nil {
				t.Fatal(err)
			}
			go func() { firstDone <- client.UpdateAuthorizer(context.Background(), "authorizer", "holding") }()
			<-started
			defer func() {
				close(release)
				if err := <-firstDone; err != nil {
					t.Error(err)
				}
			}()
			parent, cancel := context.WithCancel(context.Background())
			defer cancel()
			ctx := &checkedContext{Context: parent, checked: make(chan struct{})}
			done := make(chan error, 1)
			go func() {
				if action == "update" {
					done <- client.UpdateAuthorizer(ctx, "authorizer", "replacement")
				} else {
					done <- client.RevokeAuthorizer(ctx, "authorizer")
				}
			}()
			// Cancel after the initial context check, while the gate is occupied.
			<-ctx.checked
			cancel()
			select {
			case err := <-done:
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("error=%v", err)
				}
			case <-time.After(time.Second):
				t.Error("canceled operation waited for another caller's storage operation")
			}
		})
	}
}
