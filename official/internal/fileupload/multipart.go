// Package fileupload builds multipart media requests shared by official domains.
package fileupload

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"net/http"
	"path/filepath"
)

// Build encodes a media file and any additional form fields.
func Build(filename string, data []byte, fields map[string]string) ([]byte, http.Header, error) {
	filename = filepath.Base(filename)
	if filename == "" || filename == "." || filename == string(filepath.Separator) || len(data) == 0 {
		return nil, nil, fmt.Errorf("official: upload requires a filename and nonempty media")
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("media", filename)
	if err != nil {
		return nil, nil, err
	}
	if _, err := part.Write(data); err != nil {
		return nil, nil, err
	}
	for name, value := range fields {
		if err := writer.WriteField(name, value); err != nil {
			return nil, nil, err
		}
	}
	if err := writer.Close(); err != nil {
		return nil, nil, err
	}
	return body.Bytes(), http.Header{"Content-Type": {writer.FormDataContentType()}}, nil
}
