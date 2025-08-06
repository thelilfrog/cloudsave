package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type (
	Configuration struct {
		Server ServerConfiguration `json:"server"`
		Remote RemoteConfiguration `json:"remote"`
	}

	ServerConfiguration struct {
		Port int `json:"port"`
	}

	RemoteConfiguration struct {
		URL string `json:"url"`
	}
)

func Load(path string) (Configuration, error) {
	f, err := os.OpenFile(path, os.O_RDONLY, 0)
	if err != nil {
		return Configuration{}, fmt.Errorf("failed to open configuration file: %w", err)
	}
	defer f.Close()

	d := json.NewDecoder(f)

	var c Configuration
	err = d.Decode(&c)
	if err != nil {
		return Configuration{}, fmt.Errorf("failed to parse configuration file (%s): %w", path, err)
	}

	return c, nil
}
