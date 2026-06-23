package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx             context.Context
	voponoMu        sync.Mutex
	voponoProcesses map[string]*exec.Cmd
	processInfo     map[string]VoponoProcess
	stateMu         sync.Mutex
}

// VoponoProcess represents a tracked vopono process
type VoponoProcess struct {
	ID         string `json:"id"`
	AppName    string `json:"appName"`
	ConfigName string `json:"configName"`
	PID        int    `json:"pid"`
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{
		voponoProcesses: make(map[string]*exec.Cmd),
		processInfo:     make(map[string]VoponoProcess),
	}
}

// startup is called when the app starts
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.setupSystray(ctx)
	a.startNetMonitor()
	a.startStatusMonitor()
}

// shutdown is called when the app is closing
func (a *App) shutdown(ctx context.Context) {
	a.StopSingbox()

	a.voponoMu.Lock()
	procs := make(map[string]*exec.Cmd, len(a.voponoProcesses))
	for k, v := range a.voponoProcesses {
		procs[k] = v
	}
	a.voponoMu.Unlock()

	for _, cmd := range procs {
		if cmd.Process != nil {
			cmd.Process.Signal(syscall.SIGTERM)
		}
	}

	time.Sleep(500 * time.Millisecond)

	for _, cmd := range procs {
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
	}
}

// LaunchStrict launches an app inside a Vopono network namespace for strict isolation
func (a *App) LaunchStrict(processName string, tunnelLabel string) (VoponoProcess, error) {
	state, err := a.LoadState()
	if err != nil {
		return VoponoProcess{}, fmt.Errorf("load state: %w", err)
	}

	proxyMap := buildProxyMap(state.Proxies)
	proxy, ok := proxyMap[tunnelLabel]
	if !ok {
		return VoponoProcess{}, fmt.Errorf("no config found for tunnel %q", tunnelLabel)
	}

	configDir, err := os.UserConfigDir()
	if err != nil {
		return VoponoProcess{}, err
	}
	strictDir := filepath.Join(configDir, "xynet", "strict")
	if err := os.MkdirAll(strictDir, 0700); err != nil {
		return VoponoProcess{}, err
	}

	confPath := filepath.Join(strictDir, tunnelLabel+".conf")
	if err := os.WriteFile(confPath, []byte(proxy.Content), 0600); err != nil {
		return VoponoProcess{}, fmt.Errorf("write config: %w", err)
	}

	cmd := exec.Command("vopono", "exec", "--custom", confPath, processName)

	if err := cmd.Start(); err != nil {
		return VoponoProcess{}, fmt.Errorf("failed to start vopono: %v", err)
	}

	id := uuid.New().String()
	proc := VoponoProcess{
		ID:         id,
		AppName:    processName,
		ConfigName: tunnelLabel,
		PID:        cmd.Process.Pid,
	}

	a.voponoMu.Lock()
	a.voponoProcesses[id] = cmd
	a.processInfo[id] = proc
	a.voponoMu.Unlock()

	go func(processID string) {
		cmd.Wait()

		a.voponoMu.Lock()
		delete(a.voponoProcesses, processID)
		delete(a.processInfo, processID)
		a.voponoMu.Unlock()

		if a.ctx != nil {
			runtime.EventsEmit(a.ctx, "vopono-process-ended", processID)
		}
	}(id)

	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "vopono-process-started", proc)
	}

	return proc, nil
}

// KillVopono gracefully terminates a tracked vopono instance
func (a *App) KillVopono(id string) error {
	a.voponoMu.Lock()
	cmd, exists := a.voponoProcesses[id]
	a.voponoMu.Unlock()

	if !exists {
		return fmt.Errorf("process not found")
	}

	if cmd.Process == nil {
		return nil
	}

	cmd.Process.Signal(syscall.SIGTERM)

	done := make(chan struct{})
	go func() {
		cmd.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(3 * time.Second):
		return cmd.Process.Kill()
	}
}

// ListVoponoProcesses returns all currently running tracked processes
func (a *App) ListVoponoProcesses() []VoponoProcess {
	a.voponoMu.Lock()
	defer a.voponoMu.Unlock()

	var procs []VoponoProcess
	for _, p := range a.processInfo {
		procs = append(procs, p)
	}
	return procs
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

	content, err := os.ReadFile(selection)
	if err != nil {
		return ImportedProxy{}, err
	}

	name := filepath.Base(selection)
	return ImportedProxy{Name: name, Content: string(content)}, nil
}

// RoutingRule represents a process-to-tunnel mapping
type RoutingRule struct {
	ProcessName string `json:"processName"`
	TunnelID    string `json:"tunnelId"`
	TunnelLabel string `json:"tunnelLabel"`
	TunnelType  string `json:"tunnelType"`
}

