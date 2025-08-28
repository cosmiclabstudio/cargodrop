package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/cosmiclabstudio/cargodrop/internal/utils"
)

// ModrinthVersionFile represents a file in a Modrinth version response
type ModrinthVersionFile struct {
	Hashes struct {
		SHA512 string `json:"sha512"`
		SHA1   string `json:"sha1"`
	} `json:"hashes"`
	URL      string `json:"url"`
	Filename string `json:"filename"`
	Primary  bool   `json:"primary"`
	Size     int64  `json:"size"`
}

// ModrinthVersionResponse represents the response from Modrinth API
type ModrinthVersionResponse struct {
	GameVersions []string              `json:"game_versions"`
	Loaders      []string              `json:"loaders"`
	ID           string                `json:"id"`
	ProjectID    string                `json:"project_id"`
	Name         string                `json:"name"`
	Files        []ModrinthVersionFile `json:"files"`
}

// GetModrinthURL fetches the download URL for a file from Modrinth using its SHA1 hash
func GetModrinthURL(hash, path string) (string, error) {
	// Extract filename from path
	filename := filepath.Base(path)

	// Make request to Modrinth API
	url := fmt.Sprintf("https://api.modrinth.com/v2/version_file/%s", hash)
	resp, err := http.Get(url)
	if err != nil {
		utils.LogError(err)
		return "", err
	}
	defer resp.Body.Close()

	// If not found (404), return empty string
	if resp.StatusCode == http.StatusNotFound {
		return "", nil
	}

	// Check for other non-200 status codes
	if resp.StatusCode != http.StatusOK {
		utils.LogError(fmt.Errorf("modrinth API returned status %d", resp.StatusCode))
		return "", fmt.Errorf("modrinth API returned status %d", resp.StatusCode)
	}

	// Parse JSON response
	var versionResp ModrinthVersionResponse
	if err := json.NewDecoder(resp.Body).Decode(&versionResp); err != nil {
		utils.LogError(fmt.Errorf("failed to decode modrinth response: %v", err))
		return "", fmt.Errorf("failed to decode modrinth response: %v", err)
	}

	// If no files, return empty string
	if len(versionResp.Files) == 0 {
		return "", nil
	}

	// If only one file, return its URL
	if len(versionResp.Files) == 1 {
		return versionResp.Files[0].URL, nil
	}

	// Multiple files - find matching filename
	for _, file := range versionResp.Files {
		if file.Filename == filename {
			return file.URL, nil
		}
	}

	// If no matching filename found, return the primary file or first file
	for _, file := range versionResp.Files {
		if file.Primary {
			return file.URL, nil
		}
	}

	// Fallback to first file
	return versionResp.Files[0].URL, nil
}
