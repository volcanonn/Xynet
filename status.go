package main

import (
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type ServiceStatus struct {
	BackendRunning bool   `json:"backendRunning"`
	BackendName    string `json:"backendName"`
	VoponoCount    int    `json:"voponoCount"`
}

func (a *App) GetServiceStatus() ServiceStatus {
	bStatus := a.GetBackendStatus()
	
	a.voponoMu.Lock()
	voponoCount := len(a.processInfo)
	a.voponoMu.Unlock()

	return ServiceStatus{
		BackendRunning: bStatus.Running,
		BackendName:    bStatus.Backend,
		VoponoCount:    voponoCount,
	}
}

func (a *App) startStatusMonitor() {
	go func() {
		for {
			if a.ctx != nil {
				select {
				case <-a.ctx.Done():
					return
				default:
				}
				status := a.GetServiceStatus()
				runtime.EventsEmit(a.ctx, "service-status", status)
			}
			time.Sleep(2 * time.Second)
		}
	}()
}
