package main

import (
	"encoding/json"
	"flag"
	"os"

	"fyne.io/fyne/v2/app"
	"github.com/cosmiclabstudio/cargodrop/internal/gui"
	"github.com/cosmiclabstudio/cargodrop/internal/parsers"
	"github.com/cosmiclabstudio/cargodrop/internal/utils"
	"github.com/cosmiclabstudio/cargodrop/internal/workers"
)

func main() {
	configPath := flag.String("config", "", "Path to config file")
	resourcesPath := flag.String("resources", "", "Path to resources file")
	isGenResource := flag.Bool("generate-metadata", false, "Whether to generate metadata for server")
	flag.Parse()

	utils.LogMessage("Config file: " + *configPath)
	utils.LogMessage("Resources file: " + *resourcesPath)

	config, err := parsers.LoadConfig(*configPath)
	if err != nil {
		utils.LogError(err)
		return
	}

	resources, err := parsers.LoadResource(*resourcesPath)
	if err != nil {
		resources = &parsers.ResourceSet{
			Name:            config.Name,
			LocalVersion:    "1.0.0",
			ResourceSetHash: "",
			Resources:       []parsers.Resource{},
		}

		// Save default resources file
		err = saveDefaultResourceSet(resources, *resourcesPath)
		if err != nil {
			utils.LogError(err)
			return
		}
		utils.LogWarning("Missing resource.json! Creating one at " + *resourcesPath)
	}

	a := app.New()
	mw := gui.NewMainWindow(a, config, resources)

	err = utils.InitializeLog()
	if err != nil {
		return
	}

	// Start processing in background goroutine
	go func() {
		if *isGenResource {
			utils.LogMessage("Generating metadata for server...")
			workers.RunGenSourceSequence(config, resources, ".", *resourcesPath, mw.UpdateProgress, mw.HandleError)
		} else {
			workers.RunUpdateSequence(config, resources, ".", mw.UpdateProgress, mw.HandleError)
		}
	}()

	mw.Window.ShowAndRun()
}

func saveDefaultResourceSet(resources *parsers.ResourceSet, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			utils.LogError(closeErr)
		}
	}()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ") // Pretty print JSON with indentation
	return encoder.Encode(resources)
}
