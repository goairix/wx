package wechat

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"

	"github.com/goairix/wx/v2/core/auth"
	wxerrors "github.com/goairix/wx/v2/core/errors"
)

// DoWithToken recovers once from an explicit rejection of the token used by the
// current operation. send reports whether it is safe to replay; streaming callers
// must return false after attempting a destination write. Credential failures
// and unrelated platform errors are never replayed. The caller's context is used
// unchanged for invalidation and refresh, including external persistence hooks.
func DoWithToken(ctx context.Context, manager auth.Provider, platform, operation string, send func(auth.Credential) (bool, error)) error {
	if ctx == nil {
		ctx = context.Background()
	}
	credential, err := manager.Token(ctx)
	if err != nil {
		return err
	}
	for attempt := 0; ; attempt++ {
		replayable, sendErr := send(credential)
		if !replayable || credential.AccessToken == "" || !isTokenRejection(platform, operation, sendErr) {
			return sendErr
		}
		invalidator, ok := manager.(interface {
			Invalidate(context.Context, string) error
		})
		if !ok {
			// External providers may own their complete credential lifecycle.
			return sendErr
		}
		if invalidateErr := invalidator.Invalidate(ctx, credential.AccessToken); invalidateErr != nil {
			return fmt.Errorf("invalidate rejected %s credential: %w", platform, invalidateErr)
		}
		if attempt == 1 {
			return sendErr
		}
		credential, err = manager.Token(ctx)
		if err != nil {
			return fmt.Errorf("refresh rejected %s credential: %w", platform, err)
		}
	}
}

func isTokenRejection(platform, operation string, err error) bool {
	var platformErr *wxerrors.Error
	if !errors.As(err, &platformErr) || platformErr == nil || platformErr.Platform != platform || platformErr.Operation != operation {
		return false
	}
	switch platform {
	case "official":
		return platformErr.Code == "40001" || platformErr.Code == "42001" ||
			operation == "official.oauth.component_token" && platformErr.Code == "40014"
	case "miniapp", "openplatform":
		return platformErr.Code == "40001" || platformErr.Code == "40014" || platformErr.Code == "42001"
	case "mobileapp":
		return operation == "mobileapp.oauth.userinfo" &&
			(platformErr.Code == "40001" || platformErr.Code == "40014" || platformErr.Code == "42001")
	case "work":
		return platformErr.Code == "40014" || platformErr.Code == "42001"
	default:
		return false
	}
}

// TokenQuery copies query values before installing the credential for an attempt.
func TokenQuery(query url.Values, token string) url.Values {
	cp := make(url.Values, len(query)+1)
	for key, values := range query {
		cp[key] = append([]string(nil), values...)
	}
	cp.Set("access_token", token)
	return cp
}

// WriteTracker marks destination writes so downloads are never replayed after
// delivery begins, including when a destination itself returns a platform error.
type WriteTracker struct {
	Destination io.Writer
	Started     bool
}

// Write forwards bytes to the destination and records the attempted write.
func (w *WriteTracker) Write(data []byte) (int, error) {
	w.Started = true
	return w.Destination.Write(data)
}
