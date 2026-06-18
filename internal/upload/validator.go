package upload

import (
	"fmt"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
)

const (
	MaxFileSize = 500 << 20 // 500 MB
	MinFileSize = 1024      // 1 KB
)

var allowedExtensions = map[string]bool{
	".mp4":  true,
	".webm": true,
	".mkv":  true,
	".mov":  true,
	".avi":  true,
}

var allowedMIMETypes = map[string]bool{
	"video/mp4":        true,
	"video/webm":       true,
	"video/x-matroska": true,
	"video/quicktime":  true,
	"video/x-msvideo":  true,
}

// ValidationError describes a file upload validation failure.
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidateUpload checks file size, extension, and MIME type.
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

	// Extension check — reject unsupported formats early.
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !allowedExtensions[ext] {
		return &ValidationError{
			Field:   "file",
			Message: fmt.Sprintf("unsupported file extension: %s", ext),
		}
	}

	// MIME type check — sniff the first 512 bytes.
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

	mimeType := http.DetectContentType(buf[:n])
	if !allowedMIMETypes[mimeType] {
		return &ValidationError{
			Field:   "file",
			Message: fmt.Sprintf("unsupported MIME type: %s", mimeType),
		}
	}

	_ = n // used for mime detection
	return nil
}
