package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	libraryRoot   = "library"
	libraryConfig = "library.json"
)

type Heliport struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Location struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Heliports []Heliport `json:"heliports"`
}

type Helicopter struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Locations []Location `json:"locations"`
}

type Library struct {
	Helicopters []Helicopter `json:"helicopters"`
}

func LoadLibrary() (Library, error) {
	data, err := os.ReadFile(libraryConfig)
	if err != nil {
		return Library{}, fmt.Errorf("načtení %s: %w", libraryConfig, err)
	}

	var lib Library
	if err := json.Unmarshal(data, &lib); err != nil {
		return Library{}, fmt.Errorf("dekódování %s: %w", libraryConfig, err)
	}
	return lib, nil
}

func SaveLibrary(lib Library) error {
	data, err := json.MarshalIndent(lib, "", "  ")
	if err != nil {
		return fmt.Errorf("serializace %s: %w", libraryConfig, err)
	}
	if err := os.WriteFile(libraryConfig, data, 0644); err != nil {
		return fmt.Errorf("zápis %s: %w", libraryConfig, err)
	}
	return nil
}

func RecordingDir(heliID, locID, heliportID string) string {
	return filepath.Join(libraryRoot, heliID, locID, heliportID)
}

func ListRecordings(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(strings.ToLower(name), ".json") {
			files = append(files, filepath.Join(dir, name))
		}
	}
	sort.Strings(files)
	return files, nil
}

func EnsureDir(path string) error {
	return os.MkdirAll(path, 0755)
}

func NewRecordingPath(dir string) string {
	name := fmt.Sprintf("flight_%s.json", time.Now().Format("20060102_150405"))
	return filepath.Join(dir, name)
}

func recordingDisplayName(path string) string {
	return filepath.Base(path)
}
