package workers

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cosmiclabstudio/cargodrop/internal/parsers"
	"github.com/cosmiclabstudio/cargodrop/internal/utils"
)

// updatePreserveList scans for user-installed mods and updates the preserve list
// Returns true if the config was modified and saved
func updatePreserveList(config *parsers.Config, resources *parsers.ResourceSet, baseDir string, configPath string) bool {
	utils.LogMessage("Checking for user-installed mods to preserve...")

	// Create a set of existing resources for quick lookup
	existingResources := make(map[string]bool)
	for _, resource := range resources.Resources {
		existingResources[resource.Path] = true
	}

	// Create pattern matchers
	ignoreMatcher := utils.NewPatternMatcher(config.Ignore)
	folderMatcher := utils.NewPatternMatcher(config.Folders)

	var userInstalledMods []string
	var newPreserveEntries []string

	// Scan for files that exist locally but are not in resources.json
	err := filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(baseDir, path)
		if err != nil {
			return err
		}

		if relPath == "." || info.IsDir() {
			return nil
		}

		// Convert to forward slashes for consistent pattern matching
		normalizedPath := filepath.ToSlash(relPath)

		// Skip ignored files
		if ignoreMatcher.ShouldIgnore(normalizedPath, false) {
			return nil
		}

		// Only consider files in included folders
		if !folderMatcher.ShouldIncludePath(normalizedPath) {
			return nil
		}

		// If file is not in resources.json, it's user-installed
		if !existingResources[normalizedPath] {
			userInstalledMods = append(userInstalledMods, normalizedPath)
		}

		return nil
	})

	if err != nil {
		utils.LogWarning("Failed to scan for user-installed mods: " + err.Error())
		return false
	}

	// Check if any user-installed mods should be added to preserve list
	for _, userMod := range userInstalledMods {
		// Check if this user mod is already in the preserve list
		alreadyPreserved := false
		for _, preserved := range config.Preserve {
			if preserved == userMod {
				alreadyPreserved = true
				break
			}
		}

		if !alreadyPreserved {
			newPreserveEntries = append(newPreserveEntries, userMod)
		}
	}

	// Add new preserve entries to existing list (no cleanup)
	updatedPreserve := append(config.Preserve, newPreserveEntries...)

	// Update config if new entries were added
	if len(newPreserveEntries) > 0 {
		config.Preserve = updatedPreserve

		// Save updated config
		if err := parsers.SaveConfig(config, configPath); err != nil {
			utils.LogWarning("Failed to save updated preserve list: " + err.Error())
			return false
		}

		utils.LogMessage("Updated preserve list with " + fmt.Sprintf("%d", len(newPreserveEntries)) + " new entries")
		return true
	}

	if len(userInstalledMods) > 0 {
		utils.LogMessage("Found " + fmt.Sprintf("%d", len(userInstalledMods)) + " user-installed mods (already preserved)")
	}

	return false
}

// processPatchRemovals handles file removal patches while respecting preserve list
func processPatchRemovals(remoteResources *parsers.ResourceSet, config *parsers.Config, baseDir string) {
	if len(remoteResources.Patches) == 0 {
		return
	}

	// Create preserve set for quick lookup
	preserveSet := make(map[string]bool)
	for _, preserved := range config.Preserve {
		preserveSet[preserved] = true
	}

	removedCount := 0
	for _, removeFile := range remoteResources.Patches {
		// Check if file is in preserve list
		if preserveSet[removeFile] {
			utils.LogMessage("Skipping removal of preserved file: " + removeFile)
			continue
		}

		// Check if file exists and remove it
		filePath := filepath.Join(baseDir, removeFile)
		if _, err := os.Stat(filePath); err == nil {
			if err := os.Remove(filePath); err != nil {
				utils.LogWarning("Failed to remove file " + removeFile + ": " + err.Error())
			} else {
				utils.LogMessage("Removed file: " + removeFile)
				removedCount++
			}
		}
	}
}

// RunUpdateSequence sequence for the app to start updating stuff
func RunUpdateSequence(config *parsers.Config, _ *parsers.ResourceSet, baseDir string, resourcePath string, configPath string, progressCb func(fileName string, downloadedBytes, totalBytes int64, processed, total int), errorCb func(string, error)) {
	utils.GetProgramInformation()
	utils.LogMessage("Starting update: " + config.Name)

	utils.LogRaw(config.WelcomeMessage)

	// Scan for disabled files once at the beginning
	disabledFilesMap := utils.ScanForDisabledFiles(baseDir, config.Folders, config.DisabledExtensions)

	// Download resources.json from server FIRST
	utils.LogMessage("Checking for updates...")

	// Download to a temporary file first
	tempResourcePath := resourcePath + ".tmp"
	resourceFileName := filepath.Base(resourcePath)

	err := DownloadFile(config.UpdateServer, tempResourcePath, resourceFileName, 0, func(fileName string, downloadedBytes, totalBytes int64) {
		// callback
	})
	if err != nil {
		utils.LogError(err)
		errorCb("Failed to check for updates. Please check your internet connection and try again.", err)
		return
	}

	// Parse the downloaded resources.json file
	remoteSet, err := parsers.LoadResource(tempResourcePath)
	if err != nil {
		utils.LogError(err)
		if removeErr := os.Remove(tempResourcePath); removeErr != nil {
			utils.LogWarning("Failed to clean up temp file: " + removeErr.Error())
		}
		errorCb("Failed to parse remote resources file.", err)
		return
	}

	// Process any file removals specified in patches BEFORE updating preserve list
	utils.LogMessage("Cleaning up...")
	processPatchRemovals(remoteSet, config, baseDir)

	// NOW update preserve list after patches have been processed
	updatePreserveList(config, remoteSet, baseDir, configPath)

	// Compare local files against remote resources
	utils.LogMessage("Downloading updates...")
	toUpdate := CheckResourcesWithDisabled(remoteSet, baseDir, disabledFilesMap)

	// Filter out ignored files from download list
	ignoreMatcher := utils.NewPatternMatcher(config.Ignore)
	var filteredToUpdate []parsers.Resource
	for _, resource := range toUpdate {
		if !ignoreMatcher.ShouldIgnore(resource.Path, false) {
			filteredToUpdate = append(filteredToUpdate, resource)
		} else {
			utils.LogMessage("Skipping ignored file: " + resource.Path)
		}
	}

	total := len(filteredToUpdate)
	if total == 0 {
		utils.LogMessage("All resources are up to date.")
		if removeErr := os.Remove(tempResourcePath); removeErr != nil {
			utils.LogWarning("Failed to clean up temp file: " + removeErr.Error())
		}
		progressCb("", 0, 0, total, total)
	} else {
		processed := 0
		for _, resource := range filteredToUpdate {
			processed++

			if resource.URL == "" {
				utils.LogWarning("Unable to download " + filepath.Base(resource.Path) + ", download URL is empty.")
				continue
			}

			// Create directory if needed
			fullPath := filepath.Join(baseDir, resource.Path)
			dir := filepath.Dir(fullPath)
			if err := os.MkdirAll(dir, 0755); err != nil {
				utils.LogError(err)
				errorCb("Failed to create directory: "+dir, err)
				return
			}

			// Check for disabled files using the pre-built map
			actualPath, disabledSuffix := utils.CheckDisabledFile(resource.Path, disabledFilesMap, baseDir)

			// Check if file already exists and has correct hash
			if fileExists(actualPath) && hasCorrectHash(actualPath, resource.Hash) {
				progressCb(filepath.Base(resource.Path), resource.Size, resource.Size, processed, total)
				continue
			}

			// If file was disabled, temporarily enable it for updating
			var tempPath string
			if disabledSuffix != "" {
				tempPath = strings.TrimSuffix(actualPath, disabledSuffix)
				utils.LogMessage("Temporarily enabling disabled file: " + filepath.Base(actualPath))
				if err := os.Rename(actualPath, tempPath); err != nil {
					utils.LogWarning("Failed to temporarily enable disabled file: " + err.Error())
					tempPath = actualPath // Fall back to original path
					disabledSuffix = ""   // Don't try to restore
				}
			} else {
				tempPath = fullPath
			}

			err := DownloadFile(resource.URL, tempPath, filepath.Base(resource.Path), resource.Size, func(fileName string, downloadedBytes, totalBytes int64) {
				progressCb(fileName, downloadedBytes, totalBytes, processed, total)
			})

			if err != nil {
				utils.LogError(err)
				errorCb("Download failed: "+filepath.Base(resource.Path), err)
				return
			}

			// Restore disabled extension if the file was originally disabled
			if disabledSuffix != "" {
				if err := utils.RestoreDisabledFile(tempPath, disabledSuffix); err != nil {
					utils.LogWarning("Failed to restore disabled state for " + filepath.Base(resource.Path))
				}
			}
		}
	}

	// Replace the original resources.json with the downloaded one
	if err := os.Rename(tempResourcePath, resourcePath); err != nil {
		utils.LogWarning("Failed to update resources file: " + err.Error())
		if removeErr := os.Remove(tempResourcePath); removeErr != nil {
			utils.LogWarning("Failed to clean up temp file: " + removeErr.Error())
		}
	}

	time.Sleep(2 * time.Second)
	utils.LogMessage("Done")
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// hasCorrectHash checks if a file has the correct hash
func hasCorrectHash(path, expectedHash string) bool {
	hash, err := utils.GenerateSHA1(path)
	if err != nil {
		return false
	}
	return hash == expectedHash
}

// CheckResourcesWithDisabled checks resources and handles disabled files using the pre-built map
func CheckResourcesWithDisabled(resources *parsers.ResourceSet, baseDir string, disabledFilesMap map[string]string) []parsers.Resource {
	var toUpdate []parsers.Resource

	for _, resource := range resources.Resources {
		// Check for disabled version using the map
		actualPath, _ := utils.CheckDisabledFile(resource.Path, disabledFilesMap, baseDir)

		// Check if file exists and has correct hash
		if !fileExists(actualPath) || !hasCorrectHash(actualPath, resource.Hash) {
			toUpdate = append(toUpdate, resource)
		}
	}

	return toUpdate
}
