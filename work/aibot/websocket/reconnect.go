package websocket

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"net"
	"syscall"

	coderws "github.com/coder/websocket"
	coreerrors "github.com/goairix/wx/v2/core/errors"
)

var ErrRetryExhausted = errors.New("aibot connection retries exhausted")

type ConnectionPhase string

const (
	PhaseDial      ConnectionPhase = "dial"
	PhaseSubscribe ConnectionPhase = "subscribe"
	// PhaseReceive covers the subscribed session, including receive and
	// heartbeat failures or a delivery-ambiguous request terminating it.
	PhaseReceive ConnectionPhase = "receive"
)

// ConnectionError classifies connection and active-session failures, retaining their cause.
// Retryable is advisory: exhausting the configured budget still ends the client.
type ConnectionError struct {
	Phase     ConnectionPhase
	Retryable bool
	Err       error
}

func (e *ConnectionError) Error() string {
	return "aibot connection attempt failed during " + string(e.Phase)
}
func (e *ConnectionError) Unwrap() error { return e.Err }

func connectionError(phase ConnectionPhase, err error) error {
	classified := &ConnectionError{Phase: phase, Retryable: retryableConnectionError(err), Err: err}
	var apiErr *coreerrors.Error
	if errors.As(err, &apiErr) {
		copyError := *apiErr
		copyError.Err = classified
		return &copyError
	}
	operation := "connect"
	if phase == PhaseReceive {
		operation = "receive"
	}
	return operationError(operation, "", classified)
}

func retryableConnectionError(err error) bool {
	if err == nil {
		return false
	}
	var canceledRequest *canceledRequestCause
	hasCanceledRequest := errors.As(err, &canceledRequest)
	if errors.Is(err, context.Canceled) && !hasCanceledRequest {
		return false
	}
	for _, terminal := range []error{ErrClosed, ErrReplaced, ErrNotificationOverflow, ErrProtocol, ErrInvalidArgument} {
		if errors.Is(err, terminal) {
			return false
		}
	}
	permanent, httpTransient, hasStatus, network := false, false, false, hasCanceledRequest
	walkConnectionErrors(err, func(cause error) {
		switch e := cause.(type) {
		case x509.UnknownAuthorityError, *x509.UnknownAuthorityError,
			x509.HostnameError, *x509.HostnameError,
			x509.CertificateInvalidError, *x509.CertificateInvalidError,
			*tls.CertificateVerificationError:
			permanent = true
		case coderws.CloseError:
			if retryableCloseStatus(e.Code) {
				network = true
			} else {
				permanent = true
			}
		case *coderws.CloseError:
			if retryableCloseStatus(e.Code) {
				network = true
			} else {
				permanent = true
			}
		case *net.DNSError:
			if e.IsNotFound {
				permanent = true
			} else if e.IsTemporary {
				network = true
			}
		case *coreerrors.Error:
			if e.Code != "" {
				permanent = true
			}
			if e.HTTPStatus != 0 {
				hasStatus = true
				if e.HTTPStatus == 408 || e.HTTPStatus == 429 || e.HTTPStatus >= 500 && e.HTTPStatus <= 599 {
					httpTransient = true
				} else {
					permanent = true
				}
			}
		}
		if timeout, ok := cause.(net.Error); ok && timeout.Timeout() {
			network = true
		}
	})
	if permanent {
		return false
	}
	if hasStatus {
		return httpTransient
	}
	if network || errors.Is(err, context.DeadlineExceeded) || errors.Is(err, net.ErrClosed) || errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}
	for _, failure := range []error{syscall.ECONNREFUSED, syscall.ECONNRESET, syscall.ECONNABORTED, syscall.ETIMEDOUT, syscall.EPIPE, syscall.ENETUNREACH, syscall.EHOSTUNREACH} {
		if errors.Is(err, failure) {
			return true
		}
	}
	return false
}

func retryableCloseStatus(code coderws.StatusCode) bool {
	switch code {
	case coderws.StatusNormalClosure, coderws.StatusGoingAway,
		coderws.StatusNoStatusRcvd, coderws.StatusAbnormalClosure,
		coderws.StatusInternalError, coderws.StatusServiceRestart,
		coderws.StatusTryAgainLater, coderws.StatusBadGateway:
		return true
	}
	return false
}
func walkConnectionErrors(err error, visit func(error)) {
	if err == nil {
		return
	}
	visit(err)
	switch e := err.(type) {
	case interface{ Unwrap() []error }:
		for _, child := range e.Unwrap() {
			walkConnectionErrors(child, visit)
		}
	case interface{ Unwrap() error }:
		walkConnectionErrors(e.Unwrap(), visit)
	}
}
func validReconnectPolicy(p ReconnectPolicy) bool {
	return p.InitialDelay > 0 && p.MaxDelay >= p.InitialDelay && p.MaxAttempts >= -1
}
