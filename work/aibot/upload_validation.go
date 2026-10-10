package aibot

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	_ "image/gif" // Register the upload formats independently of inline images.
	_ "image/jpeg"
	_ "image/png"
	"unicode/utf8"
)

// MediaLimits describes platform upload limits. Formats is a caller-owned slice;
// an empty list means that ordinary files have no format restriction.
type MediaLimits struct {
	MinBytes, MaxBytes int64
	Formats            []string
}

func UploadMediaLimits(mediaType string) (MediaLimits, error) {
	limits := MediaLimits{MinBytes: 5}
	switch mediaType {
	case "image":
		limits.MaxBytes = 10 << 20
		limits.Formats = []string{"png", "jpeg", "gif"}
	case "voice":
		limits.MaxBytes = 2 << 20
		limits.Formats = []string{"amr"}
	case "video":
		limits.MaxBytes = 10 << 20
		limits.Formats = []string{"mp4"}
	case "file":
		limits.MaxBytes = 20 << 20
	default:
		return MediaLimits{}, fmt.Errorf("%w: unknown upload type", ErrInvalidArgument)
	}
	return limits, nil
}

// ValidateUploadMedia performs bounded size and header/container preflight.
// It does not decode entire files or guarantee server acceptance or codec support.
// Filename extensions do not determine the format.
func ValidateUploadMedia(mediaType, filename string, data []byte) error {
	limits, err := UploadMediaLimits(mediaType)
	if err != nil {
		return err
	}
	if filename == "" || len(filename) > 256 || !utf8.ValidString(filename) {
		return fmt.Errorf("%w: upload filename must contain 1..256 UTF-8 bytes", ErrInvalidArgument)
	}
	if int64(len(data)) < limits.MinBytes || int64(len(data)) > limits.MaxBytes {
		return fmt.Errorf("%w: upload size outside platform limits", ErrInvalidArgument)
	}
	valid := true
	switch mediaType {
	case "image":
		_, format, decodeErr := image.DecodeConfig(bytes.NewReader(data))
		valid = decodeErr == nil && (format == "png" || format == "jpeg" || format == "gif")
	case "voice":
		valid = bytes.HasPrefix(data, []byte("#!AMR\n")) || bytes.HasPrefix(data, []byte("#!AMR-WB\n"))
	case "video":
		valid = isUploadMP4(data)
	}
	if !valid {
		return fmt.Errorf("%w: unsupported or malformed upload format", ErrInvalidArgument)
	}
	return nil
}

func isUploadMP4(data []byte) bool {
	// Inspect complete leading boxes only. Avoid interpreting compatible MP4
	// brands in known image/QuickTime containers as an MP4 video upload.
	for offset, boxes := 0, 0; offset+8 <= len(data) && boxes < 32; boxes++ {
		size := uint64(binary.BigEndian.Uint32(data[offset:]))
		header := 8
		kind := string(data[offset+4 : offset+8])
		if size == 1 {
			if offset+16 > len(data) {
				return false
			}
			size = binary.BigEndian.Uint64(data[offset+8:])
			header = 16
		}
		if size < uint64(header) || size > uint64(len(data)-offset) {
			return false
		}
		end := offset + int(size)
		if kind == "ftyp" {
			body := data[offset+header : end]
			if len(body) < 8 || (len(body)-8)%4 != 0 {
				return false
			}
			switch string(body[:4]) {
			case "avif", "avis", "heic", "heix", "hevc", "hevx", "mif1", "msf1", "qt  ":
				return false
			}
			if mp4Brand(string(body[:4])) {
				return true
			}
			for pos := 8; pos+4 <= len(body); pos += 4 {
				if mp4Brand(string(body[pos : pos+4])) {
					return true
				}
			}
			return false
		}
		if kind != "free" && kind != "skip" && kind != "wide" {
			return false
		}
		offset = end
	}
	return false
}
func mp4Brand(brand string) bool {
	switch brand {
	case "isom", "iso2", "iso3", "iso4", "iso5", "iso6", "iso7", "iso8", "iso9", "mp41", "mp42", "avc1", "M4V ", "M4VH", "M4VP", "MSNV", "dash":
		return true
	}
	return false
}
