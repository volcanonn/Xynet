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

// ImportWireguardConfig opens a file dialog to select a wireguard config file
func (a *App) ImportWireguardConfig() (string, error) {
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
		return "", err
	}

	// Read the file content
	content, err := os.ReadFile(selection)
	return string(content), err
}

// WgConfig represents a WireGuard configuration file
type WgConfig struct {
	Name      string `json:"name"`
	IsAirvpn  bool   `json:"isAirvpn"`
	UsageData string `json:"usageData,omitempty"`
}

// GetWireguardConfigs returns a list of parsed wireguard configurations
func (a *App) GetWireguardConfigs() []WgConfig {
	return []WgConfig{
		{Name: "wg0", IsAirvpn: false},
		{Name: "airvpn_nl", IsAirvpn: true, UsageData: "12.4 GB / 50 GB"},
		{Name: "airvpn_us", IsAirvpn: true, UsageData: "3.1 GB / 50 GB"},
	}
}
