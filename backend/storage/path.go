package storage

import (
	"encoding/hex"
	"fmt"
	"io"
	"path/filepath"

	"lukechampine.com/blake3"
)

const (
	ResourceTypeVideo = "video"
	ResourceTypeAudio = "audio"
	ResourceTypeImage = "image"
)

// HashDir returns the relative directory path for a given hash and type.
func HashDir(resourceType, hash string) string {
	if len(hash) < 4 {
		panic(fmt.Sprintf("HashDir: hash too short (%d chars)", len(hash)))
	}
	return filepath.Join(resourceType, hash[0:2], hash[2:4], hash)
}

// HashPath returns the absolute directory path for a given hash and type.
func HashPath(dataDir, resourceType, hash string) string {
	return filepath.Join(dataDir, HashDir(resourceType, hash))
}

// OriginalPath returns the path to the original file for a given hash.
func OriginalPath(dataDir, resourceType, hash string) string {
	return filepath.Join(HashPath(dataDir, resourceType, hash), "original")
}

// ReadmePath returns the path to the readme file for a given hash.
func ReadmePath(dataDir, resourceType, hash string) string {
	return filepath.Join(HashPath(dataDir, resourceType, hash), "readme")
}

// HLSPath returns the HLS output directory for a video resource.
func HLSPath(dataDir, hash string) string {
	return filepath.Join(HashPath(dataDir, ResourceTypeVideo, hash), "hls")
}

// AudioOutputPath returns the transcoded audio output path.
func AudioOutputPath(dataDir, hash string) string {
	return filepath.Join(HashPath(dataDir, ResourceTypeAudio, hash), "transcoded", "output.mp3")
}

// ImagePath returns the path for a scaled image variant.
// variant: "thumb" or "medium"
func ImagePath(dataDir, hash, variant string) string {
	return filepath.Join(HashPath(dataDir, ResourceTypeImage, hash), variant)
}

// ComputeHash reads from reader and returns the BLAKE3-256 hex digest.
func ComputeHash(r io.Reader) (string, error) {
	hasher := blake3.New(32, nil)
	if _, err := io.Copy(hasher, r); err != nil {
		return "", fmt.Errorf("compute hash: %w", err)
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}
