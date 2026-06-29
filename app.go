package main

import (
	"bufio"
	"context"
	"net"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type LogEntry struct {
	Timestamp string `json:"timestamp"`
	Source    string `json:"source"`
	Message   string `json:"message"`
}

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
func (a *App) EmitLog(source, msg string) {
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "app-log", LogEntry{
			Timestamp: time.Now().Format("15:04:05"),
			Source:    source,
			Message:   msg,
		})
	}
}

func (a *App) GetInterfaces() []string {
	var interfaceNames []string
	interfaces, err := net.Interfaces()
	if err != nil {
		return interfaceNames
	}
	for _, i := range interfaces {
		interfaceNames = append(interfaceNames, i.Name)
	}
	return interfaceNames
}

func (a *App) shutdown(ctx context.Context) {
	a.StopSingbox()

	// Tear down the dae backend too: stopDae suspends the daemon and runs
	// wg-quick down + ip rule flush via pkexec (best-effort on quit — we
	// ignore the error since we're exiting). Without this, quitting on the
	// dae backend leaks the root daemon, xywg* interfaces, and ip rule
	// entries. See INTERNALS.md "Zombie Process Bug".
	a.stopDae()

	// stopDae only suspends the daemon; kill the tracked process so it
	// doesn't linger as a root subprocess after exit.
	daeLogMu.Lock()
	daeProc := daeCmd
	daeLogMu.Unlock()
	if daeProc != nil && daeProc.Process != nil {
		daeProc.Process.Signal(syscall.SIGTERM)
	}

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
	a.voponoMu.Lock()
	for _, p := range a.processInfo {
		if p.AppName == processName && p.ConfigName == tunnelLabel {
			a.voponoMu.Unlock()
			return VoponoProcess{}, fmt.Errorf("app %q is already running in strict mode with tunnel %q", processName, tunnelLabel)
		}
	}
	a.voponoMu.Unlock()

	state, err := a.LoadState()
	if err != nil {
		return VoponoProcess{}, fmt.Errorf("load state: %w", err)
	}

	var cmd *exec.Cmd

	if tunnelLabel == "Block" {
		// Native Linux unshare: map user to root and create empty netns to blackhole all traffic
		cmd = exec.Command("unshare", "-r", "-n", processName)
	} else {
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

		// Strip DNS from config to prevent wg-quick from invoking resolvconf inside the namespace
		reDNS := regexp.MustCompile("(?im)^DNS\\s*=.*$")
		wgContent := reDNS.ReplaceAllString(proxy.Content, "")

		// Resolve Endpoints to IP addresses to prevent vopono namespace catch-22
		reEndpoint := regexp.MustCompile("(?im)^Endpoint\\s*=\\s*(.*)$")
		wgContent = reEndpoint.ReplaceAllStringFunc(wgContent, func(m string) string {
			parts := strings.SplitN(m, "=", 2)
			if len(parts) != 2 {
				return m
			}
			endpointStr := strings.TrimSpace(parts[1])
			host, portStr, err := net.SplitHostPort(endpointStr)
			if err != nil {
				host = endpointStr
				portStr = "51820"
			}
			ips, err := net.LookupIP(host)
			if err == nil && len(ips) > 0 {
				ip := ips[0].String()
				for _, i := range ips {
					if i.To4() != nil {
						ip = i.String()
						break
					}
				}
				return fmt.Sprintf("Endpoint = %s:%s", ip, portStr)
			}
			return m
		})

		if err := os.WriteFile(confPath, []byte(wgContent), 0600); err != nil {
			return VoponoProcess{}, fmt.Errorf("write config: %w", err)
		}

		// Ensure vopono executes the application as the normal user, otherwise gui apps like firefox will fail to launch X11/Wayland displays
		// pkexec strips environment variables, so we explicitly inject X11/Wayland vars using env
		envVars := []string{
			"DISPLAY", "WAYLAND_DISPLAY", "XAUTHORITY", "XDG_RUNTIME_DIR",
			"XDG_CONFIG_HOME", "XDG_SESSION_TYPE", "MOZ_ENABLE_WAYLAND",
			"QT_QPA_PLATFORM", "GTK_MODULES", "GTK3_MODULES", "I3SOCK",
			"HOME", "USER", "LOGNAME", "PWD", "XDG_CURRENT_DESKTOP", "XDG_SESSION_DESKTOP",
			"HYPRLAND_INSTANCE_SIGNATURE", "LANG", "LC_ALL", "PATH", "TERM", "COLORTERM",
		}

		var envArgs []string
		for _, env := range envVars {
			if val := os.Getenv(env); val != "" {
				envArgs = append(envArgs, fmt.Sprintf("%s=%s", env, val))
			}
		}

		var fullCommand string
		if len(envArgs) > 0 {
			fullCommand = fmt.Sprintf("env %s %s", strings.Join(envArgs, " "), processName)
		} else {
			fullCommand = processName
		}

		currentUser := os.Getenv("USER")
		if currentUser == "" {
			currentUser = "nobody"
		}

		wg, _ := ParseWireGuardConfig(proxy.Content)
		var dns string
		if len(wg.DNS) > 0 {
			dns = wg.DNS[0]
		} else {
			dns = "1.1.1.1" // Fallback to Cloudflare if config has no DNS
		}

		workingDir := os.Getenv("HOME")
		if workingDir == "" {
			workingDir = "/"
		}

		cmd = exec.Command("pkexec", "vopono", "exec", "--custom", confPath, "--firewall", "iptables", "--disable-ipv6", "--user", currentUser, "--working-directory", workingDir, "--dns", dns, fullCommand)
	}

	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

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
		go func() {
			scanner := bufio.NewScanner(stdout)
			for scanner.Scan() {
				a.EmitLog("vopono-"+processName, scanner.Text())
			}
		}()
		go func() {
			scanner := bufio.NewScanner(stderr)
			for scanner.Scan() {
				a.EmitLog("vopono-"+processName, scanner.Text())
			}
		}()

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
	if exists {
		delete(a.voponoProcesses, id)
		delete(a.processInfo, id)
	}
	a.voponoMu.Unlock()

	if !exists {
		return fmt.Errorf("process not found")
	}

	if cmd.Process == nil {
		return nil
	}

	cmd.Process.Signal(syscall.SIGTERM)

	for i := 0; i < 30; i++ {
		if err := cmd.Process.Signal(syscall.Signal(0)); err != nil {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	return cmd.Process.Kill()
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

// MeasureLatency attempts to ping the endpoint in the config
func (a *App) MeasureLatency(configContent string) string {
	var endpoint string

	// Try to parse as WireGuard
	if wg, err := ParseWireGuardConfig(configContent); err == nil && len(wg.Peers) > 0 && wg.Peers[0].Endpoint != "" {
		endpoint, _ = splitEndpoint(wg.Peers[0].Endpoint)
	} else if strings.HasPrefix(configContent, "hysteria2://") {
		// Hysteria2 URI format: hysteria2://auth@host:port/...
		parts := strings.Split(configContent, "@")
		if len(parts) > 1 {
			hostPort := strings.Split(parts[1], "/")
			endpoint, _ = splitEndpoint(hostPort[0])
		}
	}

	if endpoint == "" {
		return "--"
	}

	// Simple ICMP ping using the ping command
	out, err := exec.Command("ping", "-c", "1", "-W", "1", endpoint).CombinedOutput()
	if err != nil {
		return "Timeout"
	}

	// Extract time=X ms
	outStr := string(out)
	idx := strings.Index(outStr, "time=")
	if idx != -1 {
		timeStr := outStr[idx+5:]
		endIdx := strings.Index(timeStr, " ms")
		if endIdx != -1 {
			return timeStr[:endIdx] + "ms"
		}
	}

	return "--"
}

// ImportedProxy represents an imported configuration file
type ImportedProxy struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

// ImportWireguardConfig opens a file dialog to select a wireguard config file
func (a *App) ImportProxyConfigs() ([]ImportedProxy, error) {
	selections, err := runtime.OpenMultipleFilesDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Proxy Configs",
		Filters: []runtime.FileFilter{
			{
				DisplayName: "Proxy Configs (*.conf, *.txt)",
				Pattern:     "*.conf;*.txt",
			},
		},
	})
	if err != nil || len(selections) == 0 {
		return nil, err
	}

	var imported []ImportedProxy
	var errs []string

	for _, selection := range selections {
		content, err := os.ReadFile(selection)
		if err != nil {
			errs = append(errs, fmt.Sprintf("Failed to read %s: %v", filepath.Base(selection), err))
			continue
		}

		contentStr := string(content)

		// Validate content
		if _, err := ParseWireGuardConfig(contentStr); err != nil && !strings.HasPrefix(strings.TrimSpace(contentStr), "hysteria2://") {
			errs = append(errs, fmt.Sprintf("Invalid format for %s: must be valid WireGuard .conf or hysteria2:// URI", filepath.Base(selection)))
			continue
		}

		name := filepath.Base(selection)
		imported = append(imported, ImportedProxy{Name: name, Content: contentStr})
	}

	if len(imported) == 0 && len(errs) > 0 {
		return nil, fmt.Errorf("%s", strings.Join(errs, "\n"))
	}

	return imported, nil
}

// RoutingRule represents a process-to-tunnel mapping
type RoutingRule struct {
	ProcessName string `json:"processName"`
	TunnelID    string `json:"tunnelId"`
	TunnelLabel string `json:"tunnelLabel"`
	TunnelType  string `json:"tunnelType"`
}

