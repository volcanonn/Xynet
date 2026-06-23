# Xynet Architecture & Internals

This document provides a deep, step-by-step architectural breakdown of how Xynet operates. It covers the frontend lifecycle, state persistence, configuration parsing, and the precise execution flows for both Standard Mode (Sing-box / Dae) and Strict Mode (Vopono).

---

## 1. High-Level System Overview

Xynet is a desktop application built on **Wails v2**. It utilizes a **Vue 3 + TypeScript** frontend (bundled via Deno) and a **Go** backend. 

The application is designed to map visual nodes (representing desktop applications) to tunnel nodes (representing WireGuard interfaces, Bypass, or Block destinations) via edges (wires). This visual graph is compiled into a strict routing table enforced by either a userspace TUN proxy (`sing-box`) or a kernel-level eBPF router (`dae`).

### Privilege Model
*   **Frontend**: Runs entirely unprivileged.
*   **Go Backend (Main App)**: Runs unprivileged as the current user.
*   **Routing Execution**: 
    *   `sing-box`: Launched as a child process via `pkexec` to obtain `CAP_NET_ADMIN` (for TUN interface creation).
    *   `dae`: Managed via `systemctl`, requiring root for systemd and eBPF injection.
    *   `vopono`: Requires sudo/root privileges internally to configure network namespaces.

---

## 2. State Management & Persistence

The entire state of the application is centralized in a single persistent file: `~/.config/xynet/state.json`.

### Frontend to Backend Sync
1.  **Frontend State**: `useAppState.ts` maintains a Vue reactive `appState` object encompassing:
    *   `canvasElements`: The nodes and edges on the Vue Flow canvas.
    *   `proxies`: An array of `ImportedProxy` objects (name and raw `.conf` content).
    *   `settings`: Global configurations (e.g., selected backend: `singbox` or `dae`).
2.  **Saving (`saveState`)**: When an update occurs (a node is moved, a proxy is imported, settings are changed), the frontend strips Vue reactivity (`toRaw`) and passes the JSON object to the Wails IPC function `SaveState()`.
3.  **Atomic Writes (Go)**: To prevent corruption during power loss or crash, `SaveState()` writes to a temporary file (`state.json.tmp`) and performs an atomic filesystem rename (`os.Rename`) to overwrite the active `state.json`. `stateMu` (a sync.Mutex) prevents concurrent disk writes.

---

## 3. The "Deploy" Lifecycle (Step-by-Step)

When a user clicks the **Deploy** button in the Header, the following sequence occurs:

### Step 3.1: Rule Generation (Frontend)
1.  `generateRoutingRules(appState)` is invoked in `routeGenerator.ts`.
2.  It iterates over all edges (wires) on the canvas.
3.  It resolves the Source (Application Node) and Target (Tunnel Node).
4.  **Strict Mode Filter**: If an Application Node is explicitly set to `mode: "Strict"`, it is **ignored** during deployment (Strict mode apps are launched independently on-demand).
5.  It outputs an array of `RoutingRule` objects:
    ```typescript
    {
      processName: "firefox",
      tunnelId: "tun-airvpn",
      tunnelLabel: "AirVPN-NL",
      tunnelType: "WireGuard"
    }
    ```

### Step 3.2: Dispatch to Go Backend
The frontend invokes the Wails IPC method `Deploy(rules)`.

1.  The Go backend calls `LoadState()` to retrieve the currently selected backend (`singbox` or `dae`) and the raw imported proxy configurations (`state.Proxies`).
2.  It routes the request to either `deploySingbox(rules)` or `deployDae(rules)`.

---

## 4. Routing Backend: Sing-box (Standard Mode)

If `sing-box` is the active backend, the following architecture is executed:

### Step 4.1: WireGuard Parsing
1.  For each unique `tunnelLabel` found in the rules, Go retrieves the raw string content of the proxy.
2.  `ParseWireGuardConfig()` (in `wgparser.go`) utilizes `gopkg.in/ini.v1` to safely parse the INI structure.
3.  It extracts:
    *   `[Interface]`: PrivateKey, Address, DNS, MTU.
    *   `[Peer]`: PublicKey, PresharedKey, Endpoint, AllowedIPs.

### Step 4.2: JSON Configuration Building
1.  Go constructs a native `sing-box` configuration map.
2.  **Inbounds**: Configures a `tun` inbound (`tun0`) with `auto_route: true` and `strict_route: true`.
3.  **Outbounds**:
    *   Adds `direct` and `block` outbounds.
    *   For each parsed WireGuard proxy, creates a `wireguard` outbound utilizing the exact keys/endpoints extracted in Step 4.1.
4.  **Routing Rules**: Maps each `rule.ProcessName` to its corresponding outbound tag (`outbound: "tun-airvpn"` or `outbound: "block"`).

### Step 4.3: Execution & Subprocess Management
1.  The generated JSON is written to `~/.config/xynet/sing-box-config.json`.
2.  `StopSingbox()` is called. It acquires `singboxMu`, sends `SIGTERM` to any currently running sing-box subprocess, waits up to 3 seconds, and upgrades to `SIGKILL` if it hasn't exited.
3.  A new command is spawned: `pkexec /usr/bin/sing-box run -c ~/.config/xynet/sing-box-config.json`.
    *   *Note:* The `org.xynet.pkexec.sing-box.policy` PolKit file ensures the user gets a polished GUI authentication prompt and caches the authentication.
4.  A background goroutine calls `cmd.Wait()` to monitor the subprocess lifecycle and update the `singboxCmd` pointer to `nil` upon exit.

---

## 5. Routing Backend: Dae (Advanced Linux eBPF Mode)

If `dae` is the active backend, Xynet leverages kernel-level eBPF hooks (`TC` - Traffic Control) rather than a userspace TUN proxy.

### Step 5.1: Interface Lifecycle Management
Unlike sing-box, `dae` does not manage WireGuard natively. It relies on system interfaces.
1.  Go iterates through the required WireGuard tunnels.
2.  Writes the raw `.conf` file to `~/.config/xynet/wg-configs/<tunnelLabel>.conf`.
3.  Executes `wg-quick down <path>` (to clean up old state) followed by `wg-quick up <path>`.

### Step 5.2: Policy Routing Allocation
To prevent traffic loops and allow dae to route specific packets to specific WireGuard interfaces, Xynet manually manipulates the Linux networking stack:
1.  Assigns a unique `fwmark` (firewall mark, e.g., `0x4E01`) and a custom routing table ID (e.g., `100`) for each active tunnel.
2.  Executes: `ip rule add fwmark 0x4E01 table 100`
3.  Executes: `ip route add default dev <TunnelLabel> table 100`

### Step 5.3: Dae Config Generation & Reload
1.  Generates `/etc/dae/config.dae`.
2.  For standard apps, outputs: `pname(firefox) -> direct(mark: 0x4E01)`
    *   *Translation:* "If the process name matches 'firefox', bypass the default route and tag the packet with `0x4E01`." The Linux kernel then sees this fwmark and forces it into routing table `100` (created in Step 5.2), pushing it out the WireGuard interface.
3.  For blocked apps, outputs: `pname(qbittorrent) -> block`
4.  Executes `systemctl reload-or-restart dae`.

---

## 6. Strict Mode (Vopono & Network Namespaces)

Strict Mode completely isolates applications from the host network namespace to prevent IPC (Inter-Process Communication) leaks (e.g., an app sending D-Bus messages to a background service that bypasses the VPN).

### Step 6.1: Launch Initiation
1.  On the Canvas, an Application Node set to "Strict" displays a "Launch" button.
2.  Clicking it scans the Vue Flow edges to find which Tunnel it is connected to.
3.  Invokes Wails IPC `LaunchStrict(processName, tunnelLabel)`.

### Step 6.2: Vopono Execution
1.  Go extracts the raw proxy config from `state.Proxies` based on `tunnelLabel`.
2.  Writes it securely to `~/.config/xynet/strict/<tunnelLabel>.conf`.
3.  Spawns: `vopono exec --custom ~/.config/xynet/strict/<tunnelLabel>.conf <processName>`
    *   *Vopono Internals:* This creates a fresh Linux network namespace, brings up a WireGuard interface *inside* the namespace, and spawns the application exclusively within it.

### Step 6.3: Process Tracking
1.  Go generates a unique UUID for the subprocess.
2.  Acquires `voponoMu` (Mutex) and stores the `*exec.Cmd` and PID in `a.voponoProcesses` and `a.processInfo`.
3.  Fires a Wails event `vopono-process-started` to the frontend.
4.  A background goroutine `Wait()`s on the process. When the user closes the application, Vopono exits, the goroutine removes it from the maps, and fires `vopono-process-ended`, instantly updating the "Strict: N" counter in the UI Header.

---

## 7. Packaging and Distribution Architecture

Xynet is packaged explicitly for Arch Linux via the AUR.

1.  **PKGBUILD**: 
    *   Pulls dependencies: `sing-box`, `webkit2gtk-4.1`, `gtk3`.
    *   Build dependencies: `go`, `wails`, `deno`.
    *   Sets CGO flags strictly matching Arch Linux standards (`-buildmode=pie -trimpath`).
2.  **XDG Integration**: 
    *   `xynet.desktop` is installed to `/usr/share/applications/`.
    *   The compiled logo `appicon.png` is placed in `/usr/share/icons/hicolor/256x256/apps/xynet.png`.
3.  **Polkit Policy**:
    *   `org.xynet.pkexec.sing-box.policy` is moved to `/usr/share/polkit-1/actions/` during package installation, securing the seamless PKEXEC privilege escalation natively required by the `sing-box` TUN configuration step.
