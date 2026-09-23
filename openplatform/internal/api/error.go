// Package api contains shared Open Platform response helpers.
package api

import (
	"strconv"

	wxerrors "github.com/goairix/wx/v2/core/errors"
	"github.com/goairix/wx/v2/core/request"
)

// ErrorFields contains the error fields returned by WeChat APIs.
type ErrorFields struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
}

// Error converts nonzero API error fields into a structured error.
func Error(operation string, fields ErrorFields, meta *request.ResponseMeta) error {
	if fields.ErrCode == 0 {
		return nil
	}

	structured := &wxerrors.Error{
		Platform:  "openplatform",
		Operation: operation,
		Code:      strconv.Itoa(fields.ErrCode),
		Message:   fields.ErrMsg,
	}
	if meta != nil {
		structured.HTTPStatus = meta.StatusCode
		structured.RequestID = meta.RequestID
	}
	return structured
}
