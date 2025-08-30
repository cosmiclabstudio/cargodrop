package parsers

import (
	"encoding/json"
	"os"
)

type Config struct {
	Name               string   `json:"name"`
	WelcomeMessage     string   `json:"welcome_message"`
	Folders            []string `json:"folders"`
	Ignore             []string `json:"ignore"`
	UpdateServer       string   `json:"update_server"`
	Preserve           []string `json:"preserve,omitempty"`
	DisabledExtensions []string `json:"disabledExtensions,omitempty"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func SaveConfig(cfg *Config, path string) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
