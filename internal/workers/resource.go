package workers

import (
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/cosmiclabstudio/cargodrop/internal/parsers"
	"github.com/cosmiclabstudio/cargodrop/internal/utils"
)

// CheckResources compares local files to expected hashes and returns resources needing update
func CheckResources(rs *parsers.ResourceSet, baseDir string) []parsers.Resource {
	var toUpdate []parsers.Resource
	for _, r := range rs.Resources {
		localPath := filepath.Join(baseDir, r.Path)

		// Check if file exists
		if _, err := os.Stat(localPath); os.IsNotExist(err) {
			// File doesn't exist, needs to be downloaded
			toUpdate = append(toUpdate, r)
			continue
		}

		// File exists, compare SHA1 hash
		localHash, err := utils.GenerateSHA1(localPath)
		if err != nil || localHash != r.Hash {
			// Hash mismatch or error reading file, needs update
			toUpdate = append(toUpdate, r)
		}
		// If hash matches, file is up to date (no logging needed)
	}
	return toUpdate
}

// DownloadFile downloads a file and reports progress
func DownloadFile(url, localPath, fileName string, expectedSize int64, progressCb func(fileName string, downloadedBytes, totalBytes int64)) error {
	utils.LogMessage("Downloading " + fileName + " (" + utils.FormatSize(expectedSize) + ") ...")
	resp, err := http.Get(url)
	if err != nil {
		utils.LogError(err)
		return err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			utils.LogError(err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		utils.LogWarning("Download failed " + url + ": " + resp.Status)
		return err
	}

	dir := filepath.Dir(localPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		utils.LogError(err)
		return err
	}

	out, err := os.Create(localPath)
	if err != nil {
		utils.LogError(err)
		return err
	}
	defer func() {
		if err := out.Close(); err != nil {
			utils.LogError(err)
		}
	}()

	totalBytes := expectedSize
	var downloadedBytes int64
	buf := make([]byte, 32*1024)
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			wn, writeErr := out.Write(buf[:n])
			if writeErr != nil {
				utils.LogError(writeErr)
				return writeErr
			}
			downloadedBytes += int64(wn)
			if progressCb != nil {
				progressCb(fileName, downloadedBytes, totalBytes)
			}
		}
		if readErr != nil {
			if readErr != io.EOF {
				utils.LogError(readErr)
				return readErr
			}
			break
		}
	}
	return nil
}
