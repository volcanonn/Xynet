package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
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

// GenerateSingboxConfig generates a base sing-box configuration with TUN, fakeip, and process_name rules.
func (a *App) GenerateSingboxConfig(apps []map[string]string) string {
	return `{
  "dns": {
    "fakeip": {
      "enabled": true,
      "inet4_range": "198.18.0.0/15"
    }
  },
  "inbounds": [
    {
      "type": "tun",
      "tag": "tun-in",
      "interface_name": "tun0",
      "inet4_address": "172.19.0.1/30",
      "auto_route": true,
      "strict_route": true
    }
  ],
  "route": {
    "rules": [
      {
        "process_name": ["firefox", "steam"],
        "outbound": "proxy"
      }
    ]
  }
}`
}

// WriteConfig writes the configuration to a file.
func (a *App) WriteConfig(path string, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}

// RestartSingbox restarts the sing-box service.
func (a *App) RestartSingbox() error {
	cmd := exec.Command("systemctl", "restart", "sing-box")
	return cmd.Run()
}

// RunVopono executes a vopono command to run an app in a network namespace.
func (a *App) RunVopono(appName string, network string) error {
	cmd := exec.Command("vopono", "exec", network, appName)
	return cmd.Start()
}
