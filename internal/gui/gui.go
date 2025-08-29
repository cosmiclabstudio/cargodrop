package gui

import (
	"fmt"
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/cosmiclabstudio/cargodrop/internal/parsers"
	"github.com/cosmiclabstudio/cargodrop/internal/utils"
)

// apparently i want to use multiline input but the framework said my stuff my rule
type customTheme struct{}

func (t *customTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	if name == theme.ColorNameDisabled {
		return color.White
	}
	return theme.DefaultTheme().Color(name, variant)
}

func (t *customTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (t *customTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (t *customTheme) Size(name fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(name)
}

// MainWindow wraps fyne.Window to provide additional functionality
type MainWindow struct {
	Window         fyne.Window
	UpdateProgress func(fileName string, downloadedBytes, totalBytes int64, processed, total int)
	HandleError    func(message string, err error)
}

// NewMainWindow creates and returns the main window for the app
func NewMainWindow(a fyne.App, config *parsers.Config, resources *parsers.ResourceSet) *MainWindow {
	// Apply custom theme to keep disabled text white
	a.Settings().SetTheme(&customTheme{})

	w := a.NewWindow(config.Name + " Updater")
	w.SetMaster()
	w.Resize(fyne.NewSize(900, 500))
	w.CenterOnScreen()

	// Custom progress bar (responsive width, 6px height, custom text)
	bar := widget.NewProgressBar()
	bar.Resize(fyne.NewSize(0, 16))
	bar.SetValue(0)

	barLeft := canvas.NewText("Please wait!", color.White)
	barRight := canvas.NewText("(0/0)", color.White)
	progressBar := container.NewStack(
		bar,
		container.NewHBox(
			barLeft,
			layout.NewSpacer(),
			barRight,
		),
	)

	// Log area using MultiLineEntry with disabled state but white text
	logEntry := widget.NewMultiLineEntry()
	logEntry.Wrapping = fyne.TextWrapWord
	logEntry.TextStyle = fyne.TextStyle{Monospace: true}
	logEntry.Disable()
	logEntry.Text = ""

	// Function to append log lines to the log area and update barLeft
	appendLog := func(line string) {
		fyne.Do(func() {
			line = strings.TrimRight(line, "\n")
			logEntry.SetText(logEntry.Text + line + "\n")
			logEntry.CursorRow = len(strings.Split(logEntry.Text, "\n"))
			logEntry.Refresh()

			if strings.Contains(line, "Processing: ") || strings.Contains(line, "Downloading: ") {
				parts := strings.SplitN(line, " ", 3)
				parts2 := strings.Split(parts[2], " (")
				barLeft.Text = parts2[0]
			} else {
				barLeft.Text = "Please wait!"
			}
			barLeft.Refresh()
		})
	}

	// Register GUI log callback
	utils.RegisterGuiLogCallback(appendLog)

	// Progress update function
	updateProgress := func(fileName string, downloadedBytes, totalBytes int64, processed, total int) {
		fyne.Do(func() {
			var percent float64
			if total > 0 {
				percent = float64(processed) / float64(total)
			} else {
				percent = 1.0
			}
			bar.SetValue(percent)

			// If totalBytes is 0, we're in generating metadata mode, show file count instead
			if totalBytes == 0 {
				barRight.Text = fmt.Sprintf("%d/%d", processed, total)
			} else {
				// In updater mode, show download progress
				barRight.Text = utils.FormatSize(downloadedBytes) + "/" + utils.FormatSize(totalBytes)
			}
			barRight.Refresh()
		})
	}

	// Error handling function
	handleError := func(message string, err error) {
		fyne.Do(func() {
			dialog.ShowInformation(
				"Failed to update resources",
				"Please report this issue to your server administrator along with the log.\nYou may close this window to continue launching your game.",
				w)
		})

		utils.LogMessage("Failed to update resources...")
		utils.LogMessage("You may close this window to continue launching your game.")
	}

	creditLeft := canvas.NewText(utils.GetFullVersionString(), color.White)
	creditLeft.TextSize = 12
	creditRight := canvas.NewText("Made with ❤️ by Cosmic Lab Studio", color.White)
	creditRight.TextSize = 12

	// Bottom section with progress and credits
	bottomSection := container.NewVBox(
		progressBar,
		container.NewHBox(
			creditLeft,
			layout.NewSpacer(),
			creditRight,
		),
	)

	// Use BorderContainer to give log area proper space
	mainContainer := container.NewBorder(
		nil,           // top
		bottomSection, // bottom
		nil,           // left
		nil,           // right
		logEntry,      // center (gets remaining space)
	)

	w.SetContent(mainContainer)

	w.Resize(fyne.NewSize(900, 500))
	w.CenterOnScreen()

	mw := &MainWindow{
		Window:         w,
		UpdateProgress: updateProgress,
		HandleError:    handleError,
	}

	return mw
}
