package aibot

import (
	"errors"

	coreerrors "github.com/goairix/wx/v2/core/errors"
)

var (
	ErrClosed               = errors.New("aibot client closed")
	ErrNotReady             = errors.New("aibot client not ready")
	ErrDisconnected         = errors.New("aibot connection lost")
	ErrReplaced             = errors.New("aibot connection replaced")
	ErrNotificationOverflow = errors.New("aibot notification buffer full")
	ErrInvalidArgument      = errors.New("invalid aibot argument")
	ErrProtocol             = errors.New("invalid aibot protocol")
	ErrUnsupported          = errors.New("unsupported aibot operation")
	ErrInvalidTarget        = errors.New("invalid aibot reply target")
	ErrBodyTooLarge         = errors.New("aibot body too large")
)

func WrapError(operation, requestID string, cause error) error {
	if cause == nil {
		return nil
	}
	e := &coreerrors.Error{Platform: "work", Operation: "work.aibot." + operation, RequestID: requestID, Message: "intelligent bot operation failed", Err: cause}
	var inner *coreerrors.Error
	if errors.As(cause, &inner) {
		e.Code = inner.Code
		e.HTTPStatus = inner.HTTPStatus
		if requestID == "" {
			e.RequestID = inner.RequestID
		}
	}
	return e
}
