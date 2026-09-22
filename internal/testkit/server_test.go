package testkit

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestNewServerRecordsRequests(t *testing.T) {
	server, recorder := NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	request, err := http.NewRequest(http.MethodPost, server.URL+"/v1/users?id=1", strings.NewReader(`{"name":"Ada"}`))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("X-Test", "yes")
	response, err := server.Client().Do(request)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.Copy(io.Discard, response.Body)
	_ = response.Body.Close()

	if recorder.Len() != 1 {
		t.Fatalf("recorded requests = %d, want 1", recorder.Len())
	}
	got, ok := recorder.Last()
	if !ok {
		t.Fatal("Last() returned no request")
	}
	if got.Method != http.MethodPost || got.URL != "/v1/users?id=1" || string(got.Body) != `{"name":"Ada"}` {
		t.Fatalf("unexpected request: %+v", got)
	}
	if got.Header.Get("X-Test") != "yes" {
		t.Fatalf("recorded headers = %v", got.Header)
	}
}

func TestNewServerRestoresBodyForHandler(t *testing.T) {
	server, _ := NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
			return
		}
		if string(body) != `{"ok":true}` {
			t.Errorf("handler body = %q", body)
		}
	}))
	defer server.Close()

	response, err := server.Client().Post(server.URL, "application/json", strings.NewReader(`{"ok":true}`))
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
}
