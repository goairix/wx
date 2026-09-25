// Package wechat decodes the shared WeChat JSON response envelope.
package wechat

import (
	"encoding/json"

	wxerrors "github.com/goairix/wx/v2/core/errors"
	"github.com/goairix/wx/v2/core/request"
)

// Response combines platform error decoding with a domain result.
type Response struct {
	Platform  string
	Operation string
	Value     interface{}
	code      string
	message   string
}

// UnmarshalJSON supports callers that decode request results directly.
func (r *Response) UnmarshalJSON(body []byte) error {
	var fields struct {
		ErrCode json.RawMessage `json:"errcode"`
		ErrMsg  string          `json:"errmsg"`
	}
	if err := json.Unmarshal(body, &fields); err != nil {
		return err
	}
	r.code = string(fields.ErrCode)
	if r.code == "null" {
		r.code = ""
	}
	if len(fields.ErrCode) > 0 && fields.ErrCode[0] == '"' {
		if err := json.Unmarshal(fields.ErrCode, &r.code); err != nil {
			return err
		}
	}
	r.message = fields.ErrMsg
	if r.code != "" && r.code != "0" || r.Value == nil {
		return nil
	}
	return json.Unmarshal(body, r.Value)
}

// DecodeResponse reports platform errors before transport observation finishes.
func (r *Response) DecodeResponse(body []byte, meta request.ResponseMeta) error {
	if err := r.UnmarshalJSON(body); err != nil {
		return err
	}
	return r.Error(meta)
}

// Error attaches actual response metadata, including for custom callers.
func (r *Response) Error(meta request.ResponseMeta) error {
	if r.code == "" || r.code == "0" {
		return nil
	}
	return &wxerrors.Error{
		Platform:   r.Platform,
		Operation:  r.Operation,
		HTTPStatus: meta.StatusCode,
		Code:       r.code,
		Message:    r.message,
		RequestID:  meta.RequestID,
	}
}
