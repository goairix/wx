package documents

import (
	"context"

	"github.com/goairix/wx/v2/work/aibot"
)

const (
	FieldTypeText         = "FIELD_TYPE_TEXT"
	FieldTypeNumber       = "FIELD_TYPE_NUMBER"
	FieldTypeCheckbox     = "FIELD_TYPE_CHECKBOX"
	FieldTypeDateTime     = "FIELD_TYPE_DATE_TIME"
	FieldTypeImage        = "FIELD_TYPE_IMAGE"
	FieldTypeUser         = "FIELD_TYPE_USER"
	FieldTypeURL          = "FIELD_TYPE_URL"
	FieldTypeSelect       = "FIELD_TYPE_SELECT"
	FieldTypeProgress     = "FIELD_TYPE_PROGRESS"
	FieldTypePhoneNumber  = "FIELD_TYPE_PHONE_NUMBER"
	FieldTypeEmail        = "FIELD_TYPE_EMAIL"
	FieldTypeSingleSelect = "FIELD_TYPE_SINGLE_SELECT"
	FieldTypeLocation     = "FIELD_TYPE_LOCATION"
	FieldTypeCurrency     = "FIELD_TYPE_CURRENCY"
	FieldTypePercentage   = "FIELD_TYPE_PERCENTAGE"
	FieldTypeBarcode      = "FIELD_TYPE_BARCODE"
)

type Field struct {
	FieldID    string `json:"field_id,omitempty"`
	FieldTitle string `json:"field_title"`
	FieldType  string `json:"field_type"`
}
type FieldsRequest struct {
	DocID   string  `json:"docid"`
	SheetID string  `json:"sheet_id"`
	Fields  []Field `json:"fields"`
}

func (c *Client) AddFields(ctx context.Context, r FieldsRequest) (*ToolResult, error) {
	return c.fields(ctx, "smartsheet_add_fields", r, false)
}

// UpdateFields changes titles only. Supply the existing FieldType from GetFields.
func (c *Client) UpdateFields(ctx context.Context, r FieldsRequest) (*ToolResult, error) {
	return c.fields(ctx, "smartsheet_update_fields", r, true)
}
func (c *Client) fields(ctx context.Context, name string, r FieldsRequest, update bool) (*ToolResult, error) {
	if r.DocID == "" || r.SheetID == "" || len(r.Fields) == 0 {
		return nil, aibot.WrapError("documents."+name, "", aibot.ErrInvalidArgument)
	}
	for _, f := range r.Fields {
		if f.FieldTitle == "" || (update && f.FieldID == "") {
			return nil, aibot.WrapError("documents."+name, "", aibot.ErrInvalidArgument)
		}
		switch f.FieldType {
		case FieldTypeText, FieldTypeNumber, FieldTypeCheckbox, FieldTypeDateTime, FieldTypeImage, FieldTypeUser, FieldTypeURL, FieldTypeSelect, FieldTypeProgress, FieldTypePhoneNumber, FieldTypeEmail, FieldTypeSingleSelect, FieldTypeLocation, FieldTypeCurrency, FieldTypePercentage, FieldTypeBarcode:
		default:
			return nil, aibot.WrapError("documents."+name, "", aibot.ErrInvalidArgument)
		}
	}
	return c.CallTool(ctx, name, r)
}
func (c *Client) GetFields(ctx context.Context, docID, sheetID string) (*ToolResult, error) {
	if docID == "" || sheetID == "" {
		return nil, aibot.WrapError("documents.smartsheet_get_fields", "", aibot.ErrInvalidArgument)
	}
	return c.CallTool(ctx, "smartsheet_get_fields", struct {
		DocID   string `json:"docid"`
		SheetID string `json:"sheet_id"`
	}{docID, sheetID})
}
