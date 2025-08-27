package config

// Config holds application configuration.
// Expand this struct as needed (e.g., add paths, endpoints, credentials, etc.).
type Config struct {
	ModpackDir     string
	RemoteIndexURL string
}

// Load returns default or discovered configuration.
// In a real app, load from file (TOML/YAML), env vars, or flags.
func Load() *Config {
	return &Config{
		ModpackDir:     "",
		RemoteIndexURL: "",
	}
}
