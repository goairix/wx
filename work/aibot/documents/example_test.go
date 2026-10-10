package documents_test

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/goairix/wx/v2/work/aibot/documents"
)

// This example caller represents a project-owned connected MCP session.
// Tool JSON below is controlled example data, not a promised platform response schema.
type exampleCaller struct{}

func (exampleCaller) Tools(context.Context) ([]documents.Tool, error) { return nil, nil }
func (exampleCaller) CallTool(_ context.Context, name string, args any) (*documents.ToolResult, error) {
	raw := json.RawMessage(`{}`)
	switch name {
	case "create_doc":
		raw = json.RawMessage(`{"docid":"doc","url":"https://example.com/doc"}`)
	case "smartsheet_get_sheet":
		raw = json.RawMessage(`{"sheets":[{"sheet_id":"sheet"}]}`)
	case "smartsheet_get_fields":
		raw = json.RawMessage(`{"fields":[{"field_id":"field","field_title":"默认","field_type":"FIELD_TYPE_TEXT"}]}`)
	}
	return &documents.ToolResult{Content: []documents.ContentBlock{}, StructuredContent: raw}, nil
}
func ExampleClient_documentWorkflow() {
	client, err := documents.NewWithCaller(exampleCaller{})
	if err != nil {
		return
	}
	ctx := context.Background()
	if err = client.Connect(ctx); err != nil {
		return
	}
	defer func() { _ = client.Close() }()
	created, err := client.CreateDocument(ctx, documents.CreateDocumentRequest{DocType: 10, DocName: "项目进度"})
	if err != nil {
		return
	}
	var doc struct {
		DocID string `json:"docid"`
		URL   string `json:"url"`
	}
	if err = json.Unmarshal(created.StructuredContent, &doc); err != nil {
		return
	}
	// Persist doc.DocID and doc.URL using the application's actual result parser.
	sheets, err := client.GetSheets(ctx, doc.DocID)
	if err != nil {
		return
	}
	var sheetList struct {
		Sheets []struct {
			SheetID string `json:"sheet_id"`
		} `json:"sheets"`
	}
	if err = json.Unmarshal(sheets.StructuredContent, &sheetList); err != nil || len(sheetList.Sheets) == 0 {
		return
	}
	sheetID := sheetList.Sheets[0].SheetID
	fields, err := client.GetFields(ctx, doc.DocID, sheetID)
	if err != nil {
		return
	}
	var fieldList struct {
		Fields []documents.Field `json:"fields"`
	}
	if err = json.Unmarshal(fields.StructuredContent, &fieldList); err != nil || len(fieldList.Fields) == 0 {
		return
	}
	field := fieldList.Fields[0]
	field.FieldTitle = "任务"
	if _, err = client.UpdateFields(ctx, documents.FieldsRequest{DocID: doc.DocID, SheetID: sheetID, Fields: []documents.Field{field}}); err != nil {
		return
	}
	if _, err = client.AddFields(ctx, documents.FieldsRequest{DocID: doc.DocID, SheetID: sheetID, Fields: []documents.Field{{FieldTitle: "完成", FieldType: documents.FieldTypeCheckbox}}}); err != nil {
		return
	}
	if _, err = client.AddRecords(ctx, documents.AddRecordsRequest{DocID: doc.DocID, SheetID: sheetID, Records: []documents.Record{{Values: map[string]any{"任务": []documents.TextValue{{Type: "text", Text: "发布"}}, "完成": false}}}}); err != nil {
		return
	}
	fmt.Println("workflow completed")
	// Output: workflow completed
}
