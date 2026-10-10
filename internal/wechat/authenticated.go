package wechat

import (
	"context"
	"net/url"

	"github.com/goairix/wx/v2/core/auth"
	wxerrors "github.com/goairix/wx/v2/core/errors"
	"github.com/goairix/wx/v2/core/request"
)

// DoAuthenticated executes JSON, buffered or streamed requests with the supplied
// credential. The parameter selects access_token or component_access_token.
// Plain providers keep their existing one-shot semantics; providers implementing
// Invalidate can recover from a rejected credential once.
func DoAuthenticated(ctx context.Context, provider auth.Provider, caller request.Caller, req request.Request, parameter string) error {
	return DoWithToken(ctx, provider, req.Platform, req.Operation, func(credential auth.Credential) (bool, error) {
		attempt := req
		attempt.Query = make(url.Values, len(req.Query)+1)
		for key, values := range req.Query {
			attempt.Query[key] = append([]string(nil), values...)
		}
		attempt.Query.Set(parameter, credential.AccessToken)
		attempt.Header = req.Header.Clone()
		meta := new(request.ResponseMeta)
		attempt.Meta = meta
		if req.Meta != nil {
			defer func() { *req.Meta = *meta }()
		}
		var tracker *WriteTracker
		var envelope *Response
		data, buffered := req.Result.(*[]byte)
		if req.ResponseWriter != nil {
			tracker = &WriteTracker{Destination: req.ResponseWriter}
			attempt.ResponseWriter = tracker
		} else if buffered {
			*data = nil
		} else {
			envelope = &Response{Platform: req.Platform, Operation: req.Operation, Value: req.Result}
			attempt.Result = envelope
		}
		err := caller.Do(ctx, attempt)
		if err == nil {
			if buffered {
				if platformErr := wxerrors.ParseResponseError(req.Platform, req.Operation, meta.StatusCode, *data, meta.RequestID); platformErr != nil {
					err = platformErr
				}
			} else if envelope != nil {
				err = envelope.Error(*meta)
			}
		}
		return tracker == nil || !tracker.Started, err
	})
}
