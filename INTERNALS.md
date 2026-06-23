# Xynet Codebase Internals & Post-Mortems

This document provides a granular, function-by-function breakdown of the Xynet codebase. Crucially, it documents *why* the architecture is structured this way, highlighting the specific glitches, edge cases, and vulnerabilities that dictated our design decisions.

---

## 1. Frontend Internals (Vue 3 + TypeScript)

The frontend is built on Vue Flow, acting as a visual state machine that ultimately compiles into a list of routing rules.

### `src/components/Canvas.vue`
*   **Purpose**: Manages the drag-and-drop workspace, rendering Application nodes, Tunnel nodes, and Wires.
*   **Key Logic**: Uses Vue Flow's `@nodeDragStop` and `@connect` events to update node positions and wiring.
*   **Glitch/Fix (The Trash & Restore Bug)**: When dragging a node to the "Trash" zone and then bringing it back, the app was losing the `processName` and `mode` (Strict/Standard) properties. 
    *   *Why*: The drag-preview component was deep-cloning a subset of properties but omitting extended data. 
    *   *Fix*: We explicitly spread the previous state `data: { ...dragPreview.data, mode: "Standard" }` when re-instantiating nodes dropped from the sidebar or trash.

### `src/components/ApplicationNode.vue` & `TunnelNode.vue`
*   **Design Decision (The "Launch" Button)**: Why do Strict mode apps need a "Launch" button, while Standard mode apps don't?
    *   *Kernel Limitation*: Linux `setns()` (Network Namespaces) only allows a calling thread to change its *own* namespace. We cannot forcefully shove an already-running process into a new namespace securely. Therefore, Vopono (Strict mode) *must* be the parent process that spawns the application. Standard mode (Sing-box/Dae) uses eBPF/cgroups, which *can* intercept packets from existing processes dynamically.
*   **Active Canvas Glowing**: When the user clicks Deploy, `useActiveDeployment.ts` caches the exact rule array. Any node matching an active rule gets the `is-active-route` CSS class, providing a solid green halo so users instantly know what the kernel is currently intercepting.

### `src/components/views/Proxies.vue`
*   **Purpose**: Handles the importation, bulk loading, and direct code-editing of WireGuard/Hysteria2 files.
*   **Glitch/Fix (The "Amnesia" Bug)**: Initially, imported proxies vanished if the user navigated to another tab.
    *   *Fix*: Migrated the component to read/write directly against `useAppState().appState.value.proxies`, ensuring global persistence to the Go backend's `state.json`.
*   **Glitch/Fix (The Render Crash)**: When adding the Code Editor `<textarea>`, Vue threw massive `undefined is not an object` errors.
    *   *Why*: The expanding editor `div` was placed just outside of the `v-for="server in serverList"` loop, destroying the variable context.
    *   *Fix*: Wrapped the list row and editor drawer together inside a `<template v-for>` block.

---

## 2. Backend Internals (Go + Wails)

The backend acts as the system mediator, talking to Vue on one side and the Linux Kernel/Binaries on the other.

### `app.go` (The IPC Controller)
*   **`LaunchStrict`**: Spawns applications in namespaces.
    *   **Glitch/Fix (The Doppelganger Bug)**: Users could click "Launch" multiple times rapidly, spawning orphaned headless namespaces.
    *   *Fix*: Implemented a Mutex lock that loops over `a.processInfo`. If an app name + tunnel label combo is already registered as running, the launch is outright rejected.
*   **`shutdown(ctx)`**: The application teardown hook.
    *   **Glitch/Fix (The Zombie Process Bug)**: Closing the Xynet window left Vopono and Sing-box processes running invisibly in the background.
    *   *Fix*: The `shutdown` hook iterates through tracked `*exec.Cmd` pointers, sends `syscall.SIGTERM`, waits 500ms, and upgrades to `syscall.SIGKILL` if they refuse to die.

### `deploy.go` (The Routing Engine)
*   **`deploySingbox()`**:
    *   **Glitch/Fix (The "Exit Status 5" Bug)**: Originally tried to restart `sing-box` via systemd (`systemctl restart sing-box`). Failed completely because systemd daemons expect configs in `/etc/`, not local user folders, and require root for daemon management.
    *   *Fix*: Spawned `sing-box` as a direct subprocess: `pkexec sing-box run -c ~/.config/xynet/...`. Guarantees Xynet owns the lifecycle.
    *   **Glitch/Fix (The Watchdog Loop)**: If the backend crashed, Xynet's UI got permanently stuck as "Active".
    *   *Fix*: Added `intentionalStop bool`. If the wait loop finishes and `intentionalStop` is false, the goroutine sleeps 3 seconds and literally recalls `deploySingbox` to auto-heal the VPN.
*   **`deployDae()`**:
    *   **Glitch/Fix (The Bash Script Nightmare)**: Previously, Xynet generated massive `#!/bin/bash` scripts, wrote them to disk, and executed them. This was horrible for security and cluttered the file system.
    *   *Fix*: Bundled all `ip rule`, `wg-quick`, and `cp` commands into a Go string array and executed them directly in memory via `exec.Command("pkexec", "sh", "-c", "command1 && command2")`.

### `wgparser.go` (WireGuard & Desktop Parsers)
*   **`ParseWireGuardConfig()`**:
    *   **Glitch/Fix (Mesh Routing Blackhole)**: The parser used to simply call `cfg.GetSection("Peer")`, which strictly only returns the *first* block. Multi-peer configs silently lost routes.
    *   *Fix*: Switched to `cfg.SectionsByName("Peer")` and built an array iteration loop.
*   **`parseDesktopFile()` (in `desktop.go`)**:
    *   **Glitch/Fix (Fragile `.desktop` reading)**: Custom `bufio.Scanner` string-splitting constantly broke on weird formatting or localized strings (`Name[es]=`).
    *   *Fix*: Scrapped the custom parser and wired it into `gopkg.in/ini.v1` using `IgnoreInlineComment: true`.

### `systray.go`
*   **`setupSystray()`**:
    *   **Glitch/Fix (Wayland Invisibility)**: Used `getlantern/systray` (based on `libayatana-appindicator`), which failed to render on Wayland KDE/GNOME.
    *   *Fix*: Migrated to `fyne.io/systray` running inside a dedicated `go systray.Run(onReady, onExit)` goroutine. Solved all DBus/StatusNotifierItem rendering glitches.

### `netmon.go`
*   **`getAggregateNetCounters()`**:
    *   **Glitch/Fix (The VPN Double-Count)**: When a VPN spun up `tun0` or `wg0`, the generic net monitor started counting the exact same bytes twice (once when the app sent them, and again when the encrypted tunnel sent them out `eth0`). 
    *   *Fix*: Stripped out virtual interfaces (`tun`, `wg`, `veth`, `lo`) and strictly bound the aggregator to the user's `DefaultInterface` defined in the Vue Settings UI.

---

## 3. Major Architectural Decisions

### Retiring Systemd for `dae`
Initially, `dae` was heavily reliant on `systemctl reload-or-restart dae`. 
**The Problem**: AUR variants (`dae-bin`, `dae-git`, `dae-avx2-bin`) sometimes broke systemd paths, or users didn't want the VPN hijacking their routing on system boot.
**The Solution**: We wrote `installer.go` to directly hit the GitHub API, download the raw `dae-linux-x86_64.zip`, and extract it into `~/.config/xynet/bin/`. We now run `dae` purely as an isolated subprocess, identical to `sing-box`. It guarantees Xynet is the only orchestrator and bypassing the AUR completely.

### The IPC Leak Vulnerability & "Strict Mode"
**The Vulnerability**: When routing an app like `firefox` through eBPF/cgroups ("Standard Mode"), standard TCP/UDP traffic is safely caught. However, Firefox can communicate with a background daemon (like `systemd-resolved` or a local proxy) via **Unix Domain Sockets (IPC / D-Bus)**. Because the background daemon is *not* in the VPN cgroup, it executes DNS queries on the host's cleartext network. The user's IP is leaked.

**The Solution**: **Strict Mode** (Vopono). Vopono leverages Linux Network Namespaces (`netns`). By trapping the application inside a namespace, it is given a virtual ethernet cable tied *exclusively* to the WireGuard interface. It physically cannot reach the host's D-Bus or IPC sockets. This guarantees 100% cryptographic isolation.
