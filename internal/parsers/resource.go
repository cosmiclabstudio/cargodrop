package parsers

import (
	"encoding/json"
	"os"
)

type Resource struct {
	Path string `json:"path"`
	Hash string `json:"hash"`
	Size int64  `json:"size"`
	URL  string `json:"url"`
}

type ResourceSet struct {
	Name            string     `json:"name"`
	LocalVersion    string     `json:"version"`
	ResourceSetHash string     `json:"resource_set_hash"`
	Resources       []Resource `json:"resources"`
}

func LoadResource(path string) (*ResourceSet, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var rs ResourceSet
	if err := json.Unmarshal(data, &rs); err != nil {
		return nil, err
	}
	return &rs, nil
}
