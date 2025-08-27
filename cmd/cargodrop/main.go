package main

import (
	"flag"

	"fyne.io/fyne/v2/app"
	"github.com/cosmiclabstudio/cargodrop/internal/gui"
	"github.com/cosmiclabstudio/cargodrop/internal/parsers"
	"github.com/cosmiclabstudio/cargodrop/internal/utils"
	"github.com/cosmiclabstudio/cargodrop/internal/workers"
)

func main() {
	// Command line flags
	configPath := flag.String("config", "", "Path to config file")
	resourcesPath := flag.String("resources", "", "Path to resources file")
	flag.Parse()

	utils.LogMessage("Config file: " + *configPath)
	utils.LogMessage("Resources file: " + *resourcesPath)

	// Parse config and resources
	config, err := parsers.LoadConfig(*configPath)
	if err != nil {
		utils.LogError(err)
		return
	}
	resources, err := parsers.LoadResource(*resourcesPath)
	if err != nil {
		utils.LogError(err)
		return
	}

	a := app.New()
	mw := gui.NewMainWindow(a, config, resources)
	go workers.RunUpdateSequence(config, resources, ".", mw.UpdateProgress, mw.HandleError)
	mw.Window.ShowAndRun()
}
