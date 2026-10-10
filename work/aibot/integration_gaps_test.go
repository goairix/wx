package aibot

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCallbackPreservesMediaFilenames(t *testing.T) {
	cb, err := ParseCallback([]byte(`{"msgtype":"file","file":{"url":"https://example.invalid/file","filename":"报告.txt"},"quote":{"msgtype":"file","file":{"filename":"quoted.txt"}},"mixed":{"msg_item":[{"msgtype":"image","image":{"filename":"mixed.png"}}]},"future":42}`))
	if err != nil {
		t.Fatal(err)
	}
	if cb.Message.File.Filename != "报告.txt" || cb.Message.Quote.File.Filename != "quoted.txt" || cb.Message.Mixed.MsgItem[0].Image.Filename != "mixed.png" || !bytes.Contains(cb.Raw, []byte(`"future":42`)) {
		t.Fatalf("lost media filenames: %+v", cb)
	}
}

func TestDownloadSeparatesHeaderFilename(t *testing.T) {
	for _, tc := range []struct{ header, path, want, headerName string }{
		{`attachment; filename="../report.txt"`, "/url.txt", "report.txt", "report.txt"},
		{`attachment; filename*=UTF-8''%E6%8A%A5%E5%91%8A.txt`, "/url.txt", "报告.txt", "报告.txt"},
		{`attachment; filename=".."`, "/url.txt", "url.txt", ""},
		{"invalid", "/url.txt", "url.txt", ""},
		{"", "/", "media", ""},
	} {
		t.Run(tc.want+tc.header, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Disposition", tc.header)
				_, _ = w.Write([]byte("payload"))
			}))
			defer server.Close()
			c, err := NewMedia(WithHTTPClient(server.Client()))
			if err != nil {
				t.Fatal(err)
			}
			media, err := c.DownloadRaw(context.Background(), server.URL+tc.path)
			if err != nil {
				t.Fatal(err)
			}
			if media.Filename != tc.want || media.HeaderFilename != tc.headerName {
				t.Fatalf("%+v", media)
			}
		})
	}
}

func encodedUploadImage(t *testing.T, kind string) []byte {
	t.Helper()
	var b bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	var err error
	switch kind {
	case "png":
		err = png.Encode(&b, img)
	case "jpeg":
		err = jpeg.Encode(&b, img, nil)
	case "gif":
		err = gif.Encode(&b, img, nil)
	}
	if err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}
func uploadMP4(brand string) []byte {
	b := make([]byte, 24)
	binary.BigEndian.PutUint32(b, 24)
	copy(b[4:], "ftyp")
	copy(b[8:], brand)
	copy(b[16:], brand)
	copy(b[20:], "mp42")
	return b
}
func TestValidateUploadMediaFormatsAndLimits(t *testing.T) {
	for _, kind := range []string{"png", "jpeg", "gif"} {
		if err := ValidateUploadMedia("image", "misleading.dat", encodedUploadImage(t, kind)); err != nil {
			t.Fatalf("%s: %v", kind, err)
		}
	}
	for _, tc := range []struct {
		kind string
		data []byte
	}{
		{"file", []byte("12345")}, {"voice", []byte("#!AMR\n\x3c")}, {"voice", []byte("#!AMR-WB\n\x3c")}, {"video", uploadMP4("isom")},
	} {
		if err := ValidateUploadMedia(tc.kind, "media", tc.data); err != nil {
			t.Fatalf("%s: %v", tc.kind, err)
		}
	}
	for _, tc := range []struct {
		kind, name string
		data       []byte
	}{
		{"image", "fake.png", []byte("arbitrary")}, {"image", "cut.png", []byte("\x89PNG\r\n\x1a\n")},
		{"voice", "bad.amr", []byte("#!AMRfake")}, {"video", "fake.mp4", uploadMP4("avif")},
		{"video", "fake.mp4", uploadMP4("qt  ")}, {"video", "cut.mp4", uploadMP4("isom")[:12]},
		{"file", "tiny", []byte("1234")}, {"file", "", []byte("12345")},
		{"file", strings.Repeat("中", 86), []byte("12345")}, {"file", string([]byte{0xff}), []byte("12345")},
	} {
		if err := ValidateUploadMedia(tc.kind, tc.name, tc.data); !errors.Is(err, ErrInvalidArgument) {
			t.Fatalf("%s/%s accepted: %v", tc.kind, tc.name, err)
		}
	}
	for _, tc := range []struct {
		kind string
		max  int64
	}{{"image", 10 << 20}, {"voice", 2 << 20}, {"video", 10 << 20}, {"file", 20 << 20}} {
		limits, err := UploadMediaLimits(tc.kind)
		if err != nil || limits.MinBytes != 5 || limits.MaxBytes != tc.max {
			t.Fatalf("%s: %+v %v", tc.kind, limits, err)
		}
		if err = ValidateUploadMedia(tc.kind, "large", make([]byte, tc.max+1)); !errors.Is(err, ErrInvalidArgument) {
			t.Fatal(err)
		}
	}
	limits, _ := UploadMediaLimits("image")
	limits.Formats[0] = "wrong"
	fresh, _ := UploadMediaLimits("image")
	if fresh.Formats[0] == "wrong" {
		t.Fatal("mutable global formats")
	}
	if _, err := UploadMediaLimits("unknown"); !errors.Is(err, ErrInvalidArgument) {
		t.Fatal(err)
	}
	if _, err := NewStreamImage(encodedUploadImage(t, "gif")); !errors.Is(err, ErrInvalidArgument) {
		t.Fatal("GIF incorrectly accepted for inline images")
	}
	if err := ValidateUploadMedia("file", strings.Repeat("a", 256), []byte("12345")); err != nil {
		t.Fatal(err)
	}
}
