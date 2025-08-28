package utils

import (
	"fmt"
	"os"
	"time"
)

const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[36m"
)

var guiLogCallback func(string)

// RegisterGuiLogCallback sets the callback for GUI logging
func RegisterGuiLogCallback(cb func(string)) {
	guiLogCallback = cb
}

// InitializeLog clears the log file at program startup
func InitializeLog() error {
	f, err := os.OpenFile("cargodrop.log", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	return f.Close()
}

// LogMessage logs an info message with timestamp in blue and to GUI if set.
func LogMessage(msg string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	formatted := fmt.Sprintf("[%s] %s", timestamp, msg)
	if guiLogCallback != nil {
		guiLogCallback(formatted)
	}
	_ = LogToFile("cargodrop.log", formatted)
}

// LogError logs an error message with timestamp in red and to GUI if set.
func LogError(err error) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	formatted := fmt.Sprintf("[%s] ERROR: %v", timestamp, err)
	if guiLogCallback != nil {
		guiLogCallback(formatted)
	}
	_ = LogToFile("cargodrop.log", formatted)
}

// LogWarning logs a warning message with timestamp in yellow and to GUI if set.
func LogWarning(msg string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	formatted := fmt.Sprintf("[%s] WARNING: %s", timestamp, msg)
	if guiLogCallback != nil {
		guiLogCallback(formatted)
	}
	_ = LogToFile("cargodrop.log", formatted)
}

// LogRaw logging raw without timestamp. This is only for the introduction and some field in the config
// never get saved into the logfile.
func LogRaw(msg string) {
	formatted := fmt.Sprintf("%s", msg)
	if guiLogCallback != nil {
		guiLogCallback(formatted)
	}
}

// LogToFile appends a log message to the specified file (no color).
func LogToFile(filename, msg string) error {
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	_, err = f.WriteString(fmt.Sprintf("%s\n", msg))
	return err
}
