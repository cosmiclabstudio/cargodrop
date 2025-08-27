package workers

import (
	"crypto/md5"
	"encoding/hex"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/cosmiclabstudio/cargodrop/internal/parsers"
	"github.com/cosmiclabstudio/cargodrop/internal/utils"
)

// ComputeFileHash computes the MD5 hash of a file at the given path
func ComputeFileHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
		}
	}(f)

	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// CheckResources compares local files to expected hashes and returns resources needing update
func CheckResources(rs *parsers.ResourceSet, baseDir string) []parsers.Resource {
	var toUpdate []parsers.Resource
	for _, r := range rs.Resources {
		localPath := filepath.Join(baseDir, r.Location, r.Filename)
		localHash, err := ComputeFileHash(localPath)
		if err != nil || localHash != r.Hash {
			toUpdate = append(toUpdate, r)
		}
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
