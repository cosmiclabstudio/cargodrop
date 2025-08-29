package utils

import (
	"path/filepath"
	"regexp"
	"strings"
)

// PatternMatcher handles gitignore-like pattern matching with advanced features
type PatternMatcher struct {
	patterns []Pattern
}

// Pattern represents a single pattern with its type and compiled regex
type Pattern struct {
	original    string
	isNegation  bool
	isDirectory bool
	compiled    *regexp.Regexp
}

// NewPatternMatcher creates a new pattern matcher with the given patterns
func NewPatternMatcher(patterns []string) *PatternMatcher {
	pm := &PatternMatcher{}
	for _, pattern := range patterns {
		if compiled := pm.compilePattern(pattern); compiled != nil {
			pm.patterns = append(pm.patterns, *compiled)
		}
	}
	return pm
}

// ShouldIgnore checks if a path should be ignored based on the configured patterns
func (pm *PatternMatcher) ShouldIgnore(path string, isDir bool) bool {
	if len(pm.patterns) == 0 {
		return false
	}

	// Normalize path separators for consistent matching
	normalizedPath := filepath.ToSlash(path)

	matched := false

	// Process patterns in order - later patterns can override earlier ones
	for _, pattern := range pm.patterns {
		if pm.matchesPattern(pattern, normalizedPath, isDir) {
			if pattern.isNegation {
				matched = false // Negation pattern - don't ignore
			} else {
				matched = true // Regular pattern - ignore
			}
		}
	}

	return matched
}

// ShouldIncludeFolder checks if a folder should be included based on folder patterns
// Now properly handles negation patterns for folders too
func (pm *PatternMatcher) ShouldIncludeFolder(path string) bool {
	if len(pm.patterns) == 0 {
		return false // No patterns means no folders are included
	}

	// Normalize path separators for consistent matching
	normalizedPath := filepath.ToSlash(path)

	matched := false

	// Process patterns in order - later patterns can override earlier ones (same logic as ShouldIgnore)
	for _, pattern := range pm.patterns {
		if pm.matchesPattern(pattern, normalizedPath, true) {
			if pattern.isNegation {
				matched = false // Negation pattern - don't include
			} else {
				matched = true // Regular pattern - include
			}
		}
	}

	return matched
}

// ShouldIncludePath checks if a file path should be included based on its location
// This checks if the file is in a directory that matches any folder pattern
func (pm *PatternMatcher) ShouldIncludePath(filePath string) bool {
	if len(pm.patterns) == 0 {
		return false
	}

	normalizedPath := filepath.ToSlash(filePath)

	// For files, we need to check if the file's directory (not the file itself) matches any folder pattern
	fileDir := filepath.Dir(normalizedPath)
	if fileDir == "." {
		fileDir = ""
	}

	// Check if the file's directory matches any folder pattern directly
	if fileDir != "" && pm.ShouldIncludeFolder(fileDir) {
		return true
	}

	// Also check each parent directory to see if any match
	if fileDir != "" {
		pathParts := strings.Split(fileDir, "/")
		for i := 0; i < len(pathParts); i++ {
			parentPath := strings.Join(pathParts[:i+1], "/")
			if pm.ShouldIncludeFolder(parentPath) {
				return true
			}
		}
	}

	// Additionally, check if any folder pattern would match this file's location
	// This handles cases where the pattern is more specific than the current directory
	for _, pattern := range pm.patterns {
		if pattern.isNegation {
			continue // Skip negation patterns for inclusion check
		}

		// Check if the file path starts with the pattern (for specific folder paths)
		patternPath := strings.TrimSuffix(pattern.original, "/")
		if strings.HasPrefix(normalizedPath, patternPath+"/") || normalizedPath == patternPath {
			return true
		}
	}

	return false
}

// compilePattern compiles a gitignore-like pattern into a Pattern struct
func (pm *PatternMatcher) compilePattern(pattern string) *Pattern {
	if pattern == "" {
		return nil
	}

	// Remove leading/trailing whitespace
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return nil
	}

	p := Pattern{ // Remove & here - create struct directly
		original: pattern,
	}

	// Check for negation
	if strings.HasPrefix(pattern, "!") {
		p.isNegation = true
		pattern = pattern[1:]
	}

	// Check for directory-only pattern
	if strings.HasSuffix(pattern, "/") {
		p.isDirectory = true
		pattern = strings.TrimSuffix(pattern, "/")
	}

	// Convert gitignore pattern to regex
	regexPattern := pm.patternToRegex(pattern)

	// Compile regex
	compiled, err := regexp.Compile("^" + regexPattern + "$")
	if err != nil {
		LogWarning("Invalid pattern '" + p.original + "': " + err.Error())
		return nil
	}

	p.compiled = compiled
	return &p // Now return pointer to p
}

// patternToRegex converts a gitignore-style pattern to a regex pattern
func (pm *PatternMatcher) patternToRegex(pattern string) string {
	// Escape special regex characters except our wildcards
	pattern = regexp.QuoteMeta(pattern)

	// Restore our wildcards and convert to regex
	pattern = strings.ReplaceAll(pattern, `\*\*`, "__DOUBLE_STAR__")
	pattern = strings.ReplaceAll(pattern, `\*`, "[^/]*")
	pattern = strings.ReplaceAll(pattern, `\?`, "[^/]")
	pattern = strings.ReplaceAll(pattern, "__DOUBLE_STAR__", ".*")

	// Handle character classes [abc] and [!abc]
	pattern = pm.handleCharacterClasses(pattern)

	return pattern
}

// handleCharacterClasses processes [abc] and [!abc] patterns
func (pm *PatternMatcher) handleCharacterClasses(pattern string) string {
	// Handle [!abc] (negated character class) - fix the missing closing bracket escape
	re := regexp.MustCompile(`\\\[!([^\]]+)\\\]`)
	pattern = re.ReplaceAllString(pattern, "[^$1]")

	// Handle [abc] (normal character class) - fix the missing closing bracket escape
	re = regexp.MustCompile(`\\\[([^\]]+)\\\]`)
	pattern = re.ReplaceAllString(pattern, "[$1]")

	return pattern
}

// matchesPattern checks if a path matches a specific pattern
func (pm *PatternMatcher) matchesPattern(pattern Pattern, path string, isDir bool) bool {
	// If pattern is directory-only and path is not a directory, skip
	if pattern.isDirectory && !isDir {
		return false
	}

	// Try to match the full path
	if pattern.compiled.MatchString(path) {
		return true
	}

	// Try to match each part of the path (for patterns without **)
	if !strings.Contains(pattern.original, "**") {
		pathParts := strings.Split(path, "/")

		// Match basename
		if pattern.compiled.MatchString(pathParts[len(pathParts)-1]) {
			return true
		}

		// Match any part of the path
		for _, part := range pathParts {
			if pattern.compiled.MatchString(part) {
				return true
			}
		}

		// Match subpaths
		for i := 0; i < len(pathParts); i++ {
			subPath := strings.Join(pathParts[i:], "/")
			if pattern.compiled.MatchString(subPath) {
				return true
			}
		}
	}

	return false
}
