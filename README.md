# Xynet — Visual Proxy & VPN Manager

## 1. Project Overview

Xynet is a visual, node-based VPN and proxy manager. It replaces list-based configurations with a drag-and-drop "Blender-style" canvas. Users drag lines from Applications (Inputs) to Network Tunnels (Outputs) to achieve process-level split-tunneling.

Wire Firefox to AirVPN on the canvas, click Deploy, and Firefox's traffic routes through the VPN. Direct, Block, and multiple simultaneous tunnels are all supported.

## 2. Tech Stack

**Backend Framework:** Wails v2 (Go). Bridges the OS and the UI.

**Frontend UI:** Vue.js 3 + TypeScript.

**Node Library:** Vue Flow (for the visual drag-and-drop canvas).

**Package Manager & Security:** Deno (replaces Node/NPM to prevent supply chain attacks).

**Routing Backends (pluggable):**
- **sing-box** (default) — Userspace TUN proxy with native WireGuard and Hysteria2 support. Cross-platform (Linux now, Windows/macOS planned). **Automatically installed as a dependency on Arch Linux**.
- **dae** (advanced, Linux-only) — eBPF TC hooks for kernel-level routing. Bypassed traffic never enters userspace.

**Strict Mode Isolation:** Vopono (Linux network namespace launcher for full traffic isolation).

**System Tray:** fyne.io/systray (pure DBus StatusNotifierItem).

**WireGuard Parsing:** `gopkg.in/ini.v1` (robust, battle-tested standard Go INI parser).

**Network Stats:** shirou/gopsutil (cross-platform interface bandwidth monitoring).

**IDE:** Zed (Configured with Deno and Vue.js LSPs).

## 3. Installation (Arch Linux / AUR)

Xynet is packaged and ready for the AUR. Because `sing-box` is explicitly declared as a dependency in the `PKGBUILD`, your AUR helper will automatically download and install it for you alongside Xynet.

```bash
# Using an AUR helper like yay
yay -S xynet

# Or using paru
paru -S xynet
```

When deploying the `sing-box` backend, Xynet launches `sing-box` directly as a managed subprocess using `pkexec` (PolicyKit). A custom polkit policy (`org.xynet.pkexec.sing-box.policy`) ensures users see a clean "Xynet needs administrator privileges..." prompt rather than a generic authentication window. There is no need to manually enable or manage a `sing-box` systemd service.

## 4. Routing Architecture

Xynet supports two routing modes per application:

### Standard Mode (default)

Traffic routing is handled by the selected backend (sing-box or dae). The backend intercepts traffic by process name and routes it through the configured tunnel.

```
User clicks Deploy
        ↓
Frontend: generateRoutingRules(canvasState)
  → [{processName, tunnelId, tunnelLabel, tunnelType}, ...]
        ↓
Go Backend: Deploy(rules)
  → reads settings.backend ("singbox" | "dae")
  → reads state.proxies (imported WireGuard .conf files)
  → parses WireGuard configs with wgparser.go
  → generates backend-specific config
  → writes config file + restarts backend service
```

**sing-box path:** Generates a JSON config with TUN inbound, WireGuard outbounds (parsed from imported .conf files), and process_name routing rules. Manages everything in userspace.

**dae path:** Runs `wg-quick up` for each WireGuard tunnel, sets up fwmark policy routing (`ip rule` + `ip route`), generates a dae config with `pname()` routing rules. Traffic is routed at the kernel level via eBPF.

### Strict Mode

For apps requiring complete network isolation (no IPC leaks), Strict mode launches the app inside a Linux network namespace using Vopono. The app is spawned inside the namespace — it cannot communicate with non-VPN'd services on the host.

Toggle an app to Strict mode on the canvas, then click the Launch button to start it inside the VPN namespace.

### Routing by Tunnel Type

| Tunnel Type | sing-box | dae |
|---|---|---|
| **WireGuard** | Userspace WireGuard outbound | Kernel `wg` interface via wg-quick |
| **Hysteria2** | Native Hysteria2 outbound | Not supported (use sing-box) |
| **Direct** | `direct` outbound | `direct` routing rule |
| **Block** | `block` outbound | `block` routing rule |

## 4. Key Features

**Automatic Process Routing:** Wire an app to a tunnel on the canvas and click Deploy. The backend routes the app's traffic through the configured tunnel by process name.

**WireGuard Support:** Import standard WireGuard `.conf` files. The Go backend parses PrivateKey, PublicKey, Endpoint, Address, DNS, and AllowedIPs to generate real backend configs.

**Hysteria2 Support:** Low-latency, DPI-bypassing connections handled natively by sing-box.

**Block / Kill Switch:** Apps wired to the "Block" tunnel have their traffic dropped.

**Direct / Bypass:** Apps wired to "Direct" use the system's default routing.

**Strict Mode:** Launch apps inside Vopono network namespaces for complete isolation — prevents IPC/D-Bus traffic leaks.

**Selectable Backend:** Choose between sing-box (recommended, cross-platform) and dae (high performance, Linux eBPF) in Settings.

## 5. Project Structure

```
app.go                        # Wails app struct, LaunchStrict, KillVopono
deploy.go                     # Pluggable backend deploy system (sing-box + dae)
wgparser.go                   # WireGuard .conf file parser
main.go                       # Wails entry point
state.go                      # Canvas state persistence + settings (backend choice)
desktop.go                    # Desktop app discovery (XDG .desktop files)
netmon.go                     # Network bandwidth monitoring (gopsutil)
status.go                     # Backend-aware service status polling
systray.go                    # System tray (fyne.io/systray)

frontend/
├── src/
│   ├── components/
│   │   ├── Canvas.vue          # Vue Flow canvas with drag-and-drop
│   │   ├── ApplicationNode.vue # App nodes (Standard/Strict mode toggle + Launch)
│   │   ├── TunnelNode.vue      # Tunnel output nodes
│   │   ├── WireEdge.vue        # Connection wires
│   │   ├── Header.vue          # Status bar with backend status + Deploy button
│   │   ├── Sidebar.vue         # Navigation
│   │   └── views/
│   │       └── Settings.vue    # Backend selector (sing-box / dae)
│   └── composables/
│       ├── routeGenerator.ts   # Canvas state → routing rules
│       └── useAppState.ts      # State management
└── wailsjs/                    # Auto-generated Wails bindings
```

## 6. The UI Layout

**Toolbar (Top):** Trash zone, Applications dropdown, Tunnels dropdown, Import Config button.

**Canvas (Center):** Drag-and-drop workspace. Application nodes on the left, Tunnel nodes on the right. Connect a wire from an app to a tunnel to create a routing rule.

**Header (Top):** Live upload/download speeds, backend status (sing-box/dae), Deploy button, Strict mode process count.

**Sidebar (Left):** Navigation between Canvas, Dashboard, Proxies, Logs, and Settings views.

## 7. Development Environment

**OS:** Arch Linux (CachyOS).

**Wails Compilation:** Must ALWAYS use the WebKit 4.1 tag:
```
wails dev -tags webkit2_41
```

If wails is not found:
```
export PATH="$PATH:$(go env GOPATH)/bin"
```

**Deno over NPM:** Node.js/NPM is banned. Use deno commands only:
```
deno add npm:@vue-flow/core
```

## 9. Instructions for the LLM

When acting as a coding assistant for this project:

- The routing backends are sing-box (default) and dae (Linux-only advanced option).
- Config generation happens in Go (`deploy.go`), not in the frontend.
- WireGuard configs are parsed from imported `.conf` files using `wgparser.go` which leverages the battle-tested `gopkg.in/ini.v1` package.
- `sing-box` is managed as a direct subprocess (`pkexec sing-box run`), NOT via systemd. This makes it cross-platform compatible.
- Imported configs in the `Proxies` tab are saved to the persistent state via `useAppState()`.
- Strict mode uses Vopono network namespaces — apps must be launched inside the namespace, not attached after the fact.
- The project targets Linux now but is designed for future Windows/macOS support. sing-box is the cross-platform path; dae is Linux-only.
- Do NOT suggest adding Electron, Node.js, or heavy web dependencies.
- Do NOT generate npm, yarn, or pnpm commands. Use deno.
- Assume a working Wails/Vue/Deno setup.
