package errors

import (
	"bytes"
	"encoding/json"
	"fmt"
)

const maxMalformedErrorDetail = 256

// ParsePlatformError converts a platform error response into a structured Error.
// Both errcode/errmsg and code/message response shapes are supported. Numeric
// and string codes are normalized to a string representation.
func ParsePlatformError(platform, operation string, status int, body []byte, requestID string) *Error {
	err := &Error{
		Platform:   platform,
		Operation:  operation,
		HTTPStatus: status,
		RequestID:  requestID,
	}

	var payload struct {
		ErrCode   json.RawMessage `json:"errcode"`
		Code      json.RawMessage `json:"code"`
		ErrMsg    string          `json:"errmsg"`
		Message   string          `json:"message"`
		CommonOut struct {
			RequestID  string          `json:"requestId"`
			ResultCode json.RawMessage `json:"resultCode"`
			ErrMsg     string          `json:"errMsg"`
		} `json:"commonOut"`
	}
	if decodeErr := json.Unmarshal(body, &payload); decodeErr != nil {
		err.Err = fmt.Errorf("parse platform error response: %w", decodeErr)
		detail := decodeErr.Error()
		if len(detail) > maxMalformedErrorDetail {
			detail = detail[:maxMalformedErrorDetail] + "..."
		}
		err.Message = fmt.Sprintf("malformed platform error response (%d bytes): %s", len(body), detail)
		return err
	}

	code := payload.ErrCode
	if len(code) == 0 || string(code) == "null" {
		code = payload.Code
	}
	if len(code) == 0 || string(code) == "null" {
		code = payload.CommonOut.ResultCode
	}
	err.Code = rawValueString(code)
	if payload.ErrMsg != "" {
		err.Message = payload.ErrMsg
	} else if payload.CommonOut.ErrMsg != "" {
		err.Message = payload.CommonOut.ErrMsg
	} else {
		err.Message = payload.Message
	}
	if err.RequestID == "" {
		err.RequestID = payload.CommonOut.RequestID
	}
	return err
}

// ParseResponseError recognizes platform error envelopes returned with a
// successful HTTP status. A nil result means the body does not describe a
// platform failure.
func ParseResponseError(
	platform string,
	operation string,
	status int,
	body []byte,
	requestID string,
) *Error {
	if len(bytes.TrimSpace(body)) == 0 {
		return nil
	}
	var payload struct {
		ErrCode   json.RawMessage `json:"errcode"`
		ErrMsg    string          `json:"errmsg"`
		CommonOut struct {
			RequestID  string          `json:"requestId"`
			ResultCode json.RawMessage `json:"resultCode"`
			ErrMsg     string          `json:"errMsg"`
		} `json:"commonOut"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil
	}

	code := rawValueString(payload.ErrCode)
	message := payload.ErrMsg
	if code == "" {
		code = rawValueString(payload.CommonOut.ResultCode)
		message = payload.CommonOut.ErrMsg
	}
	if code == "" || code == "0" {
		return nil
	}
	if requestID == "" {
		requestID = payload.CommonOut.RequestID
	}
	return &Error{
		Platform:   platform,
		Operation:  operation,
		HTTPStatus: status,
		Code:       code,
		Message:    message,
		RequestID:  requestID,
	}
}

func rawValueString(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var value string
	if raw[0] == '"' {
		if json.Unmarshal(raw, &value) == nil {
			return value
		}
	}
	return string(raw)
}
