package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx             context.Context
	voponoMu        sync.Mutex
	voponoProcesses map[string]*exec.Cmd
	processInfo     map[string]VoponoProcess
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

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.setupSystray(ctx)
	a.startNetMonitor()
	a.startStatusMonitor()
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

// StopSingbox stops the sing-box daemon
func (a *App) StopSingbox() error {
	cmd := exec.Command("systemctl", "stop", "sing-box")
	return cmd.Run()
}

// ExecVopono executes a vopono command asynchronously and tracks it
func (a *App) ExecVopono(appName string, configName string) (VoponoProcess, error) {
	cmd := exec.Command("vopono", "exec", configName, appName)

	// Start asynchronously to prevent UI blocking
	if err := cmd.Start(); err != nil {
		return VoponoProcess{}, fmt.Errorf("failed to start vopono: %v", err)
	}

	id := uuid.New().String()
	proc := VoponoProcess{
		ID:         id,
		AppName:    appName,
		ConfigName: configName,
		PID:        cmd.Process.Pid,
	}

	a.voponoMu.Lock()
	a.voponoProcesses[id] = cmd
	a.processInfo[id] = proc
	a.voponoMu.Unlock()

	// Wait for process to exit in the background to clean up
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

// KillVopono gracefully kills a tracked vopono instance
func (a *App) KillVopono(id string) error {
	a.voponoMu.Lock()
	cmd, exists := a.voponoProcesses[id]
	a.voponoMu.Unlock()

	if !exists {
		return fmt.Errorf("process not found")
	}

	if cmd.Process != nil {
		return cmd.Process.Kill()
	}
	return nil
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

	// Read the file content
	content, err := os.ReadFile(selection)
	if err != nil {
		return ImportedProxy{}, err
	}

	name := filepath.Base(selection)
	return ImportedProxy{Name: name, Content: string(content)}, nil
}


