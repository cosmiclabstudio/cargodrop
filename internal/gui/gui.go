package gui

import (
	"image/color"
	"os"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

// NewMainWindow creates and returns the main window for the app
func NewMainWindow(a fyne.App) fyne.Window {
	w := a.NewWindow("cargodrop updater")
	w.SetMaster()
	w.Resize(fyne.NewSize(900, 500))
	w.CenterOnScreen()

	// Progress bar (responsive width, 6px height, percentage text inside)
	progress := widget.NewProgressBar()
	progress.SetValue(0.45)
	progress.Resize(fyne.NewSize(0, 6))
	progress.TextFormatter = func() string {
		return "45% (23/92)"
	}

	progressBar := container.NewStack(progress)

	// Log area (scrollable, read-only, white text, monospace font)
	logVBox := container.NewVBox()
	if data, err := os.ReadFile("internal/gui/demo_log.txt"); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if line != "" {
				logText := canvas.NewText(line, color.White)
				logText.TextSize = 14
				logText.TextStyle = fyne.TextStyle{Monospace: true}
				logVBox.Add(logText)
			}
		}
	}
	logScroller := container.NewVScroll(logVBox)
	logScroller.SetMinSize(fyne.NewSize(900, 460))

	// Credit (small, white, emoji, centered)
	credit := canvas.NewText("Made with ❤️ by Cosmic Lab Studio", color.White)
	credit.TextSize = 12

	w.SetContent(container.NewVBox(
		logScroller,
		progressBar,
		container.NewHBox(
			layout.NewSpacer(),
			credit,
			layout.NewSpacer(),
		),
	))

	return w
}
