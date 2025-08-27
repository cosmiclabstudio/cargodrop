package main

import (
	"fyne.io/fyne/v2/app"
	"github.com/cosmiclabstudio/cargodrop/internal/gui"
)

func main() {
	a := app.New()
	w := gui.NewMainWindow(a)
	w.ShowAndRun()
}
