package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx context.Context
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// Greet returns a greeting for the given name
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}

// WriteSingboxConfig writes the given JSON string to the sing-box config file
func (a *App) WriteSingboxConfig(config string) error {
	path := filepath.Join(os.TempDir(), "sing-box-config.json")
	return os.WriteFile(path, []byte(config), 0644)
}

// RestartSingbox restarts the sing-box daemon
func (a *App) RestartSingbox() error {
	cmd := exec.Command("systemctl", "restart", "sing-box")
	return cmd.Run()
}

// ExecVopono executes a vopono command
func (a *App) ExecVopono(appName string, configName string) (string, error) {
	cmd := exec.Command("vopono", "exec", configName, appName)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// ImportedProxy represents an imported configuration file
type ImportedProxy struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

// ImportWireguardConfig opens a file dialog to select a wireguard config file
func (a *App) ImportWireguardConfig() (ImportedProxy, error) {
	selection, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Wireguard Config",
		Filters: []runtime.FileFilter{
			{
				DisplayName: "Wireguard Config (*.conf)",
				Pattern:     "*.conf",
			},
		},
	})
	if err != nil || selection == "" {
		return ImportedProxy{}, err
	}

	// Read the file content
	content, err := os.ReadFile(selection)
	if err != nil {
		return ImportedProxy{}, err
	}

	name := filepath.Base(selection)
	return ImportedProxy{Name: name, Content: string(content)}, nil
}


