package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Module struct {
	Name        string            `json:"name"`
	Variant     map[string]string `json:"variant,omitempty"`
	ID          string            `json:"test"`
	Status      string            `json:"status"`
	Result      string            `json:"result"`
	EventCounts map[string]int    `json:"event_counts"`
	LogURL      string            `json:"log_url"`
	Error       string            `json:"error,omitempty"`
}

type Summary struct {
	Plan    string   `json:"plan"`
	Passed  bool     `json:"passed"`
	Modules []Module `json:"modules"`
	Error   string   `json:"error,omitempty"`
}

func writeArtifact(dir, name string, data []byte) error {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	} else {
		return os.WriteFile(filepath.Join(dir, name), data, 0600)
	}
}

func writeJSON(dir, name string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	} else {
		return writeArtifact(dir, name, append(data, '\n'))
	}
}
