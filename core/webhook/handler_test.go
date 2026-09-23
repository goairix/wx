package webhook

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandlerPassesRequestContextAndParsesXML(t *testing.T) {
	type contextKey string
	key := contextKey("request-context")
	req := httptest.NewRequest(
		http.MethodPost,
		"/callback",
		strings.NewReader(
			`<xml><MsgType>text</MsgType><Content>Hello</Content></xml>`,
		),
	).WithContext(context.WithValue(context.Background(), key, "ok"))
	req.Header.Set("Content-Type", "application/xml")
	called := false
	handler := NewHandler(HandlerFunc(func(
		ctx context.Context,
		payload Payload,
	) (Response, error) {
		called = true
		if ctx.Value(key) != "ok" ||
			payload.Format != "xml" ||
			payload.Values["MsgType"] != "text" ||
			payload.Values["Content"] != "Hello" {
			t.Errorf(
				"unexpected context/payload: context=%v payload=%+v",
				ctx.Value(key),
				payload,
			)
		}
		return EmptyResponse(), nil
	}))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if !called || rr.Code != http.StatusOK {
		t.Fatalf("called=%v status=%d", called, rr.Code)
	}
}

func TestHandlerParsesJSONAndWritesResponse(t *testing.T) {
	req := httptest.NewRequest(
		http.MethodPost,
		"/callback",
		strings.NewReader(`{"event":"subscribe","count":2}`),
	)
	resp := Response{
		Status: http.StatusAccepted,
		Header: http.Header{"X-Webhook": []string{"yes"}},
		Body:   []byte("accepted"),
	}
	handler := NewHandler(HandlerFunc(func(
		ctx context.Context,
		payload Payload,
	) (Response, error) {
		if payload.Format != "json" ||
			payload.Values["event"] != "subscribe" ||
			payload.Values["count"] != "2" {
			t.Errorf("unexpected payload: %+v", payload)
		}
		return resp, nil
	}))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusAccepted ||
		rr.Header().Get("X-Webhook") != "yes" ||
		rr.Body.String() != "accepted" {
		t.Fatalf("response: %+v", rr)
	}
}

func TestHandlerUsesInjectedErrorResponsePolicy(t *testing.T) {
	policyCalls := 0
	handler := NewHandler(HandlerFunc(func(context.Context, Payload) (Response, error) {
		return Response{}, context.Canceled
	}), WithErrorResponse(func(err error) Response {
		policyCalls++
		if err != context.Canceled {
			t.Errorf("error=%v", err)
		}
		return Response{Status: http.StatusServiceUnavailable, Body: []byte("retry")}
	}))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(
		rr,
		httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`<xml/>`)),
	)
	if policyCalls != 1 ||
		rr.Code != http.StatusServiceUnavailable ||
		rr.Body.String() != "retry" {
		t.Fatalf("calls=%d response=%+v", policyCalls, rr)
	}
}

func TestHandlerRejectsMalformedPayloadThroughPolicy(t *testing.T) {
	for _, body := range []string{`<xml><bad>`, `{"bad":`} {
		t.Run(body, func(t *testing.T) {
			called := false
			handler := NewHandler(
				HandlerFunc(func(context.Context, Payload) (Response, error) {
					called = true
					return EmptyResponse(), nil
				}),
				WithErrorResponse(func(error) Response {
					return Response{Status: http.StatusBadRequest}
				}),
			)
			rr := httptest.NewRecorder()
			handler.ServeHTTP(
				rr,
				httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body)),
			)
			if called || rr.Code != http.StatusBadRequest {
				t.Fatalf("called=%v status=%d", called, rr.Code)
			}
		})
	}
}
