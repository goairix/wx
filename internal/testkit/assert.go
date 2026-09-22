package testkit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"testing"
)

// AssertJSONBody decodes source as JSON and compares it with want. Both values
// are normalized through JSON, so callers may use maps, structs, or slices
// without depending on map iteration order.
func AssertJSONBody(t testing.TB, source interface{}, want interface{}) {
	t.Helper()
	body, err := jsonSourceBytes(source)
	if err != nil {
		t.Fatalf("read JSON body: %v", err)
	}
	gotValue, err := decodeJSON(body)
	if err != nil {
		t.Fatalf("decode JSON body: %v", err)
	}
	wantBytes, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("encode expected JSON body: %v", err)
	}
	wantValue, err := decodeJSON(wantBytes)
	if err != nil {
		t.Fatalf("decode expected JSON body: %v", err)
	}
	if !reflect.DeepEqual(gotValue, wantValue) {
		t.Fatalf("JSON body mismatch: got %s, want %s", body, wantBytes)
	}
}

func jsonSourceBytes(source interface{}) ([]byte, error) {
	switch value := source.(type) {
	case *http.Request:
		if value == nil || value.Body == nil {
			return nil, fmt.Errorf("request body is nil")
		}
		body, err := io.ReadAll(value.Body)
		if err == nil {
			_ = value.Body.Close()
			value.Body = io.NopCloser(bytes.NewReader(body))
		}
		return body, err
	case RecordedRequest:
		return value.Body, nil
	case []byte:
		return value, nil
	case string:
		return []byte(value), nil
	case io.Reader:
		return io.ReadAll(value)
	default:
		return nil, fmt.Errorf("unsupported JSON source %T", source)
	}
}

func decodeJSON(body []byte) (interface{}, error) {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	var value interface{}
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	var extra interface{}
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("multiple JSON values")
		}
		return nil, err
	}
	return value, nil
}
