package utils

const (
	Version = "1.2"
)

// GetFullVersionString returns the version with build hash for GUI display
func GetFullVersionString() string {
	return "CargoDrop ver." + Version
}

func GetProgramInformation() {
	LogMessage(GetFullVersionString())
	LogMessage("By using this program, you agree to the License that comes with the program.")
	LogMessage("Available at https://github.com/cosmiclabstudio/cargodrop")
}
