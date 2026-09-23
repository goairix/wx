package api

import (
	"encoding/json"
)

type responseEnvelope struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
	value   interface{}
}

func (r *responseEnvelope) UnmarshalJSON(data []byte) error {
	type errorFields struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	var fields errorFields
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	r.ErrCode = fields.ErrCode
	r.ErrMsg = fields.ErrMsg
	if r.value == nil || r.ErrCode != 0 {
		return nil
	}
	return json.Unmarshal(data, r.value)
}
