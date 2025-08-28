package workers

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cosmiclabstudio/cargodrop/internal/api"
	"github.com/cosmiclabstudio/cargodrop/internal/parsers"
	"github.com/cosmiclabstudio/cargodrop/internal/utils"
)

func RunGenSourceSequence(config *parsers.Config, resources *parsers.ResourceSet, baseDir string, resourcesPath string, progressCb func(fileName string, downloadedBytes, totalBytes int64, processed, total int), errorCb func(string, error)) {
	// some introduction
	utils.LogRaw("Cargodrop ver.1.0 by Cosmic Lab Studio")
	utils.LogRaw("By using this software, you agree to the Terms of Conditions and the License of this program.")
	utils.LogRaw("Read more at: https://github.com/cosmiclabstudio/cargodrop") // TODO: Replace this link

	utils.LogWarning("Generating metadata, this will overwrite your provided resource.json!")

	utils.LogMessage("Resource name: " + resources.Name)
	utils.LogMessage("Version: " + resources.LocalVersion)
	utils.LogMessage("Please wait...")

	utils.LogMessage("Scanning folders: " + fmt.Sprintf("%v", config.Folders))

	newResources := &parsers.ResourceSet{
		Name:         config.Name,                                    // Copy from config
		LocalVersion: utils.IncrementVersion(resources.LocalVersion), // Increment version
		Resources:    []parsers.Resource{},
	}

	existingResources := make(map[string]*parsers.Resource)
	for i, resource := range resources.Resources {
		existingResources[resource.Path] = &resources.Resources[i]
	}

	totalFiles := 0
	for _, folder := range config.Folders {
		folderPath := filepath.Join(baseDir, folder)
		err := filepath.Walk(folderPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				totalFiles++
			}
			return nil
		})
		if err != nil {
			utils.LogError(fmt.Errorf("failed to scan folder %s: %v", folder, err))
			errorCb("Failed to scan folder "+folder, err)
			return
		}
	}

	processedFiles := 0
	for _, folder := range config.Folders {
		folderPath := filepath.Join(baseDir, folder)
		utils.LogMessage("Processing folder: " + folder)

		err := filepath.Walk(folderPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			filename := info.Name()
			progressCb(filename, 0, info.Size(), processedFiles, totalFiles)
			utils.LogMessage("Processing: " + filename + " (" + utils.FormatSize(info.Size()) + ")")

			// Generate SHA1 hash
			hash, err := utils.GenerateSHA1(path)
			if err != nil {
				utils.LogError(fmt.Errorf("failed to generate hash for %s: %v", filename, err))
				return err
			}

			// just in case we want to use other provider
			url, err := "", nil

			url, err = api.GetModrinthURL(hash, filename)
			if err != nil {
				utils.LogError(fmt.Errorf("", filename, err))
				return err
			}

			// Create resource entry
			resource := parsers.Resource{
				Path: filepath.Join(folder, filename),
				Hash: hash,
				Size: info.Size(),
				URL:  url,
			}

			// Preserve existing URL if it exists, only when there is actual text
			resourcePath := filepath.Join(folder, filename)
			if existing, exists := existingResources[resourcePath]; exists && len(resourcePath) > 0 {
				resource.URL = existing.URL
			}

			newResources.Resources = append(newResources.Resources, resource)
			processedFiles++
			progressCb(filename, info.Size(), info.Size(), processedFiles, totalFiles)
			return nil
		})

		if err != nil {
			utils.LogError(fmt.Errorf("failed to process folder %s: %v", folder, err))
			errorCb("Failed to process folder "+folder, err)
			return
		}
	}

	// Generate resource set hash
	newResources.ResourceSetHash = generateResourceSetHash(newResources)

	// Save updated resources.json to the original file path
	err := saveResourceSet(newResources, resourcesPath)
	if err != nil {
		utils.LogError(fmt.Errorf("failed to save resources.json: %v", err))
		errorCb("Failed to save resources.json", err)
		return
	}

	progressCb("", 0, 0, totalFiles, totalFiles)
	utils.LogMessage("Resource generation complete!")
	utils.LogMessage("New version: " + newResources.LocalVersion)
	utils.LogMessage("Total resources: " + fmt.Sprintf("%d", len(newResources.Resources)))
	utils.LogMessage("Saved to: " + resourcesPath)
	utils.LogMessage("Done!")
}

// generateResourceSetHash generates a hash for the entire resource set
func generateResourceSetHash(resources *parsers.ResourceSet) string {
	hash := sha1.New()
	for _, resource := range resources.Resources {
		hash.Write([]byte(resource.Path + resource.Hash))
	}
	return hex.EncodeToString(hash.Sum(nil))
}

// saveResourceSet saves the resource set to a JSON file
func saveResourceSet(resources *parsers.ResourceSet, outputPath string) error {
	data, err := json.MarshalIndent(resources, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(outputPath, data, 0644)
}
