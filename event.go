package main

import (
	"encoding/json"
	"fmt"
	"os"
)

// Event reprezentuje jeden klávesový vstup (DOWN nebo UP) včetně času od startu nahrávky.
type Event struct {
	Key  string `json:"key"`
	Type string `json:"type"`
	At   int64  `json:"at"`
}

// Recorder schromažďuje všechny eventy nahrávky.
type Recorder struct {
	Events []Event `json:"events"`
}

// LoadRecording načte JSON soubor a vrátí Recorder.
func LoadRecording(path string) (Recorder, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Recorder{}, fmt.Errorf("načtení souboru: %w", err)
	}

	var rec Recorder
	if err := json.Unmarshal(data, &rec); err != nil {
		return Recorder{}, fmt.Errorf("dekódování JSON: %w", err)
	}
	return rec, nil
}

// SaveRecording uloží Recorder jako JSON na danou cestu.
func SaveRecording(path string, recorder Recorder) error {
	data, err := json.Marshal(recorder)
	if err != nil {
		return fmt.Errorf("serializace JSON: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("zápis souboru: %w", err)
	}
	return nil
}
