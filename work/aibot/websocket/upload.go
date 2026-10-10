package websocket

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"unicode/utf8"

	"github.com/goairix/wx/v2/work/aibot"
)

type UploadInitRequest struct {
	Type        string `json:"type"`
	Filename    string `json:"filename"`
	TotalSize   int64  `json:"total_size"`
	TotalChunks int    `json:"total_chunks"`
	MD5         string `json:"md5"`
}
type UploadInitResult struct {
	UploadID string `json:"upload_id"`
}
type UploadChunkRequest struct {
	UploadID   string `json:"upload_id"`
	ChunkIndex int    `json:"chunk_index"`
	Base64Data string `json:"base64_data"`
}
type UploadFinishRequest struct {
	UploadID string `json:"upload_id"`
}

func (c *Client) InitUpload(ctx context.Context, r UploadInitRequest) (*UploadInitResult, error) {
	limits, limitsErr := aibot.UploadMediaLimits(r.Type)
	if limitsErr != nil || r.Filename == "" || len(r.Filename) > 256 || !utf8.ValidString(r.Filename) || r.TotalSize < limits.MinBytes || r.TotalSize > min(limits.MaxBytes, c.options.uploadLimit) || r.TotalChunks < 1 || r.TotalChunks > 100 || r.TotalSize > int64(r.TotalChunks)*(512<<10) || len(r.MD5) != 32 {
		return nil, operationError(CmdUploadInit, "", ErrInvalidArgument)
	}
	if _, err := hex.DecodeString(r.MD5); err != nil {
		return nil, operationError(CmdUploadInit, "", ErrInvalidArgument)
	}
	f, err := c.request(ctx, CmdUploadInit, "", r)
	if err != nil {
		return nil, err
	}
	result := new(UploadInitResult)
	if err = json.Unmarshal(f.Body, result); err != nil {
		return nil, operationError(CmdUploadInit, f.Headers.ReqID, errors.Join(ErrProtocol, err))
	}
	if result.UploadID == "" {
		return nil, operationError(CmdUploadInit, f.Headers.ReqID, ErrProtocol)
	}
	return result, nil
}
func (c *Client) UploadChunk(ctx context.Context, r UploadChunkRequest) error {
	if r.UploadID == "" || r.ChunkIndex < 0 || r.ChunkIndex >= 100 || len(r.Base64Data) > base64.StdEncoding.EncodedLen(512<<10) {
		return operationError(CmdUploadChunk, "", ErrInvalidArgument)
	}
	b, err := base64.StdEncoding.Strict().DecodeString(r.Base64Data)
	if err != nil || len(b) < 1 || len(b) > 512<<10 {
		return operationError(CmdUploadChunk, "", ErrInvalidArgument)
	}
	_, err = c.request(ctx, CmdUploadChunk, "", r)
	return err
}
func (c *Client) FinishUpload(ctx context.Context, r UploadFinishRequest) (*UploadResult, error) {
	if r.UploadID == "" {
		return nil, operationError(CmdUploadFinish, "", ErrInvalidArgument)
	}
	f, err := c.request(ctx, CmdUploadFinish, "", r)
	if err != nil {
		return nil, err
	}
	result := new(UploadResult)
	if err = json.Unmarshal(f.Body, result); err != nil {
		return nil, operationError(CmdUploadFinish, f.Headers.ReqID, errors.Join(ErrProtocol, err))
	}
	result.reqID = f.Headers.ReqID
	if result.MediaID == "" || result.CreatedAt == "" {
		return nil, operationError(CmdUploadFinish, f.Headers.ReqID, ErrProtocol)
	}
	return result, nil
}
func (c *Client) UploadMedia(ctx context.Context, mediaType, filename string, data io.Reader) (*UploadResult, error) {
	result, err := c.uploadMedia(ctx, mediaType, filename, data)
	if err != nil {
		return nil, err
	}
	return &result, nil
}
func (c *Client) Download(ctx context.Context, source aibot.MediaSource) (*aibot.DownloadedMedia, error) {
	media, err := aibot.NewMedia(aibot.WithHTTPClient(c.options.httpClient), aibot.WithLogger(c.options.logger), aibot.WithRequestTimeout(c.options.requestTimeout), aibot.WithDownloadLimit(c.options.downloadLimit))
	if err != nil {
		return nil, err
	}
	return media.Download(ctx, source)
}
