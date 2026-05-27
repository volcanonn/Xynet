package main

import (
	"context"
	"fmt"
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
	// A real implementation would serialize this to JSON properly
	// This is just a scaffolding string based on the ticket requirements.
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
