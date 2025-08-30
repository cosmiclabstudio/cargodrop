package workers

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cosmiclabstudio/cargodrop/internal/api"
	"github.com/cosmiclabstudio/cargodrop/internal/parsers"
	"github.com/cosmiclabstudio/cargodrop/internal/utils"
)

var isUsingService = false

func RunGenSourceSequence(config *parsers.Config, resources *parsers.ResourceSet, baseDir string, resourcesPath string, progressCb func(fileName string, downloadedBytes, totalBytes int64, processed, total int), errorCb func(string, error), isServiceModrinth bool) {
	utils.GetProgramInformation()

	utils.LogWarning("Generating metadata, this will overwrite your provided resource file!")

	utils.LogMessage("Resource name: " + resources.Name)
	utils.LogMessage("Version: " + resources.LocalVersion)

	totalFiles := 0

	if isServiceModrinth {
		isUsingService = true
		utils.LogMessage("Using Modrinth Provider.")
	}

	utils.LogMessage("Scanning folders: " + fmt.Sprintf("%v", config.Folders))

	ignoreMatcher := utils.NewPatternMatcher(config.Ignore)
	folderMatcher := utils.NewPatternMatcher(config.Folders)

	err := filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(baseDir, path)
		if err != nil {
			return err
		}

		if relPath == "." {
			return nil
		}

		if ignoreMatcher.ShouldIgnore(relPath, info.IsDir()) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if info.IsDir() {
			// For directories, we need to be more permissive to allow walking into subdirectories
			// Check if this directory should be included OR if it could be a parent of an included directory
			if folderMatcher.ShouldIncludeFolder(relPath) {
				return nil
			}

			// Check if this directory could be a parent of any folder pattern
			for _, pattern := range config.Folders {
				patternPath := strings.TrimSuffix(pattern, "/")
				if strings.HasPrefix(patternPath, relPath+"/") || strings.HasPrefix(patternPath, relPath) {
					return nil
				}
			}

			// This directory doesn't match any patterns and isn't a parent of any pattern
			return filepath.SkipDir
		} else {
			// For files, use the fixed ShouldIncludePath logic
			if !folderMatcher.ShouldIncludePath(relPath) {
				return nil
			}
			totalFiles++
		}

		return nil
	})

	if err != nil {
		utils.LogError(fmt.Errorf("failed to scan folders: %v", err))
		errorCb("Failed to scan folders", err)
		return
	}

	newResources := &parsers.ResourceSet{
		Name:         config.Name,
		LocalVersion: utils.IncrementVersion(resources.LocalVersion),
		Patches:      resources.Patches, // Preserve existing patches
		Resources:    []parsers.Resource{},
	}

	existingResources := make(map[string]*parsers.Resource)
	for i, resource := range resources.Resources {
		existingResources[resource.Path] = &resources.Resources[i]
	}

	// Track missing URLs for developer notification
	var missingURLs []string

	processedFiles := 0
	utils.LogMessage("Processing files...")

	err = filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		resourceRelPath, err := filepath.Rel(baseDir, path)
		if err != nil {
			return err
		}

		if resourceRelPath == "." {
			return nil
		}

		if ignoreMatcher.ShouldIgnore(resourceRelPath, info.IsDir()) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if info.IsDir() {
			// For directories, we need to be more permissive to allow walking into subdirectories
			// Check if this directory should be included OR if it could be a parent of an included directory
			if folderMatcher.ShouldIncludeFolder(resourceRelPath) {
				// This directory matches a pattern, continue walking
				return nil
			}

			// Check if this directory could be a parent of any folder pattern
			for _, pattern := range config.Folders {
				patternPath := strings.TrimSuffix(pattern, "/")
				// If any folder pattern starts with this directory path, allow walking
				if strings.HasPrefix(patternPath, resourceRelPath+"/") || strings.HasPrefix(patternPath, resourceRelPath) {
					return nil // Continue walking, this could lead to a matching folder
				}
			}

			// This directory doesn't match any patterns and isn't a parent of any pattern
			return filepath.SkipDir
		}

		// For files, check if they should be included
		if !folderMatcher.ShouldIncludePath(resourceRelPath) {
			return nil
		}

		filename := info.Name()
		progressCb(filename, 0, info.Size(), processedFiles, totalFiles)
		utils.LogMessage("Processing: " + filename + " (" + utils.FormatSize(info.Size()) + ")")

		hash, err := utils.GenerateSHA1(path)
		if err != nil {
			utils.LogError(fmt.Errorf("failed to generate hash for %s: %v", filename, err))
			return err
		}

		url, err := "", nil

		if isServiceModrinth {
			url, err = api.GetModrinthURL(hash, filename)
			if err != nil {
				utils.LogError(fmt.Errorf("failed to get Modrinth URL for %s: %v", filename, err))
				return err
			}
		}

		resource := parsers.Resource{
			Path: resourceRelPath,
			Hash: hash,
			Size: info.Size(),
			URL:  url,
		}

		if existing, exists := existingResources[resourceRelPath]; exists && len(existing.URL) > 0 {
			resource.URL = existing.URL
		}

		// Check if URL is missing after all attempts to get it (including existing resources)
		if resource.URL == "" && isUsingService {
			missingURLs = append(missingURLs, filename)
		}

		newResources.Resources = append(newResources.Resources, resource)
		processedFiles++
		progressCb(filename, info.Size(), info.Size(), processedFiles, totalFiles)
		return nil
	})

	if err != nil {
		utils.LogError(fmt.Errorf("failed to process files: %v", err))
		errorCb("Failed to process files", err)
		return
	}

	// Sort resources: entries with URLs first, blank URLs at the bottom
	sortResourcesByURL(newResources)

	// Generate patches by comparing old and new resources
	generatePatches(resources, newResources)

	// Generate resource set hash after sorting
	newResources.ResourceSetHash = generateResourceSetHash(newResources)

	err = saveResourceSet(newResources, resourcesPath)
	if err != nil {
		utils.LogError(fmt.Errorf("failed to save resources.json: %v", err))
		errorCb("Failed to save resources.json", err)
		return
	}

	progressCb("", 0, 0, totalFiles, totalFiles)
	utils.LogMessage("Resource generation complete!")
	utils.LogMessage("New version: " + newResources.LocalVersion)
	utils.LogMessage("Total resources: " + fmt.Sprintf("%d", len(newResources.Resources)))

	// Notify developer about missing URLs if using a service
	if isUsingService && len(missingURLs) > 0 {
		utils.LogWarning("Found " + fmt.Sprintf("%d", len(missingURLs)) + " files with missing download URLs:")
		for _, filename := range missingURLs {
			utils.LogWarning("  - " + filename)
		}
		utils.LogWarning("These files have been placed at the bottom of the resources list.")
		utils.LogWarning("You may need to manually add download URLs for these files.")
	} else {
		utils.LogMessage("You are not using any service provider, so download URLs may have been left blank.")
		utils.LogMessage("You can rerun with the respective provider arguments to add them.")
		utils.LogMessage("Check out the Wiki on the download provider at: https://github.com/cosmiclabstudio/cargodrop/wiki")
	}

	utils.LogMessage("Saved to: " + resourcesPath)
	utils.LogMessage("Done! You may close this window.")
}

// sortResourcesByURL sorts resources so that entries with URLs come first, blank URLs at the bottom
func sortResourcesByURL(resources *parsers.ResourceSet) {
	var withURL, withoutURL []parsers.Resource

	for _, resource := range resources.Resources {
		if resource.URL != "" {
			withURL = append(withURL, resource)
		} else {
			withoutURL = append(withoutURL, resource)
		}
	}

	// Combine: URLs first, then blank URLs
	resources.Resources = append(withURL, withoutURL...)
}

func generateResourceSetHash(resources *parsers.ResourceSet) string {
	hash := sha1.New()
	for _, resource := range resources.Resources {
		hash.Write([]byte(resource.Path + resource.Hash))
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func saveResourceSet(resources *parsers.ResourceSet, outputPath string) error {
	data, err := json.MarshalIndent(resources, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(outputPath, data, 0644)
}

// generatePatches compares old and new resources to detect removed files
// and adds them to patches list while respecting the preserve list
func generatePatches(oldResources, newResources *parsers.ResourceSet) {
	// Initialize patches as empty array if null to avoid JSON null values
	if newResources.Patches == nil {
		newResources.Patches = []string{}
	}

	// Create a map of new resource paths for quick lookup
	newResourcePaths := make(map[string]bool)
	for _, resource := range newResources.Resources {
		newResourcePaths[resource.Path] = true
	}

	// Find files that were removed (exist in old but not in new)
	var newPatches []string
	for _, oldResource := range oldResources.Resources {
		// Check if file is removed (not in new resources)
		if !newResourcePaths[oldResource.Path] {
			// Check if it's not already in patches to avoid duplicates
			alreadyExists := false
			for _, existingPatch := range newResources.Patches {
				if existingPatch == oldResource.Path {
					alreadyExists = true
					break
				}
			}
			if !alreadyExists {
				newPatches = append(newPatches, oldResource.Path)
				utils.LogMessage("Added to patches: " + oldResource.Path)
			}
		}
	}

	// Append new patches to existing ones
	newResources.Patches = append(newResources.Patches, newPatches...)

	if len(newPatches) > 0 {
		utils.LogMessage(fmt.Sprintf("Generated %d new patches for removed files ", len(newPatches)))
	}
}
