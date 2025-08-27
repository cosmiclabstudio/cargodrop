package parsers

import (
	"encoding/json"
	"os"
)

type Config struct {
	Name           string   `json:"name"`
	WelcomeMessage string   `json:"welcome_message"`
	Folders        []string `json:"folders"`
	UpdateServer   string   `json:"update_server"`
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
