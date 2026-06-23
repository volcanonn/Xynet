package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// AppState represents the persisted state of the application
type AppState struct {
	CanvasElements []interface{}   `json:"canvasElements"`
	Proxies        []ImportedProxy `json:"proxies"`
	Settings       AppSettings     `json:"settings"`
}

// AppSettings represents global application settings
type AppSettings struct {
	DefaultInterface string `json:"defaultInterface"`
	Theme            string `json:"theme"`
	Backend          string `json:"backend"` // "singbox" (default) or "dae"
}

func getConfigPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	appDir := filepath.Join(configDir, "xynet")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		return "", err
	}
	return filepath.Join(appDir, "state.json"), nil
}

// LoadState loads the application state from disk
func (a *App) LoadState() (AppState, error) {
	a.stateMu.Lock()
	defer a.stateMu.Unlock()

	path, err := getConfigPath()
	if err != nil {
		return AppState{}, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return AppState{
				CanvasElements: []interface{}{},
				Proxies:        []ImportedProxy{},
			}, nil
		}
		return AppState{}, err
	}

	var state AppState
	if err := json.Unmarshal(data, &state); err != nil {
		return AppState{}, err
	}

	return state, nil
}

// SaveState saves the application state to disk using atomic write
func (a *App) SaveState(state AppState) error {
	a.stateMu.Lock()
	defer a.stateMu.Unlock()

	path, err := getConfigPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return err
	}

	return os.Rename(tmpPath, path)
}
