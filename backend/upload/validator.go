package upload

import (
	"fmt"
	"mime/multipart"
	"path/filepath"
	"strings"

	"github.com/gabriel-vasile/mimetype"
)

const (
	MaxFileSize = 500 << 20 // 500 MB
	MinFileSize = 1024      // 1 KB
)

// Allowed MIME types per resource type.
var videoMIMETypes = map[string]bool{
	"video/mp4": true, "video/webm": true, "video/x-matroska": true,
	"video/quicktime": true, "video/x-msvideo": true, "video/avi": true,
	"video/x-flv": true,
}
var audioMIMETypes = map[string]bool{
	"audio/mpeg": true, "audio/mp3": true, "audio/mp4": true,
	"audio/wav": true, "audio/x-wav": true, "audio/ogg": true,
	"audio/flac": true, "audio/aac": true, "audio/x-m4a": true,
	"audio/x-flac": true,
}
var imageMIMETypes = map[string]bool{
	"image/jpeg": true, "image/png": true, "image/webp": true, "image/gif": true,
	"image/bmp": true, "image/tiff": true, "image/svg+xml": true,
}

// allowedMIMETypes is the union of all allowed types.
var allowedMIMETypes = map[string]bool{}

func init() {
	for k, v := range videoMIMETypes {
		allowedMIMETypes[k] = v
	}
	for k, v := range audioMIMETypes {
		allowedMIMETypes[k] = v
	}
	for k, v := range imageMIMETypes {
		allowedMIMETypes[k] = v
	}
}

// DetectResourceType returns the resource type (video/audio/image) from a MIME type.
func DetectResourceType(mimeType string) string {
	mime := strings.ToLower(mimeType)
	// Strip parameters (e.g. "video/mp4; charset=binary" -> "video/mp4")
	if idx := strings.IndexByte(mime, ';'); idx != -1 {
		mime = strings.TrimSpace(mime[:idx])
	}
	if videoMIMETypes[mime] {
		return "video"
	}
	if audioMIMETypes[mime] {
		return "audio"
	}
	if imageMIMETypes[mime] {
		return "image"
	}
	return ""
}

// CorrectExtension returns the filename with the correct extension for the detected MIME type.
// If the file already has a correct extension, it is returned unchanged.
// If the extension is wrong or missing, the correct one is appended/replaced.
func CorrectExtension(filename, mimeType string) string {
	ext := expectedExtension(mimeType)
	if ext == "" {
		return filename // unknown MIME, can't correct
	}

	currentExt := strings.ToLower(filepath.Ext(filename))
	if currentExt == ext {
		return filename // already correct
	}

	// Strip wrong extension if present, then add correct one.
	base := strings.TrimSuffix(filename, filepath.Ext(filename))
	if base == "" {
		base = filename // no extension at all
	}
	return base + ext
}

// expectedExtension returns the canonical extension for a MIME type.
func expectedExtension(mimeType string) string {
	mime := strings.ToLower(mimeType)
	if idx := strings.IndexByte(mime, ';'); idx != -1 {
		mime = strings.TrimSpace(mime[:idx])
	}
	switch mime {
	case "video/mp4":
		return ".mp4"
	case "video/webm":
		return ".webm"
	case "video/x-matroska":
		return ".mkv"
	case "video/quicktime":
		return ".mov"
	case "video/x-msvideo", "video/avi":
		return ".avi"
	case "video/x-flv":
		return ".flv"
	case "audio/mpeg", "audio/mp3":
		return ".mp3"
	case "audio/mp4", "audio/x-m4a":
		return ".m4a"
	case "audio/wav", "audio/x-wav":
		return ".wav"
	case "audio/ogg":
		return ".ogg"
	case "audio/flac", "audio/x-flac":
		return ".flac"
	case "audio/aac":
		return ".aac"
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	case "image/bmp":
		return ".bmp"
	case "image/tiff":
		return ".tiff"
	case "image/svg+xml":
		return ".svg"
	}
	return ""
}

// ValidationError describes a file upload validation failure.
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// DetectMIMEAndResourceType reads the first 512 bytes and detects the MIME type
// using magic bytes. Returns the detected MIME type and resource type.
// The reader is reset to position 0 after detection.
func DetectMIMEAndResourceType(file multipart.File) (mimeType string, resourceType string, err error) {
	// Read header bytes for magic number detection.
	buf := make([]byte, 512)
	n, readErr := file.Read(buf)
	if readErr != nil || n == 0 {
		return "", "", fmt.Errorf("cannot read file header for magic number detection")
	}

	// Detect MIME from magic bytes using the mimetype library.
	detected := mimetype.Detect(buf[:n])
	if detected == nil {
		return "", "", fmt.Errorf("unable to detect MIME type from file content")
	}
	mimeType = detected.String()

	resourceType = DetectResourceType(mimeType)
	if resourceType == "" {
		return mimeType, "", fmt.Errorf("unsupported MIME type: %s", mimeType)
	}

	// Reset read position for subsequent processing (hash computation etc.)
	if _, err := file.Seek(0, 0); err != nil {
		return "", "", fmt.Errorf("failed to reset file position: %w", err)
	}

	return mimeType, resourceType, nil
}

// ValidateUpload checks file size and MIME type (magic bytes only, no extension check).
func ValidateUpload(file multipart.File, header *multipart.FileHeader) error {
	// Size check — fail fast for invalid sizes.
	if header.Size > MaxFileSize {
		return &ValidationError{
			Field:   "file",
			Message: fmt.Sprintf("file too large (max %d MB)", MaxFileSize/(1<<20)),
		}
	}
	if header.Size < MinFileSize {
		return &ValidationError{
			Field:   "file",
			Message: "file too small",
		}
	}

	// MIME type check — sniff the first 512 bytes using magic numbers.
	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil {
		return &ValidationError{
			Field:   "file",
			Message: "cannot read file header",
		}
	}

	// Reset file read position to the start for subsequent processing.
	file.Seek(0, 0)

	mimeType := mimetype.Detect(buf[:n])
	if mimeType == nil || mimeType.String() == "application/octet-stream" {
		return &ValidationError{
			Field:   "file",
			Message: "unable to detect file type from content",
		}
	}

	if !allowedMIMETypes[mimeType.String()] {
		return &ValidationError{
			Field:   "file",
			Message: fmt.Sprintf("unsupported file type: %s", mimeType.String()),
		}
	}

	_ = n
	return nil
}
