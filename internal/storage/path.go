package storage

import (
	"encoding/hex"
	"fmt"
	"io"
	"path/filepath"

	"lukechampine.com/blake3"
)

// HashDir returns the relative directory path for a given hash:
// video/{hash[0:2]}/{hash[2:4]}/{hash}
func HashDir(hash string) string {
	if len(hash) < 4 {
		panic(fmt.Sprintf("HashDir: hash too short (%d chars)", len(hash)))
	}
	return filepath.Join("video", hash[0:2], hash[2:4], hash)
}

// HashPath returns the absolute directory path for a given hash:
// dataDir/video/{hash[0:2]}/{hash[2:4]}/{hash}
func HashPath(dataDir, hash string) string {
	return filepath.Join(dataDir, HashDir(hash))
}

// OriginalPath returns the path to the original video file for a given hash:
// dataDir/video/{hash[0:2]}/{hash[2:4]}/{hash}/original
func OriginalPath(dataDir, hash string) string {
	return filepath.Join(HashPath(dataDir, hash), "original")
}

// ReadmePath returns the path to the readme file for a given hash:
// dataDir/video/{hash[0:2]}/{hash[2:4]}/{hash}/readme
func ReadmePath(dataDir, hash string) string {
	return filepath.Join(HashPath(dataDir, hash), "readme")
}

// ComputeHash reads from reader and returns the BLAKE3-256 hex digest.
func ComputeHash(r io.Reader) (string, error) {
	hasher := blake3.New(32, nil)
	if _, err := io.Copy(hasher, r); err != nil {
		return "", fmt.Errorf("compute hash: %w", err)
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}
