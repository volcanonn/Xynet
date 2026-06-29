package main

import (
	"bufio"
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"regexp"
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
	// singboxRestartCount tracks consecutive auto-restarts triggered by the
	// watchdog. It is reset to 0 on every user-initiated Deploy. Without a
	// cap, a fundamentally broken config loops forever — each iteration
	// prompting for pkexec privileges.
	singboxRestartCount int

	daeCmd          *exec.Cmd
	daeActive       bool
	daeLogMu        sync.Mutex

	//go:embed templates/dae.tmpl
	daeTemplateStr string
)

// maxSingboxRestarts bounds the watchdog's auto-heal attempts before it gives
// up, so a broken config cannot loop forever (and spam pkexec prompts).
const maxSingboxRestarts = 3

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

	// A user-initiated Deploy resets the watchdog restart budget — the user is
	// (re)deploying intentionally, so crashes from here count as a fresh cycle.
	singboxMu.Lock()
	singboxRestartCount = 0
	singboxMu.Unlock()

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

	daeLogMu.Lock()
	isProcessAlive := daeCmd != nil && daeCmd.Process != nil
	isActive := daeActive
	daeLogMu.Unlock()

	running := isProcessAlive && isActive
	status := "offline"
	
	if isProcessAlive && !isActive {
		status = "suspended"
	} else if running {
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

	// sing-box >= 1.11 moved WireGuard outbounds to top-level "endpoints".
	endpoints := []map[string]interface{}{}

	routeRules := []map[string]interface{}{
		// Modern (sing-box >= 1.11) rule actions: sniff protocols and hijack
		// DNS queries before per-process routing is evaluated.
		{"action": "sniff"},
		{"protocol": "dns", "action": "hijack-dns"},
	}
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

					// sing-box >= 1.11: WireGuard is a top-level endpoint, not an
					// outbound. Each address must carry an explicit prefix.
					var cleanAddrs []string
					for _, addr := range wg.Address {
						if strings.Contains(addr, "/") {
							cleanAddrs = append(cleanAddrs, addr)
						} else {
							if strings.Contains(addr, ":") {
								cleanAddrs = append(cleanAddrs, addr+"/128")
							} else {
								cleanAddrs = append(cleanAddrs, addr+"/32")
							}
						}
					}

					// Default AllowedIPs to full tunnel if the peer omits it.
					allowedIPs := func(p WireGuardPeer) []string {
						if len(p.AllowedIPs) > 0 {
							return p.AllowedIPs
						}
						return []string{"0.0.0.0/0", "::/0"}
					}

					var peersList []map[string]interface{}
					for _, p := range wg.Peers {
						server, port := splitEndpoint(p.Endpoint)
						peerObj := map[string]interface{}{
							"address":     server,
							"port":        port,
							"public_key":  p.PublicKey,
							"allowed_ips": allowedIPs(p),
						}
						if p.PresharedKey != "" {
							peerObj["pre_shared_key"] = p.PresharedKey
						}
						peersList = append(peersList, peerObj)
					}

					ep := map[string]interface{}{
						"type":        "wireguard",
						"tag":         outboundTag,
						"address":     cleanAddrs,
						"private_key": wg.PrivateKey,
						"peers":       peersList,
					}
					if wg.MTU > 0 {
						ep["mtu"] = wg.MTU
					}
					endpoints = append(endpoints, ep)
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

	inbounds := []map[string]interface{}{
		{
			// sing-box >= 1.10 merged inet4_address/inet6_address into "address".
			// sniff/hijack-dns are now route rule actions, not inbound flags.
			"type":           "tun",
			"tag":            "tun-in",
			"interface_name": "tun0",
			"address":        []string{"172.19.0.1/30", "fdfe:dcba:9876::1/126"},
			"auto_route":     true,
			"strict_route":   true,
		},
	}

	// sing-box >= 1.12: DNS servers use an explicit "type"; fakeip is a server
	// type, not a magic address + separate fakeip block.
	dnsServers := []map[string]interface{}{
		{"tag": "dns-remote", "type": "https", "server": "1.1.1.1", "detour": "direct"},
		{"tag": "dns-fakeip", "type": "fakeip", "inet4_range": "198.18.0.0/15", "inet6_range": "fc00::/18"},
	}

	config := map[string]interface{}{
		"log": map[string]interface{}{
			"level": "info",
		},
		"inbounds":  inbounds,
		"outbounds": outbounds,
		"route": map[string]interface{}{
			"rules":                   routeRules,
			"auto_detect_interface":   true,
			"default_domain_resolver": "dns-fakeip",
		},
		"dns": map[string]interface{}{
			"servers": dnsServers,
		},
	}

	if len(endpoints) > 0 {
		config["endpoints"] = endpoints
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

		if isIntentional {
			return
		}
		// App is shutting down — don't resurrect sing-box into a dying process.
		if a.ctx == nil || a.ctx.Err() != nil {
			return
		}

		time.Sleep(3 * time.Second)

		// Re-check after the sleep: the user may have clicked Disconnect or
		// quit the app during the grace window, either of which flips
		// intentionalStop or invalidates ctx.
		if a.ctx == nil || a.ctx.Err() != nil {
			return
		}
		singboxMu.Lock()
		stillUnintentional := !intentionalStop
		noNewCmd := singboxCmd == nil
		withinBudget := singboxRestartCount < maxSingboxRestarts
		if stillUnintentional && noNewCmd && withinBudget {
			// Consume one restart credit before recursing. The budget is only
			// replenished by an explicit user Deploy (which resets the count),
			// so a broken config loops at most maxSingboxRestarts times
			// rather than prompting for pkexec forever.
			singboxRestartCount++
			attempt := singboxRestartCount
			singboxMu.Unlock()
			a.EmitLog("sing-box", fmt.Sprintf("watchdog: process exited unexpectedly, auto-restarting (attempt %d/%d)", attempt, maxSingboxRestarts))
			if err := a.deploySingbox(rules, state); err != nil {
				a.EmitLog("sing-box", fmt.Sprintf("watchdog: restart failed: %v", err))
			}
		} else {
			singboxMu.Unlock()
			if !withinBudget {
				a.EmitLog("sing-box", fmt.Sprintf("watchdog: giving up after %d consecutive restarts — check your config in the Logs tab", maxSingboxRestarts))
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

	for i := 0; i < 30; i++ {
		if err := cmd.Process.Signal(syscall.Signal(0)); err != nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	if err := cmd.Process.Signal(syscall.Signal(0)); err == nil {
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
func (a *App) getDaePath() string {
	configDir, err := os.UserConfigDir()
	if err == nil {
		localDae := filepath.Join(configDir, "xynet", "bin", "dae")
		if _, err := os.Stat(localDae); err == nil {
			return localDae
		}
	}
	if path, err := exec.LookPath("dae"); err == nil {
		return path
	}
	return ""
}

func (a *App) deployDae(rules []RoutingRule, state AppState) error {
	daePath := a.getDaePath()
	if daePath == "" {
		return fmt.Errorf("dae backend is not installed. Please install it from Settings")
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

				safeIfName := fmt.Sprintf("xywg%d", len(seenTunnels))
				confPath := filepath.Join(wgDir, safeIfName+".conf")
				
				// Inject Table = off into the WireGuard config to prevent wg-quick from hijacking the global default route
				re := regexp.MustCompile("(?i)\\[Interface\\]")
				wgContent := re.ReplaceAllString(proxy.Content, "[Interface]\nTable = off")

				// Strip DNS to prevent wg-quick from overwriting global resolv.conf and breaking bypass traffic
				reDNS := regexp.MustCompile("(?im)^DNS\\s*=.*$")
				wgContent = reDNS.ReplaceAllString(wgContent, "")

				if err := os.WriteFile(confPath, []byte(wgContent), 0600); err != nil {
					return fmt.Errorf("write wg config: %w", err)
				}

				table := tableBase + len(seenTunnels) - 1

				daeCmds = append(daeCmds,
					fmt.Sprintf("wg-quick down %q 2>/dev/null || true", confPath),
					fmt.Sprintf("wg-quick up %q", confPath),
					fmt.Sprintf("ip rule del fwmark 0x%x table %d 2>/dev/null || true", fwmark, table),
					fmt.Sprintf("ip rule add fwmark 0x%x table %d", fwmark, table),
					fmt.Sprintf("ip route add default dev %q table %d 2>/dev/null || true", safeIfName, table),
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
	if err := os.WriteFile(daeConfigPath, rendered.Bytes(), 0600); err != nil {
		return fmt.Errorf("write dae config: %w", err)
	}
	os.Chmod(daeConfigPath, 0600) // Force chmod in case the file already existed with 0644

	daeLogMu.Lock()

	// Teardown of any previously-deployed interfaces/rules must NOT run under
	// `set -e` (the teardown lines are deliberately tolerant), so run it as a
	// separate best-effort chain before the strict setup. This guarantees an
	// incremental redeploy starts clean: removed tunnels' xywg* interfaces and
	// fwmark ip rules don't accumulate across deploys.
	if teardown := daeTeardownCmds(wgDir); len(teardown) > 0 {
		exec.Command("pkexec", "sh", "-c", strings.Join(teardown, " ; ")).Run()
	}

	var setupScript string
	if daeCmd != nil && daeCmd.Process != nil {
		// Bundle suspend, interface setup, and reload into ONE pkexec chain
		scriptLines := []string{
			"set -e",
			fmt.Sprintf("%q suspend", daePath),
		}
		scriptLines = append(scriptLines, daeCmds...)
		scriptLines = append(scriptLines, fmt.Sprintf("%q reload", daePath))

		setupScript = strings.Join(scriptLines, "\n")
		cmd := exec.Command("pkexec", "sh", "-c", setupScript)
		if out, err := cmd.CombinedOutput(); err != nil {
			daeLogMu.Unlock()
			return fmt.Errorf("dae reload chain failed: %s (%w)", string(out), err)
		}
	} else {
		// Bundle interface setup for the first time into ONE pkexec chain
		if len(daeCmds) > 0 {
			scriptLines := []string{"set -e"}
			scriptLines = append(scriptLines, daeCmds...)
			setupScript = strings.Join(scriptLines, "\n")

			// Try to avoid a double prompt by running sh -c directly, but pkexec caching is strict for binary paths.
			cmd := exec.Command("pkexec", "sh", "-c", setupScript)
			if out, err := cmd.CombinedOutput(); err != nil {
				daeLogMu.Unlock()
				return fmt.Errorf("dae interface setup failed: %s (%w)", string(out), err)
			}
		}

		// Then launch dae in background (this is the 2nd prompt on initial launch, but 0 extra prompts on subsequent deploys!)
		daeCmd = exec.Command("pkexec", daePath, "run", "-c", daeConfigPath)
		stdout, _ := daeCmd.StdoutPipe()
		stderr, _ := daeCmd.StderrPipe()
		
		if err := daeCmd.Start(); err != nil {
			daeLogMu.Unlock()
			return fmt.Errorf("failed to start dae: %w", err)
		}
		
		go func() {
			scanner := bufio.NewScanner(stdout)
			for scanner.Scan() {
				a.EmitLog("dae-engine", scanner.Text())
			}
		}()
		go func() {
			scanner := bufio.NewScanner(stderr)
			for scanner.Scan() {
				a.EmitLog("dae-engine", scanner.Text())
			}
		}()
		
		go func() {
			daeCmd.Wait()
			daeLogMu.Lock()
			daeCmd = nil
			daeActive = false
			daeLogMu.Unlock()
		}()
	}
	daeActive = true
	daeLogMu.Unlock()

	return nil
}

func (a *App) stopDae() error {
	daePath := a.getDaePath()
	
	daeLogMu.Lock()
	if daeCmd != nil && daeCmd.Process != nil && daePath != "" {
		// Suspend routes so traffic bypasses normally without tearing down the background daemon
		exec.Command("pkexec", daePath, "suspend").Run()
	}
	daeActive = false
	daeLogMu.Unlock()

	configDir, err := os.UserConfigDir()
	if err != nil {
		return err
	}

	wgDir := filepath.Join(configDir, "xynet", "wg-configs")
	cmds := daeTeardownCmds(wgDir)

	exec.Command("pkexec", "sh", "-c", strings.Join(cmds, " ; ")).Run()
	return nil
}

// daeTeardownCmds returns the privileged shell commands that tear down every
// Xynet-owned WireGuard interface (xywg*) and flush the fwmark ip rule range.
// It is shared by deployDae (so an incremental redeploy starts from a clean
// slate — removed tunnels no longer leave orphaned interfaces/rules) and by
// stopDae (full undeploy). Each line is tolerant of failure (|| true) so it
// works whether or not the interfaces/rules currently exist.
func daeTeardownCmds(wgDir string) []string {
	var cmds []string
	if entries, err := os.ReadDir(wgDir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".conf") {
				confPath := filepath.Join(wgDir, entry.Name())
				cmds = append(cmds, fmt.Sprintf("wg-quick down %q 2>/dev/null || true", confPath))
			}
		}
	}
	// deployDae assigns one fwmark per tunnel starting at fwmarkBase (0x4E01):
	// 0x4E01, 0x4E02, ... Flush the whole allocated range via the /0xFFFFFF00
	// mask so every Xynet-owned fwmark rule is removed (covers tunnels 2..N,
	// which the old 0x4E01-only flush left behind).
	cmds = append(cmds, "ip rule flush fwmark 0x4E00/0xFFFFFF00 2>/dev/null || true")
	return cmds
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
	// Trim brackets for IPv6
	endpoint = strings.TrimSpace(endpoint)
	if strings.HasPrefix(endpoint, "[") {
		idx := strings.LastIndex(endpoint, "]")
		if idx != -1 {
			host := endpoint[1:idx]
			portStr := endpoint[idx+1:]
			if strings.HasPrefix(portStr, ":") {
				portStr = portStr[1:]
			}
			port := 51820
			fmt.Sscanf(portStr, "%d", &port)
			return host, port
		}
	}

	host, portStr, err := net.SplitHostPort(endpoint)
	if err != nil {
		// If it's just an IP/hostname with no port
		return endpoint, 51820
	}
	port := 51820
	fmt.Sscanf(portStr, "%d", &port)
	return host, port
}
