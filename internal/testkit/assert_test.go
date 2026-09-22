package testkit

import (
	"net/http"
	"strings"
	"testing"
)

func TestAssertJSONBodyMatchesObjects(t *testing.T) {
	request, err := http.NewRequest(http.MethodPost, "http://example.test", strings.NewReader(`{"name":"Ada","roles":["admin","writer"],"active":true}`))
	if err != nil {
		t.Fatal(err)
	}
	AssertJSONBody(t, request, map[string]interface{}{
		"active": true,
		"roles":  []interface{}{"admin", "writer"},
		"name":   "Ada",
	})
}

func TestAssertJSONBodyAcceptsRecordedRequest(t *testing.T) {
	AssertJSONBody(t, RecordedRequest{Body: []byte(`{"count":1}`)}, struct {
		Count int `json:"count"`
	}{Count: 1})
}
