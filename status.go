package main

import (
	"os/exec"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type ServiceStatus struct {
	SingboxRunning bool `json:"singboxRunning"`
	VoponoCount    int  `json:"voponoCount"`
}

func (a *App) GetServiceStatus() ServiceStatus {
	singboxRunning := false
	out, err := exec.Command("systemctl", "is-active", "sing-box").Output()
	if err == nil && strings.TrimSpace(string(out)) == "active" {
		singboxRunning = true
	}

	a.voponoMu.Lock()
	voponoCount := len(a.processInfo)
	a.voponoMu.Unlock()

	return ServiceStatus{
		SingboxRunning: singboxRunning,
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
			time.Sleep(3 * time.Second)
		}
	}()
}
