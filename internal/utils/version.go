package utils

const (
	Version = "1.0"
)

// GetFullVersionString returns the version with build hash for GUI display
func GetFullVersionString() string {
	return "CargoDrop ver." + Version
}

func GetProgramInformation() {
	LogMessage(GetFullVersionString())
	LogMessage("By using this program, you agree to the License and Terms of Conditions.")
	LogMessage("Available at https://https://github.com/cosmiclabstudio/cargodrop")
}
