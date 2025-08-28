package workers

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/cosmiclabstudio/cargodrop/internal/parsers"
	"github.com/cosmiclabstudio/cargodrop/internal/utils"
)

// RunUpdateSequence sequence for the app to start updating stuff
func RunUpdateSequence(config *parsers.Config, resources *parsers.ResourceSet, baseDir string, progressCb func(fileName string, downloadedBytes, totalBytes int64, processed, total int), errorCb func(string, error)) {
	// some introduction
	utils.LogRaw("Cargodrop ver.1.0 by Cosmic Lab Studio")
	utils.LogRaw("By using this software, you agree to the Terms of Conditions and the License of this program.")
	utils.LogRaw("Read more at: https://github.com/cosmiclabstudio/cargodrop") // TODO: Replace this link

	utils.LogMessage("Starting update: " + config.Name)

	utils.LogRaw(config.WelcomeMessage)

	// Download resources.json from server
	utils.LogMessage("Checking for updates...")

	resourcesPath := filepath.Join(baseDir, "resources.json")
	err := DownloadFile(config.UpdateServer, resourcesPath, "cargodrop.json", 0, func(fileName string, downloadedBytes, totalBytes int64) {
		// callback
	})
	if err != nil {
		utils.LogError(err)
		errorCb("Failed to check for updates. Please check your internet connection and try again.", err)
		return
	}

	// Parse the downloaded resources.json file
	remoteSet, err := parsers.LoadResource(resourcesPath)
	if err != nil {
		utils.LogError(err)
		errorCb("Failed to parse remote resources file.", err)
		return
	}

	toUpdate := CheckResources(resources, baseDir)
	total := len(toUpdate)
	if total == 0 {
		utils.LogMessage("All resources are up to date.")
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

		progressCb("", 0, 0, total, total)
		utils.LogMessage("Done!")
	}

	time.Sleep(3 * time.Second)
	os.Exit(0)
}
