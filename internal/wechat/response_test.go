package wechat

import (
	"encoding/json"
	"errors"
	"testing"

	wxerrors "github.com/goairix/wx/v2/core/errors"
	"github.com/goairix/wx/v2/core/request"
)

type countedResult struct{ calls int }

func (r *countedResult) UnmarshalJSON([]byte) error { r.calls++; return nil }
func TestResponseCodeNormalization(t *testing.T) {
	for _, code := range []string{`40013`, `"40013"`} {
		r := Response{Platform: "official", Operation: "test"}
		err := r.DecodeResponse([]byte(`{"errcode":`+code+`,"errmsg":"  exact message  "}`), request.ResponseMeta{StatusCode: 202, RequestID: "id"})
		var platformErr *wxerrors.Error
		if !errors.As(err, &platformErr) || platformErr.Code != "40013" || platformErr.Message != "  exact message  " || platformErr.HTTPStatus != 202 || platformErr.RequestID != "id" {
			t.Fatalf("code=%s err=%#v", code, err)
		}
	}
}
func TestResponseSuccessDecodesResultOnce(t *testing.T) {
	for _, code := range []string{`0`, `"0"`, `null`, `""`} {
		result := new(countedResult)
		r := Response{Value: result}
		if err := r.DecodeResponse([]byte(`{"errcode":`+code+`}`), request.ResponseMeta{}); err != nil {
			t.Fatalf("code=%s: %v", code, err)
		}
		if result.calls != 1 {
			t.Fatalf("result decoded %d times", result.calls)
		}
	}
}
func TestResponseRejectsMalformedJSONWithNoResult(t *testing.T) {
	r := Response{}
	err := r.DecodeResponse([]byte(`{"errcode":`), request.ResponseMeta{})
	var syntaxErr *json.SyntaxError
	if !errors.As(err, &syntaxErr) {
		t.Fatalf("err=%v", err)
	}
}
