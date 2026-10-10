package integration

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"testing"

	wxerrors "github.com/goairix/wx/v2/core/errors"
	"github.com/goairix/wx/v2/official"
	"github.com/goairix/wx/v2/official/customer"
	"github.com/goairix/wx/v2/official/material"
	"github.com/goairix/wx/v2/work"
	"github.com/goairix/wx/v2/work/message"
)

type recoveryContextKey struct{}

// Exercise real domain clients, authentication, transport and decoding. Wrong
// token classification, lost upload bytes, or replayed error bodies must fail.
func TestTokenRecoveryAcrossDomains(t *testing.T) {
	for _, platform := range []string{"official", "work"} {
		codes := []string{"40001", "42001"}
		if platform == "work" {
			codes = []string{"40014", "42001"}
		}
		for _, mode := range []string{"message", "upload", "download", "stream", "delete"} {
			for _, code := range codes {
				t.Run(platform+"/"+mode+"/"+code, func(t *testing.T) {
					ctx := context.WithValue(context.Background(), recoveryContextKey{}, "lease-and-version")
					tokens, attempts := 0, 0
					var firstBody []byte
					var firstContentType string
					h := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
						if r.Context() != ctx {
							t.Error("original request context was replaced")
						}
						if r.URL.Path == "/cgi-bin/token" || r.URL.Path == "/cgi-bin/gettoken" {
							tokens++
							return recoveryResponse("application/json", "token", fmt.Sprintf(`{"access_token":"token-%d","expires_in":7200}`, tokens)), nil
						}
						attempts++
						if got := r.URL.Query().Get("access_token"); got != fmt.Sprintf("token-%d", attempts) {
							t.Errorf("token = %s at attempt %d", got, attempts)
						}
						if platform == "official" && r.Header.Get("Authorization") != fmt.Sprintf("Bearer token-%d", attempts) {
							t.Error("stale Authorization header")
						}
						body, err := io.ReadAll(r.Body)
						if err != nil {
							t.Error(err)
						}
						if attempts == 1 {
							firstBody = body
							firstContentType = r.Header.Get("Content-Type")
						} else if !bytes.Equal(firstBody, body) || firstContentType != r.Header.Get("Content-Type") {
							t.Error("retry changed request body or Content-Type")
						}
						if mode == "upload" {
							_, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
							if err != nil {
								t.Error(err)
								return nil, err
							}
							part, err := multipart.NewReader(bytes.NewReader(body), params["boundary"]).NextPart()
							if err != nil {
								t.Error(err)
								return nil, err
							}
							data, err := io.ReadAll(part)
							if err != nil {
								t.Error(err)
							}
							if part.FormName() != "media" || part.FileName() != "file.png" || string(data) != "PNG-file" || r.URL.Query().Get("type") != "image" {
								t.Error("invalid multipart upload")
							}
						}
						if attempts == 1 {
							return recoveryResponse("text/plain", "rejected", `{"errcode":"`+code+`","errmsg":"rejected token"}`), nil
						}
						if mode == "download" || mode == "stream" {
							return recoveryResponse("image/png", "success", "PNG-file"), nil
						}
						return recoveryResponse("application/json", "success", `{"errcode":0,"media_id":"new-media","msgid":"new-message"}`), nil
					})}
					call := recoveryDomain(t, platform, mode, h)
					if err := call(ctx); err != nil {
						t.Fatal(err)
					}
					if tokens != 2 || attempts != 2 {
						t.Fatalf("token fetches=%d attempts=%d, want 2/2", tokens, attempts)
					}
				})
			}
		}
	}
}

func recoveryResponse(contentType, requestID, body string) *http.Response {
	return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {contentType}, "X-Request-Id": {requestID}}, Body: io.NopCloser(bytes.NewBufferString(body))}
}

func recoveryDomain(t *testing.T, platform, mode string, h *http.Client) func(context.Context) error {
	t.Helper()
	if platform == "official" {
		c, err := official.NewClient(official.Config{AppID: "app", AppSecret: "secret"}, official.WithHTTPClient(h))
		if err != nil {
			t.Fatal(err)
		}
		return func(ctx context.Context) error {
			switch mode {
			case "message":
				return c.CustomerService().Send(ctx, customer.Message{ToUser: "user", MsgType: customer.MessageText, Text: &customer.Text{Content: "hello"}})
			case "upload":
				v, err := c.Materials().UploadTemporary(ctx, material.TypeImage, "file.png", []byte("PNG-file"))
				if err == nil && v.MediaID != "new-media" {
					t.Errorf("upload result = %#v", v)
				}
				return err
			case "download":
				v, err := c.Materials().Get(ctx, "media")
				if err == nil && (string(v.Data) != "PNG-file" || v.ContentType != "image/png") {
					t.Errorf("download result = %#v", v)
				}
				return err
			case "stream":
				var b bytes.Buffer
				typ, err := c.Materials().DownloadTemporaryTo(ctx, "media", &b)
				if err == nil && (b.String() != "PNG-file" || typ != "image/png") {
					t.Errorf("stream = %q %s", b.String(), typ)
				}
				return err
			case "delete":
				return c.Menu().Delete(ctx)
			default:
				t.Fatalf("unknown mode %s", mode)
				return nil
			}
		}
	}
	c, err := work.NewClient(work.Config{CorpID: "corp", CorpSecret: "secret", AgentID: 100}, work.WithHTTPClient(h))
	if err != nil {
		t.Fatal(err)
	}
	return func(ctx context.Context) error {
		switch mode {
		case "message":
			v, err := c.Message().Send(ctx, message.SendOption{ToUser: "user"}, &message.Text{Content: "hello"})
			if err == nil && v.MsgId != "new-message" {
				t.Errorf("message result = %#v", v)
			}
			return err
		case "upload":
			v, err := c.Media().Upload(ctx, "image", "file.png", []byte("PNG-file"))
			if err == nil && v.MediaId != "new-media" {
				t.Errorf("upload result = %#v", v)
			}
			return err
		case "download":
			data, typ, err := c.Media().Download(ctx, "media")
			if err == nil && (string(data) != "PNG-file" || typ != "image/png") {
				t.Errorf("download = %q %s", data, typ)
			}
			return err
		case "stream":
			var b bytes.Buffer
			typ, err := c.Media().DownloadTo(ctx, "media", &b)
			if err == nil && (b.String() != "PNG-file" || typ != "image/png") {
				t.Errorf("stream = %q %s", b.String(), typ)
			}
			return err
		case "delete":
			return c.Agent().DeleteMenu(ctx, 100)
		default:
			t.Fatalf("unknown mode %s", mode)
			return nil
		}
	}
}

func TestTokenRecoveryStopsAfterOneReplay(t *testing.T) {
	for _, platform := range []string{"official", "work"} {
		for _, mode := range []string{"message", "upload", "download", "stream"} {
			t.Run(platform+"/"+mode, func(t *testing.T) {
				tokens, attempts := 0, 0
				h := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
					if r.URL.Path == "/cgi-bin/token" || r.URL.Path == "/cgi-bin/gettoken" {
						tokens++
						return recoveryResponse("application/json", "token", fmt.Sprintf(`{"access_token":"token-%d","expires_in":7200}`, tokens)), nil
					}
					attempts++
					return recoveryResponse("text/plain", fmt.Sprintf("business-%d", attempts), `{"errcode":42001,"errmsg":"still expired"}`), nil
				})}
				err := recoveryDomain(t, platform, mode, h)(context.Background())
				var pe *wxerrors.Error
				if !errors.As(err, &pe) || pe.Code != "42001" || pe.RequestID != "business-2" || attempts != 2 || tokens != 2 {
					t.Fatalf("tokens=%d attempts=%d err=%+v", tokens, attempts, err)
				}
			})
		}
	}
}

func TestTokenRecoveryIgnoresUnrelatedErrors(t *testing.T) {
	for _, tc := range []struct{ platform, code string }{{"work", "40001"}, {"work", "45009"}, {"official", "40014"}, {"official", "40013"}} {
		for _, mode := range []string{"message", "upload", "download", "stream"} {
			t.Run(tc.platform+"/"+tc.code+"/"+mode, func(t *testing.T) {
				tokens, attempts := 0, 0
				h := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
					if r.URL.Path == "/cgi-bin/token" || r.URL.Path == "/cgi-bin/gettoken" {
						tokens++
						return recoveryResponse("application/json", "token", `{"access_token":"token","expires_in":7200}`), nil
					}
					attempts++
					return recoveryResponse("text/plain", "original", `{"errcode":"`+tc.code+`","errmsg":"do not retry"}`), nil
				})}
				err := recoveryDomain(t, tc.platform, mode, h)(context.Background())
				var pe *wxerrors.Error
				if !errors.As(err, &pe) || pe.Code != tc.code || pe.RequestID != "original" || attempts != 1 || tokens != 1 {
					t.Fatalf("tokens=%d attempts=%d err=%+v", tokens, attempts, err)
				}
			})
		}
	}
}

type recoveryErrorWriter struct {
	written int
	err     error
}

func (w recoveryErrorWriter) Write([]byte) (int, error) { return w.written, w.err }

func TestTokenRecoveryDoesNotReplayDestinationErrors(t *testing.T) {
	for _, platform := range []string{"official", "work"} {
		for _, written := range []int{0, 1} {
			t.Run(fmt.Sprintf("%s/written-%d", platform, written), func(t *testing.T) {
				tokens, requests := 0, 0
				h := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
					if r.URL.Path == "/cgi-bin/token" || r.URL.Path == "/cgi-bin/gettoken" {
						tokens++
						return recoveryResponse("application/json", "token", `{"access_token":"token","expires_in":7200}`), nil
					}
					requests++
					return recoveryResponse("image/png", "binary", "PNG-file"), nil
				})}
				operation := "work.media.download"
				if platform == "official" {
					operation = "official.material.temporary.get"
				}
				failure := &wxerrors.Error{Platform: platform, Operation: operation, Code: "42001", Message: "destination rejected write"}
				writer := recoveryErrorWriter{written: written, err: failure}
				ctx := context.Background()
				var err error
				if platform == "official" {
					c, createErr := official.NewClient(official.Config{AppID: "app", AppSecret: "secret"}, official.WithHTTPClient(h))
					if createErr != nil {
						t.Fatal(createErr)
					}
					_, err = c.Materials().DownloadTemporaryTo(ctx, "media", writer)
				} else {
					c, createErr := work.NewClient(work.Config{CorpID: "corp", CorpSecret: "secret"}, work.WithHTTPClient(h))
					if createErr != nil {
						t.Fatal(createErr)
					}
					_, err = c.Media().DownloadTo(ctx, "media", writer)
				}
				if !errors.Is(err, failure) || tokens != 1 || requests != 1 {
					t.Fatalf("tokens=%d requests=%d err=%v", tokens, requests, err)
				}
			})
		}
	}
}
