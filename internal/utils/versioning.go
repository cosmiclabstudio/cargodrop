package utils

import (
	"strconv"
	"strings"
)

// IncrementVersion increments a version string by the smallest unit
// Examples: "1.0" -> "1.1", "1.0.0" -> "1.0.1", "1.5.9" -> "1.5.10"
func IncrementVersion(version string) string {
	parts := strings.Split(version, ".")
	if len(parts) == 0 {
		return "1.0"
	}

	// Find the last part and increment it
	lastIdx := len(parts) - 1
	lastPart, err := strconv.Atoi(parts[lastIdx])
	if err != nil {
		// If the last part is not a number, append .1
		return version + ".1"
	}

	parts[lastIdx] = strconv.Itoa(lastPart + 1)
	return strings.Join(parts, ".")
}
