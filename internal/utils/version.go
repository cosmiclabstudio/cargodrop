package utils

const (
	Version = "1.0"
)

// BuildHash can be set at build time using ldflags
// Example: go build -ldflags "-X github.com/cosmiclabstudio/cargodrop/internal/utils.BuildHash=$(git rev-parse --short HEAD)"
var BuildHash = "dev"

// GetFullVersionString returns the version with build hash for GUI display
func GetFullVersionString() string {
	return "CargoDrop ver." + Version + " (build " + BuildHash + ")"
}
