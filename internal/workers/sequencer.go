package workers

import (
	"fmt"
	"path/filepath"

	"github.com/cosmiclabstudio/cargodrop/internal/parsers"
	"github.com/cosmiclabstudio/cargodrop/internal/utils"
)

// RunUpdateSequence sequence for the app to start updating stuff
func RunUpdateSequence(config *parsers.Config, resources *parsers.ResourceSet, baseDir string, progressCb func(fileName string, downloadedBytes, totalBytes int64, processed, total int), errorCb func(string, error)) {
	// some introduction
	utils.LogRaw("Cargodrop ver.1.0 by Cosmic Lab Studio")
	utils.LogRaw("By using this software, you agree to the Terms of Conditions and the License of this program.")
	utils.LogRaw("Read more at: https://github.com/cosmiclabstudio/cargodrop") // TODO: Replace this link

	utils.LogMessage("Starting modpack update: " + config.Name)

	utils.LogRaw(config.WelcomeMessage)

	// Download remote_resources.json from server
	utils.LogMessage("Checking for updates...")
	remoteSet, err := parsers.LoadRemoteResource(config.UpdateServer + "/cargodrop.json")
	if err != nil {
		utils.LogError(err)
		errorCb("Failed to check for updates. Please check your internet connection and try again.", err)
		return
	}

	toUpdate := CheckResources(resources, baseDir)
	total := len(toUpdate)
	if total == 0 {
		utils.LogMessage("All resources are up to date.")
		progressCb("", 0, 0, total, total)
		return
	}

	// Helper to find remote resource by filename/location
	findRemote := func(filename, location string) *parsers.RemoteResource {
		for _, rr := range remoteSet.Resources {
			if rr.Filename == filename && rr.Location == location {
				return &rr
			}
		}
		return nil
	}

	failedUpdates := 0
	for i, r := range toUpdate {
		progressCb(r.Filename, 0, r.Size, i, total)
		utils.LogMessage("Updating: " + r.Filename + " (" + utils.FormatSize(r.Size) + ")")
		remote := findRemote(r.Filename, r.Location)
		if remote == nil {
			utils.LogError(fmt.Errorf("remote resource not found for %s/%s", r.Location, r.Filename))
			failedUpdates++
			continue
		}
		err := DownloadFile(remote.URL, filepath.Join(baseDir, r.Location, r.Filename), r.Filename, r.Size, func(fileName string, downloadedBytes, totalBytes int64) {
			progressCb(fileName, downloadedBytes, totalBytes, i, total)
		})
		if err != nil {
			utils.LogError(err)
			failedUpdates++
		} else {
			utils.LogMessage("Updated: " + r.Filename + " (" + utils.FormatSize(r.Size) + ")")
		}
	}

	progressCb("", 0, 0, total, total)

	if failedUpdates > 0 {
		errorMsg := fmt.Sprintf("Update completed with %d failed downloads. Some files may not be up to date.", failedUpdates)
		utils.LogMessage(errorMsg)
		errorCb(errorMsg, fmt.Errorf("%d files failed to download", failedUpdates))
	} else {
		utils.LogMessage("Modpack update complete.")
	}
}
