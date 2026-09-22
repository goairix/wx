package errors

import (
	"context"
	stderrors "errors"
	"testing"
)

func TestWrapfPreservesCause(t *testing.T) {
	err := Wrapf(context.Canceled, "request %s", "failed")
	if err == nil {
		t.Fatal("Wrapf returned nil for a non-nil cause")
	}
	if got, want := err.Error(), "request failed: context canceled"; got != want {
		t.Fatalf("Wrapf error = %q, want %q", got, want)
	}
	if !Is(err, context.Canceled) {
		t.Fatal("errors.Is did not find context.Canceled")
	}
	if got := Unwrap(err); got != context.Canceled {
		t.Fatalf("errors.Unwrap = %v, want context.Canceled", got)
	}
}

func TestAsFindsStructuredError(t *testing.T) {
	structured := &Error{Platform: "official", Operation: "send", Message: "denied"}
	err := Wrap(structured, "request failed")

	var got *Error
	if !As(err, &got) {
		t.Fatal("errors.As did not find *Error")
	}
	if got != structured {
		t.Fatalf("errors.As returned %p, want %p", got, structured)
	}
}

func TestParsePlatformError(t *testing.T) {
	err := ParsePlatformError("official", "token", 401, []byte(`{"errcode":40014,"errmsg":"invalid access token"}`), "rid-1")
	if err.Platform != "official" || err.Operation != "token" || err.HTTPStatus != 401 {
		t.Fatalf("metadata = %#v", err)
	}
	if err.Code != "40014" {
		t.Fatalf("Code = %q, want %q", err.Code, "40014")
	}
	if err.Message != "invalid access token" {
		t.Fatalf("Message = %q, want %q", err.Message, "invalid access token")
	}
	if err.RequestID != "rid-1" {
		t.Fatalf("RequestID = %q, want %q", err.RequestID, "rid-1")
	}

	stringCode := ParsePlatformError("miniapp", "code", 400, []byte(`{"code":"E_BAD","message":"bad request"}`), "rid-2")
	if stringCode.Code != "E_BAD" || stringCode.Message != "bad request" {
		t.Fatalf("string code error = %#v", stringCode)
	}
}

func TestParsePlatformErrorMalformedBody(t *testing.T) {
	err := ParsePlatformError("official", "token", 502, []byte("not-json"), "rid-3")
	if err == nil {
		t.Fatal("ParsePlatformError returned nil for malformed body")
	}
	if err.Platform != "official" || err.Operation != "token" || err.HTTPStatus != 502 || err.RequestID != "rid-3" {
		t.Fatalf("metadata = %#v", err)
	}
}

func TestStructuredErrorString(t *testing.T) {
	err := &Error{Platform: "official", Operation: "token", Message: "invalid access token"}
	if got, want := err.Error(), "official token: invalid access token"; got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
	withoutMessage := &Error{Platform: "official", Operation: "token"}
	if got, want := withoutMessage.Error(), "official token"; got != want {
		t.Fatalf("Error() without message = %q, want %q", got, want)
	}
}

func TestNewAndErrorf(t *testing.T) {
	if got := New("local failure").Error(); got != "local failure" {
		t.Fatalf("New() = %q", got)
	}
	if got := Errorf("item %d", 7).Error(); got != "item 7" {
		t.Fatalf("Errorf() = %q", got)
	}
	if !stderrors.Is(Wrap(context.Canceled, "cancelled"), context.Canceled) {
		t.Fatal("wrapped error is not compatible with standard errors.Is")
	}
}
