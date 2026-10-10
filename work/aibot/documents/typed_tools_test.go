package documents

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
)

type schemaCaller struct {
	names  []string
	params [][]byte
}

func (c *schemaCaller) Tools(context.Context) ([]Tool, error) { return nil, nil }
func (c *schemaCaller) CallTool(_ context.Context, name string, args any) (*ToolResult, error) {
	b, err := json.Marshal(args)
	c.names = append(c.names, name)
	c.params = append(c.params, b)
	return &ToolResult{Content: []ContentBlock{}}, err
}
func TestEightToolArgumentsExact(t *testing.T) {
	caller := new(schemaCaller)
	c, _ := NewWithCaller(caller)
	if err := c.Connect(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Close() }()
	ctx := context.Background()
	_, _ = c.CreateDocument(ctx, CreateDocumentRequest{DocType: 3, DocName: "name"})
	_, _ = c.EditDocumentContent(ctx, EditDocumentContentRequest{DocID: "d", Content: "# markdown\n中文", ContentType: 1})
	_, _ = c.AddSheet(ctx, AddSheetRequest{DocID: "d"})
	_, _ = c.GetSheets(ctx, "d")
	_, _ = c.AddFields(ctx, FieldsRequest{DocID: "d", SheetID: "s", Fields: []Field{{FieldTitle: "数量", FieldType: FieldTypeNumber}}})
	_, _ = c.UpdateFields(ctx, FieldsRequest{DocID: "d", SheetID: "s", Fields: []Field{{FieldID: "f", FieldTitle: "新标题", FieldType: FieldTypeText}}})
	_, _ = c.GetFields(ctx, "d", "s")
	_, _ = c.AddRecords(ctx, AddRecordsRequest{DocID: "d", SheetID: "s", Records: []Record{{Values: map[string]any{"数量": 0, "完成": false, "文本": []TextValue{{Type: "text", Text: "content"}}, "日期": "2026-10-10 08:00"}}}})
	want := []string{"create_doc", "edit_doc_content", "smartsheet_add_sheet", "smartsheet_get_sheet", "smartsheet_add_fields", "smartsheet_update_fields", "smartsheet_get_fields", "smartsheet_add_records"}
	if !reflect.DeepEqual(caller.names, want) {
		t.Fatal(caller.names)
	}
	if string(caller.params[0]) != `{"doc_type":3,"doc_name":"name"}` || string(caller.params[2]) != `{"docid":"d"}` {
		t.Fatalf("%s %s", caller.params[0], caller.params[2])
	}
	var records struct {
		Records []Record `json:"records"`
	}
	if err := json.Unmarshal(caller.params[7], &records); err != nil {
		t.Fatal(err)
	}
	if records.Records[0].Values["数量"] != float64(0) || records.Records[0].Values["完成"] != false || records.Records[0].Values["日期"] != "2026-10-10 08:00" {
		t.Fatal(records)
	}
	if _, err := c.CreateDocument(ctx, CreateDocumentRequest{DocType: 4, DocName: "bad"}); err == nil {
		t.Fatal("bad type")
	}
}
func TestAllFieldTypesAndRecordValues(t *testing.T) {
	values := map[string]any{"text": []TextValue{{Type: "text", Text: "t"}}, "user": []UserValue{{UserID: "u"}}, "url": []URLValue{{Type: "url", Link: "https://example.com"}}, "image": []ImageValue{{ImageURL: "https://example.com/i", Width: new(int)}}, "option": []OptionValue{{Text: "o"}}, "location": []LocationValue{{SourceType: 1, ID: "p", Latitude: "1.1", Longitude: "2.2", Title: "place"}}}
	caller := new(schemaCaller)
	c, _ := NewWithCaller(caller)
	_ = c.Connect(context.Background())
	defer func() { _ = c.Close() }()
	_, err := c.AddRecords(context.Background(), AddRecordsRequest{DocID: "d", SheetID: "s", Records: []Record{{Values: values}}})
	if err != nil {
		t.Fatal(err)
	}
	var body map[string]any
	if err = json.Unmarshal(caller.params[0], &body); err != nil {
		t.Fatal(err)
	}
	record := body["records"].([]any)[0].(map[string]any)["values"].(map[string]any)
	if len(record) != 6 {
		t.Fatal(record)
	}
	fields := []string{"FIELD_TYPE_TEXT", "FIELD_TYPE_NUMBER", "FIELD_TYPE_CHECKBOX", "FIELD_TYPE_DATE_TIME", "FIELD_TYPE_IMAGE", "FIELD_TYPE_USER", "FIELD_TYPE_URL", "FIELD_TYPE_SELECT", "FIELD_TYPE_PROGRESS", "FIELD_TYPE_PHONE_NUMBER", "FIELD_TYPE_EMAIL", "FIELD_TYPE_SINGLE_SELECT", "FIELD_TYPE_LOCATION", "FIELD_TYPE_CURRENCY", "FIELD_TYPE_PERCENTAGE", "FIELD_TYPE_BARCODE"}
	for _, kind := range fields {
		_, err = c.AddFields(context.Background(), FieldsRequest{DocID: "d", SheetID: "s", Fields: []Field{{FieldTitle: "title", FieldType: kind}}})
		if err != nil {
			t.Fatal(err)
		}
	}
}
