package workers

import (
	"fmt"
	"os"
	"path/filepath"
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

		// Skip ignored files
		if ignoreMatcher.ShouldIgnore(relPath, false) {
			return nil
		}

		// Only consider files in included folders
		if !folderMatcher.ShouldIncludePath(relPath) {
			return nil
		}

		// If file is not in resources.json, it's user-installed
		if !existingResources[relPath] {
			userInstalledMods = append(userInstalledMods, relPath)
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

	// Add new preserve entries to existing listno cleanup)
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
func RunUpdateSequence(config *parsers.Config, resources *parsers.ResourceSet, baseDir string, resourcePath string, progressCb func(fileName string, downloadedBytes, totalBytes int64, processed, total int), errorCb func(string, error)) {
	utils.GetProgramInformation()
	utils.LogMessage("Starting update: " + config.Name)

	utils.LogRaw(config.WelcomeMessage)

	// Update preserve list before checking for updates
	configPath := filepath.Join(filepath.Dir(resourcePath), "cargodrop.json")
	updatePreserveList(config, resources, baseDir, configPath)

	// Download resources.json from server
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

	// Process any file removals specified in patches before updating files
	utils.LogMessage("Cleaning up...")
	processPatchRemovals(remoteSet, config, baseDir)

	// Compare local files against remote resources
	utils.LogMessage("Downloading updates...")
	toUpdate := CheckResources(remoteSet, baseDir)
	total := len(toUpdate)
	if total == 0 {
		utils.LogMessage("All resources are up to date.")
		if removeErr := os.Remove(tempResourcePath); removeErr != nil {
			utils.LogWarning("Failed to clean up temp file: " + removeErr.Error())
		}
		progressCb("", 0, 0, total, total)
	} else {
		// Helper to find remote resource by path
		findRemote := func(path string) *parsers.Resource {
			for _, r := range remoteSet.Resources {
				if r.Path == path {
					return &r
				}
			}
			return nil
		}

		for i, r := range toUpdate {
			filename := filepath.Base(r.Path)
			progressCb(filename, 0, r.Size, i, total)
			remote := findRemote(r.Path)
			if remote == nil {
				err := fmt.Errorf("remote resource not found for %s", r.Path)
				utils.LogError(err)
				errorCb("Failed to find remote resource for "+filename, err)
				return
			}

			// Check if URL is empty
			if remote.URL == "" {
				utils.LogWarning("Unable to download " + filename + ", download URL is empty.")
				continue // Skip this file and continue with the next one
			}

			err := DownloadFile(remote.URL, filepath.Join(baseDir, r.Path), filename, r.Size, func(fileName string, downloadedBytes, totalBytes int64) {
				progressCb(fileName, downloadedBytes, totalBytes, i, total)
			})
			if err != nil {
				utils.LogError(err)
				errorCb("Failed to download "+filename, err)
				return
			}
		}

		// After successful updates, replace the local resources.json with the new one
		if err := os.Rename(tempResourcePath, resourcePath); err != nil {
			utils.LogWarning("Failed to update local resources file: " + err.Error())
			if removeErr := os.Remove(tempResourcePath); removeErr != nil {
				utils.LogWarning("Failed to clean up temp file: " + removeErr.Error())
			}
		} else {
			utils.LogMessage("Resources file updated successfully.")
		}

		progressCb("", 0, 0, total, total)
		utils.LogMessage("Done!")
	}

	time.Sleep(3 * time.Second)
	os.Exit(0)
}
