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

	// Custom progress bar (responsive width, 6px height, custom text)
	bar := widget.NewProgressBar()
	bar.Resize(fyne.NewSize(0, 16))
	bar.SetValue(0.45)

	// additional text
	barLeft := canvas.NewText("Downloading [1.20.1] [Sinytra] Hybrid Aquatic 1.4.4.jar...", color.White)
	barRight := canvas.NewText("(280.40 KB/3.4 MB)", color.White)
	progressBar := container.NewStack(
		bar,
		container.NewHBox(
			barLeft,
			layout.NewSpacer(),
			barRight,
		),
	)

	// Log area (scrollable, read-only, white text, monospace font)
	logVBox := container.NewVBox()
	logScroller := container.NewVScroll(logVBox)
	logScroller.SetMinSize(fyne.NewSize(900, 460))

	// Simulate log file printing, 1 second per line
	go func() {
		if data, err := os.ReadFile("cmd/demofiles/demo_log.txt"); err == nil {
			for _, line := range strings.Split(string(data), "\n") {
				if line != "" {
					logText := canvas.NewText(line, color.White)
					logText.TextSize = 14
					logText.TextStyle = fyne.TextStyle{Monospace: true}
					logVBox.Add(logText)
					logScroller.ScrollToBottom()
				}
			}
		}
	}()

	// Credit (small, white, emoji, centered)
	credit := canvas.NewText("Made with ❤️ by Cosmic Lab Studio", color.White)
	credit.TextSize = 12

	resourceName := canvas.NewText("All Things Considered SMP", color.White)
	resourceName.TextSize = 12

	w.SetContent(container.NewVBox(
		logScroller,
		progressBar,
		container.NewHBox(
			resourceName,
			layout.NewSpacer(),
			credit,
		),
	))

	w.Resize(fyne.NewSize(900, 500))
	w.CenterOnScreen()

	return w
}
