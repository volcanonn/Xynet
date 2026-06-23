# Xynet Architecture & Internals

This document provides a deep, step-by-step architectural breakdown of how Xynet operates. It covers the frontend lifecycle, state persistence, configuration parsing, and the precise execution flows for both Standard Mode (Sing-box / Dae) and Strict Mode (Vopono).

---

## 1. High-Level System Overview

Xynet is a desktop application built on **Wails v2**. It utilizes a **Vue 3 + TypeScript** frontend (bundled via Deno) and a **Go** backend. 

The application is designed to map visual nodes (representing desktop applications) to tunnel nodes (representing WireGuard interfaces, Bypass, or Block destinations) via edges (wires). This visual graph is compiled into a strict routing table enforced by either a userspace TUN proxy (`sing-box`) or a kernel-level eBPF router (`dae`).

### Privilege Model
*   **Frontend**: Runs entirely unprivileged.
*   **Go Backend (Main App)**: Runs unprivileged as the current user.
*   **Routing Execution**: All actual routing modifications are executed via `pkexec`. Xynet manages all background daemons as *direct subprocesses*, entirely bypassing `systemd`. This ensures Xynet perfectly tracks lifecycle states and exits cleanly.

---

## 2. State Management & Event Bus

The entire state of the application is centralized in a single persistent file: `~/.config/xynet/state.json`.

### Frontend to Backend Sync
1.  **Frontend State**: `useAppState.ts` maintains a Vue reactive `appState` object.
2.  **Saving (`saveState`)**: When an update occurs, the frontend strips Vue reactivity (`toRaw`) and passes the JSON object to the Wails IPC function `SaveState()`.
3.  **Atomic Writes (Go)**: To prevent corruption during power loss or crash, `SaveState()` writes to a temporary file (`state.json.tmp`) and performs an atomic filesystem rename (`os.Rename`).

### Global Telemetry & Event Bus
Instead of relying on heavy frontend-driven polling, Xynet uses the Wails event bus:
1.  `netmon.go`: Aggregates bandwidth for the user's explicit `DefaultInterface` (ignoring virtual TUN/WG interfaces) and emits `net-stats` once per second.
2.  `status.go`: Checks the internal `*exec.Cmd` pointers of the routing engines and emits `service-status`.
3.  **Frontend Receivers**: `useTelemetry.ts` subscribes to these events and globally updates the UI for the Dashboard, Header, and active canvas indicators.

---

## 3. The "Deploy" Lifecycle (Step-by-Step)

When a user clicks the **Deploy** button:

1.  `generateRoutingRules(appState)` is invoked in `routeGenerator.ts`.
2.  **Strict Mode Filter**: If an Application Node is explicitly set to `mode: "Strict"`, it is ignored during deployment (Strict mode apps are launched independently).
3.  The frontend invokes the Wails IPC method `Deploy(rules)`.
4.  The Go backend calls `LoadState()` to retrieve the backend choice and raw proxy configurations, routing the request to `deploySingbox` or `deployDae`.

---

## 4. Routing Backend: Sing-box (Standard Mode)

### Step 4.1: Config Parsing & Latency Testing
1.  Go retrieves the raw string content of the proxies.
2.  `ParseWireGuardConfig()` safely parses the INI structure, capturing all `[Peer]` blocks for mesh routing.
3.  If it's a Hysteria2 `.txt` URI, it natively extracts the authentication, SNI, and ports.
4.  The UI automatically calls `MeasureLatency()` via IPC, which performs a native ICMP ping against the extracted endpoint to color-code the node on the canvas.

### Step 4.2: Execution & Subprocess Management
1.  The generated JSON is written to `~/.config/xynet/sing-box-config.json`.
2.  `StopSingbox()` is called. It acquires `singboxMu`, sets an `intentionalStop` flag, and sends `SIGTERM` (falling back to `SIGKILL`) to any running process.
3.  A new command is spawned: `pkexec sing-box run -c ~/.config/xynet/sing-box-config.json`.
4.  A background goroutine `Wait()`s on the process. If `sing-box` crashes unexpectedly (without the `intentionalStop` flag), a built-in watchdog waits 3 seconds and automatically restarts the deploy cycle.
5.  `StdoutPipe` and `StderrPipe` are ingested by a `bufio.Scanner` and piped to the frontend's Wails `app-log` event bus, populating the Logs tab in real-time.

---

## 5. Routing Backend: Dae (Advanced Linux eBPF Mode)

Xynet manages `dae` independently of `systemd`, allowing users to install the binary natively from GitHub via Xynet's built-in `installer.go`.

### Step 5.1: Interface Lifecycle Management (Privileged Chains)
Instead of writing messy `.sh` files to disk, Xynet uses Go's string arrays to build a privileged memory chain:
1.  Writes the raw `.conf` file to `~/.config/xynet/wg-configs/`.
2.  Assigns a unique `fwmark` (e.g., `0x4E01`) and a custom routing table ID.
3.  Constructs a command: `pkexec sh -c "wg-quick down ... && wg-quick up ... && ip rule add fwmark ..."`
4.  Executes the chain securely in one pass.

### Step 5.2: Dae Execution
1.  Injects the routing maps (`pname(firefox) -> direct(mark: 0x4E01)`) into the `//go:embed templates/dae.tmpl` file.
2.  Spawns `pkexec ~/.config/xynet/bin/dae run -c config.dae` as a tracked subprocess, feeding its stdout to the Logs view identically to `sing-box`.

---

## 6. Strict Mode (Vopono & Network Namespaces)

Strict Mode isolates applications from the host network namespace to prevent IPC (Inter-Process Communication) and D-Bus leaks.

1.  On the Canvas, an Application Node set to "Strict" displays a "Launch" button.
2.  Wails IPC `LaunchStrict` writes the proxy config to a secure temp folder.
3.  A Mutex lock checks `a.processInfo` to guarantee duplicate launches are blocked.
4.  Spawns: `pkexec vopono exec --custom <conf> <processName>`.
5.  Fires a `vopono-process-started` event to increment the UI counters, and pipes all isolated process logs to the Logs tab.
