package parsers

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
)

type Resource struct {
	Filename string `json:"filename"`
	Location string `json:"location"`
	Hash     string `json:"hash"`
	Size     int64  `json:"size"`
}

type ResourceSet struct {
	Name            string     `json:"name"`
	LocalVersion    string     `json:"local_version"`
	ResourceSetHash string     `json:"resource_set_hash"`
	Resources       []Resource `json:"resources"`
}

type RemoteResource struct {
	Filename string `json:"filename"`
	Location string `json:"location"`
	Hash     string `json:"hash"`
	URL      string `json:"url"`
}

type RemoteResourceSet struct {
	Resources []RemoteResource `json:"resources"`
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

// LoadRemoteResource downloads and parses remote_resources.json from a URL
func LoadRemoteResource(url string) (*RemoteResourceSet, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var rs RemoteResourceSet
	if err := json.Unmarshal(data, &rs); err != nil {
		return nil, err
	}
	return &rs, nil
}
