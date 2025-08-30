package utils

import (
	"os"
	"path/filepath"
	"strings"
)

// HasDisabledExtension checks if a file has a disabled extension and returns the base path
// Returns the base path (without disabled extension) and whether it was disabled
func HasDisabledExtension(filePath string, disabledExtensions []string) (string, bool) {
	if len(disabledExtensions) == 0 {
		return filePath, false
	}

	for _, ext := range disabledExtensions {
		if strings.HasSuffix(filePath, ext) {
			basePath := strings.TrimSuffix(filePath, ext)
			return basePath, true
		}
	}

	return filePath, false
}

// RestoreDisabledExtension restores the disabled extension to a file path
// If the original file had a disabled extension, it restores it to the updated file
func RestoreDisabledExtension(originalPath, updatedPath string, disabledExtensions []string) string {
	if len(disabledExtensions) == 0 {
		return updatedPath
	}

	// Check if original file had a disabled extension
	for _, ext := range disabledExtensions {
		if strings.HasSuffix(originalPath, ext) {
			return updatedPath + ext
		}
	}

	return updatedPath
}

// ScanForDisabledFiles scans a directory for files with disabled extensions
// Returns a map of base paths to their disabled paths
func ScanForDisabledFiles(baseDir string, folders []string, disabledExtensions []string) map[string]string {
	disabledFiles := make(map[string]string)

	if len(disabledExtensions) == 0 {
		return disabledFiles
	}

	folderMatcher := NewPatternMatcher(folders)

	_ = filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}

		relPath, err := filepath.Rel(baseDir, path)
		if err != nil {
			return err
		}

		normalizedPath := filepath.ToSlash(relPath)

		// Only consider files in included folders
		if !folderMatcher.ShouldIncludePath(normalizedPath) {
			return nil
		}

		// Check if file has disabled extension
		basePath, isDisabled := HasDisabledExtension(normalizedPath, disabledExtensions)
		if isDisabled {
			disabledFiles[basePath] = normalizedPath
		}

		return nil
	})

	return disabledFiles
}

// CheckDisabledFile checks if a resource path has a corresponding disabled file
// Returns the actual file path and any disabled suffix that should be restored
func CheckDisabledFile(resourcePath string, disabledFilesMap map[string]string, baseDir string) (string, string) {
	// Check if this resource path has a disabled version
	if disabledPath, exists := disabledFilesMap[resourcePath]; exists {
		fullDisabledPath := filepath.Join(baseDir, disabledPath)
		// Check if the disabled file actually exists
		if _, err := os.Stat(fullDisabledPath); err == nil {
			// Extract the disabled suffix
			disabledSuffix := strings.TrimPrefix(disabledPath, resourcePath)
			return fullDisabledPath, disabledSuffix
		}
	}

	// Return the normal path if no disabled version exists
	return filepath.Join(baseDir, resourcePath), ""
}

// RestoreDisabledFile renames a file back to its disabled state
func RestoreDisabledFile(filePath, disabledSuffix string) error {
	if disabledSuffix == "" {
		return nil
	}

	disabledPath := filePath + disabledSuffix
	return os.Rename(filePath, disabledPath)
}
