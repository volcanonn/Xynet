package main

import (
	"bufio"
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"text/template"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

var (
	singboxCmd      *exec.Cmd
	singboxMu       sync.Mutex
	intentionalStop bool

	daeLogCmd       *exec.Cmd
	daeLogMu        sync.Mutex

	//go:embed templates/dae.tmpl
	daeTemplateStr string
)

// Deploy sends routing rules to the configured backend
func (a *App) Deploy(rules []RoutingRule) error {
	state, err := a.LoadState()
	if err != nil {
		return fmt.Errorf("load state: %w", err)
	}

	backend := state.Settings.Backend
	if backend == "" {
		backend = "singbox"
	}

	switch backend {
	case "singbox":
		return a.deploySingbox(rules, state)
	case "dae":
		return a.deployDae(rules, state)
	default:
		return fmt.Errorf("unknown backend: %s", backend)
	}
}

// Undeploy stops the active backend
func (a *App) Undeploy() error {
	state, err := a.LoadState()
	if err != nil {
		return fmt.Errorf("load state: %w", err)
	}

	backend := state.Settings.Backend
	if backend == "" {
		backend = "singbox"
	}

	switch backend {
	case "singbox":
		return a.StopSingbox()
	case "dae":
		return a.stopDae()
	default:
		return nil
	}
}

// GetBackendStatus returns the status of the configured backend
func (a *App) GetBackendStatus() BackendStatus {
	state, err := a.LoadState()
	if err != nil {
		return BackendStatus{Backend: "singbox", Running: false, Status: "error"}
	}

	backend := state.Settings.Backend
	if backend == "" {
		backend = "singbox"
	}

	if backend == "singbox" {
		singboxMu.Lock()
		running := singboxCmd != nil && singboxCmd.Process != nil
		singboxMu.Unlock()
		status := "offline"
		if running {
			status = "active"
		}
		return BackendStatus{Backend: backend, Running: running, Status: status}
	}

	// dae uses systemd
	running := false
	out, err := exec.Command("systemctl", "is-active", "dae").Output()
	if err == nil && strings.TrimSpace(string(out)) == "active" {
		running = true
	}

	status := "offline"
	if running {
		status = "active"
	}

	return BackendStatus{Backend: backend, Running: running, Status: status}
}

// BackendStatus represents the current backend state
type BackendStatus struct {
	Backend string `json:"backend"`
	Running bool   `json:"running"`
	Status  string `json:"status"`
}

// deploySingbox generates a sing-box config and runs sing-box as a subprocess
func (a *App) deploySingbox(rules []RoutingRule, state AppState) error {
	singboxPath, err := exec.LookPath("sing-box")
	if err != nil {
		return fmt.Errorf("sing-box not found in PATH — install it first (https://sing-box.sagernet.org/installation/)")
	}

	proxyMap := buildProxyMap(state.Proxies)

	outbounds := []map[string]interface{}{
		{"type": "direct", "tag": "direct"},
		{"type": "block", "tag": "block"},
	}

	routeRules := []map[string]interface{}{}
	seenTunnels := map[string]bool{}

	for _, rule := range rules {
		if rule.TunnelType == "Bypass" {
			continue
		}

		if rule.TunnelType == "Block" {
			routeRules = append(routeRules, map[string]interface{}{
				"process_name": []string{rule.ProcessName},
				"outbound":     "block",
			})
			continue
		}

		outboundTag := rule.TunnelID

		if !seenTunnels[rule.TunnelID] {
			seenTunnels[rule.TunnelID] = true

			proxy, ok := proxyMap[rule.TunnelLabel]
				if !ok {
					return fmt.Errorf("no imported config found for tunnel %q — import it in the Proxies tab first", rule.TunnelLabel)
				}

				if rule.TunnelType == "WireGuard" {
					wg, err := ParseWireGuardConfig(proxy.Content)
					if err != nil {
						return fmt.Errorf("parse config %q: %w", rule.TunnelLabel, err)
					}

					ob := map[string]interface{}{
						"type":          "wireguard",
						"tag":           outboundTag,
						"local_address": wg.Address,
						"private_key":   wg.PrivateKey,
					}
					if wg.MTU > 0 {
						ob["mtu"] = wg.MTU
					}

					if len(wg.Peers) == 1 {
						p := wg.Peers[0]
						server, port := splitEndpoint(p.Endpoint)
						ob["server"] = server
						ob["server_port"] = port
						ob["peer_public_key"] = p.PublicKey
						if p.PresharedKey != "" {
							ob["pre_shared_key"] = p.PresharedKey
						}
					} else if len(wg.Peers) > 1 {
						var peersList []map[string]interface{}
						for _, p := range wg.Peers {
							server, port := splitEndpoint(p.Endpoint)
							peerObj := map[string]interface{}{
								"server":          server,
								"server_port":     port,
								"peer_public_key": p.PublicKey,
							}
							if p.PresharedKey != "" {
								peerObj["pre_shared_key"] = p.PresharedKey
							}
							peersList = append(peersList, peerObj)
						}
						ob["peers"] = peersList
					}
					outbounds = append(outbounds, ob)
				} else if rule.TunnelType == "Hysteria2" {
					// Parse Hysteria2 URI: hysteria2://auth@host:port/?sni=domain.com&insecure=1
					uri := strings.TrimSpace(proxy.Content)
					if !strings.HasPrefix(uri, "hysteria2://") {
						return fmt.Errorf("invalid hysteria2 URI for tunnel %q", rule.TunnelLabel)
					}
					
					parts := strings.SplitN(strings.TrimPrefix(uri, "hysteria2://"), "@", 2)
					if len(parts) != 2 {
						return fmt.Errorf("invalid hysteria2 URI format for tunnel %q", rule.TunnelLabel)
					}
					auth := parts[0]
					
					hostQuery := strings.SplitN(parts[1], "/", 2)
					server, port := splitEndpoint(hostQuery[0])
					
					sni := ""
					insecure := false
					if len(hostQuery) > 1 && strings.HasPrefix(hostQuery[1], "?") {
						query := hostQuery[1][1:]
						qParts := strings.Split(query, "&")
						for _, qp := range qParts {
							kv := strings.SplitN(qp, "=", 2)
							if len(kv) == 2 {
								if kv[0] == "sni" {
									sni = kv[1]
								} else if kv[0] == "insecure" && kv[1] == "1" {
									insecure = true
								}
							}
						}
					}

					ob := map[string]interface{}{
						"type":        "hysteria2",
						"tag":         outboundTag,
						"server":      server,
						"server_port": port,
						"password":    auth,
					}
					
					tls := map[string]interface{}{
						"enabled": true,
					}
					if sni != "" {
						tls["server_name"] = sni
					}
					if insecure {
						tls["insecure"] = true
					}
					ob["tls"] = tls
					
					outbounds = append(outbounds, ob)
				}
		}

		routeRules = append(routeRules, map[string]interface{}{
			"process_name": []string{rule.ProcessName},
			"outbound":     outboundTag,
		})
	}

	// Default fallback to direct
	routeRules = append(routeRules, map[string]interface{}{
		"outbound": "direct",
	})

	config := map[string]interface{}{
		"log": map[string]interface{}{
			"level": "info",
		},
		"inbounds": []map[string]interface{}{
			{
				"type":                       "tun",
				"tag":                        "tun-in",
				"interface_name":             "tun0",
				"inet4_address":              "172.19.0.1/30",
				"auto_route":                 true,
				"strict_route":               true,
				"sniff":                      true,
				"sniff_override_destination": true,
			},
		},
		"outbounds": outbounds,
		"route": map[string]interface{}{
			"rules":                 routeRules,
			"auto_detect_interface": true,
		},
		"dns": map[string]interface{}{
			"servers": []map[string]interface{}{
				{"tag": "dns-fakeip", "address": "fakeip"},
				{"tag": "dns-remote", "address": "https://1.1.1.1/dns-query", "detour": "direct"},
			},
			"rules": []map[string]interface{}{
				{"outbound": []string{"any"}, "server": "dns-fakeip"},
			},
			"fakeip": map[string]interface{}{
				"enabled":    true,
				"inet4_range": "198.18.0.0/15",
				"inet6_range": "fc00::/18",
			},
		},
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	configDir, err := os.UserConfigDir()
	if err != nil {
		return err
	}
	dir := filepath.Join(configDir, "xynet")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	configPath := filepath.Join(dir, "sing-box-config.json")
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	// Stop any existing sing-box process
	a.StopSingbox()
	
	singboxMu.Lock()
	intentionalStop = false
	singboxMu.Unlock()

	// Start sing-box as a subprocess with pkexec for TUN privileges
	cmd := exec.Command("pkexec", singboxPath, "run", "-c", configPath)
	
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start sing-box: %w", err)
	}

	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			a.EmitLog("sing-box", scanner.Text())
		}
	}()
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			a.EmitLog("sing-box", scanner.Text())
		}
	}()

	singboxMu.Lock()
	singboxCmd = cmd
	singboxMu.Unlock()

	// Monitor process in background
	go func() {
		cmd.Wait()
		singboxMu.Lock()
		isIntentional := intentionalStop
		if singboxCmd == cmd {
			singboxCmd = nil
		}
		singboxMu.Unlock()

		if !isIntentional && a.ctx != nil {
			time.Sleep(3 * time.Second)
			singboxMu.Lock()
			stillUnintentional := !intentionalStop
			noNewCmd := singboxCmd == nil
			singboxMu.Unlock()
			if stillUnintentional && noNewCmd {
				a.deploySingbox(rules, state)
			}
		}
	}()

	return nil
}

// RestartSingbox restarts sing-box (used internally by deploySingbox)
func (a *App) RestartSingbox() error {
	return nil // handled by deploySingbox directly
}

// StopSingbox stops the sing-box subprocess
func (a *App) StopSingbox() error {
	singboxMu.Lock()
	intentionalStop = true
	cmd := singboxCmd
	singboxMu.Unlock()

	if cmd == nil || cmd.Process == nil {
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
	case <-time.After(3 * time.Second):
		cmd.Process.Kill()
	}

	singboxMu.Lock()
	if singboxCmd == cmd {
		singboxCmd = nil
	}
	singboxMu.Unlock()

	return nil
}

// WriteSingboxConfig writes the given JSON string to the sing-box config file
func (a *App) WriteSingboxConfig(config string) error {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return err
	}
	dir := filepath.Join(configDir, "xynet")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	path := filepath.Join(dir, "sing-box-config.json")
	return os.WriteFile(path, []byte(config), 0600)
}

// deployDae generates a dae config and manages WireGuard interfaces
func (a *App) deployDae(rules []RoutingRule, state AppState) error {
	if _, err := exec.LookPath("dae"); err != nil {
		return fmt.Errorf("dae not found in PATH — install it first (https://github.com/daeuniverse/dae)")
	}

	proxyMap := buildProxyMap(state.Proxies)

	configDir, err := os.UserConfigDir()
	if err != nil {
		return err
	}
	wgDir := filepath.Join(configDir, "xynet", "wg-configs")
	if err := os.MkdirAll(wgDir, 0700); err != nil {
		return err
	}

	var routingLines []string
	var daeCmds []string

	fwmarkBase := uint32(0x4E01)
	tableBase := 100
	seenTunnels := map[string]uint32{}

	for _, rule := range rules {
		switch rule.TunnelType {
		case "Bypass":
			continue
		case "Block":
			routingLines = append(routingLines, fmt.Sprintf("    pname(%s) -> block", rule.ProcessName))
		default:
			fwmark, ok := seenTunnels[rule.TunnelID]
			if !ok {
				fwmark = fwmarkBase + uint32(len(seenTunnels))
				seenTunnels[rule.TunnelID] = fwmark

				proxy, found := proxyMap[rule.TunnelLabel]
				if !found {
					return fmt.Errorf("no imported config found for tunnel %q — import it in the Proxies tab first", rule.TunnelLabel)
				}

				if rule.TunnelType == "Hysteria2" {
					return fmt.Errorf("dae backend does not support Hysteria2. Please switch to sing-box in Settings")
				}

				confPath := filepath.Join(wgDir, rule.TunnelLabel+".conf")
				if err := os.WriteFile(confPath, []byte(proxy.Content), 0600); err != nil {
					return fmt.Errorf("write wg config: %w", err)
				}

				table := tableBase + len(seenTunnels) - 1
				ifName := rule.TunnelLabel

				daeCmds = append(daeCmds,
					fmt.Sprintf("wg-quick down %q 2>/dev/null || true", confPath),
					fmt.Sprintf("wg-quick up %q", confPath),
					fmt.Sprintf("ip rule del fwmark 0x%x table %d 2>/dev/null || true", fwmark, table),
					fmt.Sprintf("ip rule add fwmark 0x%x table %d", fwmark, table),
					fmt.Sprintf("ip route add default dev %q table %d 2>/dev/null || true", ifName, table),
				)
			}

			routingLines = append(routingLines, fmt.Sprintf("    pname(%s) -> direct(mark: 0x%x)", rule.ProcessName, fwmark))
		}
	}

	routingLines = append(routingLines, "    fallback: direct")

	tmpl, err := template.New("dae").Parse(daeTemplateStr)
	if err != nil {
		return err
	}
	var rendered bytes.Buffer
	if err := tmpl.Execute(&rendered, map[string]interface{}{"Routing": strings.Join(routingLines, "\n")}); err != nil {
		return err
	}

	daeConfigPath := filepath.Join(configDir, "xynet", "config.dae")
	if err := os.WriteFile(daeConfigPath, rendered.Bytes(), 0644); err != nil {
		return fmt.Errorf("write dae config: %w", err)
	}

	daeCmds = append(daeCmds,
		"mkdir -p /etc/dae",
		fmt.Sprintf("cp %q /etc/dae/config.dae", daeConfigPath),
		"systemctl reload-or-restart dae",
	)

	if len(daeCmds) > 0 {
		cmd := exec.Command("pkexec", "sh", "-c", strings.Join(daeCmds, " && "))
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("dae deploy commands failed: %s (%w)", string(out), err)
		}
	}

	daeLogMu.Lock()
	if daeLogCmd != nil && daeLogCmd.Process != nil {
		daeLogCmd.Process.Kill()
	}
	daeLogCmd = exec.Command("journalctl", "-u", "dae", "-f", "-n", "0")
	stdout, _ := daeLogCmd.StdoutPipe()
	if err := daeLogCmd.Start(); err == nil {
		go func() {
			scanner := bufio.NewScanner(stdout)
			for scanner.Scan() {
				a.EmitLog("dae", scanner.Text())
			}
			daeLogCmd.Wait()
		}()
	}
	daeLogMu.Unlock()

	return nil
}

func (a *App) stopDae() error {
	daeLogMu.Lock()
	if daeLogCmd != nil && daeLogCmd.Process != nil {
		daeLogCmd.Process.Kill()
		daeLogCmd = nil
	}
	daeLogMu.Unlock()

	configDir, err := os.UserConfigDir()
	if err != nil {
		return err
	}

	var cmds []string
	cmds = append(cmds, "systemctl stop dae || true")

	wgDir := filepath.Join(configDir, "xynet", "wg-configs")
	if entries, err := os.ReadDir(wgDir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".conf") {
				confPath := filepath.Join(wgDir, entry.Name())
				cmds = append(cmds, fmt.Sprintf("wg-quick down %q 2>/dev/null || true", confPath))
			}
		}
	}

	cmds = append(cmds, "ip rule flush fwmark 0x4E01/0xFFFF 2>/dev/null || true")

	exec.Command("pkexec", "sh", "-c", strings.Join(cmds, " ; ")).Run()
	return nil
}

// buildProxyMap maps tunnel labels to their proxy configs
func buildProxyMap(proxies []ImportedProxy) map[string]ImportedProxy {
	m := make(map[string]ImportedProxy)
	for _, p := range proxies {
		label := strings.TrimSuffix(strings.TrimSuffix(p.Name, ".conf"), ".txt")
		m[label] = p
		m[p.Name] = p
	}
	return m
}

// splitEndpoint splits "host:port" into host and port
func splitEndpoint(endpoint string) (string, int) {
	host, portStr, err := net.SplitHostPort(endpoint)
	if err != nil {
		return endpoint, 51820
	}
	port := 51820
	fmt.Sscanf(portStr, "%d", &port)
	return host, port
}
