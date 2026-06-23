package main

import (
	"os/exec"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type ServiceStatus struct {
	BackendRunning bool   `json:"backendRunning"`
	BackendName    string `json:"backendName"`
	VoponoCount    int    `json:"voponoCount"`
}

func (a *App) GetServiceStatus() ServiceStatus {
	state, _ := a.LoadState()
	backend := state.Settings.Backend
	if backend == "" {
		backend = "singbox"
	}

	serviceName := "sing-box"
	if backend == "dae" {
		serviceName = "dae"
	}

	backendRunning := false
	out, err := exec.Command("systemctl", "is-active", serviceName).Output()
	if err == nil && strings.TrimSpace(string(out)) == "active" {
		backendRunning = true
	}

	a.voponoMu.Lock()
	voponoCount := len(a.processInfo)
	a.voponoMu.Unlock()

	return ServiceStatus{
		BackendRunning: backendRunning,
		BackendName:    backend,
		VoponoCount:    voponoCount,
	}
}

func (a *App) startStatusMonitor() {
	go func() {
		for {
			if a.ctx != nil {
				status := a.GetServiceStatus()
				runtime.EventsEmit(a.ctx, "service-status", status)
			}
			time.Sleep(10 * time.Second)
		}
	}()
}
