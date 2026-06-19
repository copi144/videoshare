package upload

import (
	"fmt"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
)

// allowedExtensions maps extension → resource type for fallback detection.
var extensionResourceType = map[string]string{
	".mp4": "video", ".webm": "video", ".mkv": "video", ".mov": "video", ".avi": "video",
	".mp3": "audio", ".m4a": "audio", ".wav": "audio", ".ogg": "audio", ".flac": "audio", ".aac": "audio",
	".jpg": "image", ".jpeg": "image", ".png": "image", ".webp": "image", ".gif": "image",
}

// DetectResourceTypeByExtension returns the resource type based on file extension.
func DetectResourceTypeByExtension(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	if rt, ok := extensionResourceType[ext]; ok {
		return rt
	}
	return ""
}

const (
	MaxFileSize = 500 << 20 // 500 MB
	MinFileSize = 1024      // 1 KB
)

// Accepted MIME types per resource type.
var videoMIMETypes = []string{"video/mp4", "video/webm", "video/x-matroska", "video/quicktime", "video/avi", "video/x-msvideo"}
var audioMIMETypes = []string{"audio/mpeg", "audio/mp3", "audio/mp4", "audio/wav", "audio/x-wav", "audio/ogg", "audio/flac", "audio/aac", "audio/x-m4a"}
var imageMIMETypes = []string{"image/jpeg", "image/png", "image/webp", "image/gif"}

// DetectResourceType returns the resource type (video/audio/image) based on MIME type.
func DetectResourceType(contentType string) string {
	for _, mt := range videoMIMETypes {
		if strings.EqualFold(contentType, mt) {
			return "video"
		}
	}
	for _, mt := range audioMIMETypes {
		if strings.EqualFold(contentType, mt) {
			return "audio"
		}
	}
	for _, mt := range imageMIMETypes {
		if strings.EqualFold(contentType, mt) {
			return "image"
		}
	}
	return ""
}

var allowedExtensions = map[string]bool{
	".mp4":  true,
	".webm": true,
	".mkv":  true,
	".mov":  true,
	".avi":  true,
	".mp3":  true,
	".m4a":  true,
	".wav":  true,
	".ogg":  true,
	".flac": true,
	".aac":  true,
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".webp": true,
	".gif":  true,
}

var allowedMIMETypes = map[string]bool{
	"video/mp4":        true,
	"video/webm":       true,
	"video/x-matroska": true,
	"video/quicktime":  true,
	"video/x-msvideo":  true,
	"audio/mpeg":       true,
	"audio/mp3":        true,
	"audio/mp4":        true,
	"audio/wav":        true,
	"audio/x-wav":      true,
	"audio/ogg":        true,
	"audio/flac":       true,
	"audio/aac":        true,
	"audio/x-m4a":      true,
	"image/jpeg":       true,
	"image/png":        true,
	"image/webp":       true,
	"image/gif":        true,
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
