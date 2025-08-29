package utils

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

func FormatSize(size int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)
	switch {
	case size >= GB:
		return fmt.Sprintf("%.2f GB", float64(size)/float64(GB))
	case size >= MB:
		return fmt.Sprintf("%.2f MB", float64(size)/float64(MB))
	case size >= KB:
		return fmt.Sprintf("%.2f KB", float64(size)/float64(KB))
	default:
		return fmt.Sprintf("%d B", size)
	}
}

func GenerateSHA1(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			LogError(fmt.Errorf("failed to close file %s: %v", filePath, closeErr))
		}
	}()

	hash := sha1.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// ShouldIgnore checks if a file or folder should be ignored based on ignore patterns
// Supports advanced gitignore-like patterns including **, !, and [abc] character classes
func ShouldIgnore(path string, ignorePatterns []string) bool {
	matcher := NewPatternMatcher(ignorePatterns)

	// Check if path is a directory by trying to stat it
	// If we can't stat it, assume it's a file
	isDir := false
	if info, err := os.Stat(path); err == nil {
		isDir = info.IsDir()
	}

	return matcher.ShouldIgnore(path, isDir)
}
