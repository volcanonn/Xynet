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
    *   *Fix*: We explicitly spread the previous state `data: { ...dragPreview.data, mode: "Standard" }` when re-instantiating nodes dropped from the sidebar or trash, ensuring the process payload is never lost.

### `src/components/ApplicationNode.vue`
*   **Purpose**: The custom UI for Application (Input) nodes. Displays the app icon, name, and mode controls.
*   **Key Logic**:
    *   `toggleMode()`: Flips between Standard and Strict mode. Requires user confirmation if flipping to Strict, as it changes how the app must be launched.
    *   `launchApp()`: Only visible when `mode === 'Strict'` AND a wire is connected. Triggers `LaunchStrict()` in the Go backend.
*   **Design Decision**: Why do Strict mode apps need a "Launch" button, while Standard mode apps don't?
    *   *Kernel Limitation*: Linux `setns()` (Network Namespaces) only allows a calling thread to change its *own* namespace. We cannot forcefully shove an already-running process into a new namespace securely. Therefore, Vopono (Strict mode) *must* be the parent process that spawns the application. Standard mode (Sing-box/Dae) uses eBPF/cgroups, which *can* intercept packets from existing/arbitrary processes dynamically.

### `src/components/views/Proxies.vue`
*   **Purpose**: Handles the importation and management of WireGuard `.conf` files.
*   **Glitch/Fix (The "Amnesia" Bug)**: Initially, imported proxies vanished if the user navigated to another tab.
    *   *Why*: The imported configurations were stored in a component-local Vue `ref` (`const configs = ref([])`). Vue destroys local state when unmounting the view.
    *   *Fix*: Migrated the component to read/write directly against `useAppState().appState.value.proxies`, ensuring global persistence to the Go backend's `state.json`.

### `src/composables/useAppState.ts`
*   **Purpose**: The central bridge between Vue reactivity and Wails IPC persistent storage.
*   **Glitch/Fix (Reactivity Crash)**: Sending raw Vue Proxy objects to Go via Wails `SaveState()` caused serialization crashes or empty files.
    *   *Fix*: Implemented deep serialization stripping: `const raw = JSON.parse(JSON.stringify(toRaw(appState.value)))`. This removes all Vue reactivity wrappers before passing the struct over the IPC bridge.

---

## 2. Backend Internals (Go + Wails)

The backend acts as the system mediator, talking to Vue on one side and the Linux Kernel/Binaries on the other.

### `app.go` (The IPC Controller)
*   **`LaunchStrict(processName, tunnelLabel)`**: Finds the requested proxy config, writes it to a secure `~/.config/xynet/strict/` folder, and executes `vopono exec`.
*   **`shutdown(ctx)`**: The application teardown hook.
    *   **Glitch/Fix (The Zombie Process Bug)**: Closing the Xynet window left Vopono and Sing-box processes running invisibly in the background, locking up network ports.
    *   *Fix*: We added a `sync.Mutex` and maps (`voponoProcesses`, `singboxCmd`) to track raw `*exec.Cmd` pointers. The `shutdown` hook iterates through these, sends `syscall.SIGTERM`, waits 500ms, and if they refuse to close, sends `syscall.SIGKILL`.
*   **`KillVopono(id)`**: Kills specific Strict mode instances dynamically if the user requests it.

### `deploy.go` (The Routing Engine)
*   **`Deploy(rules)`**: Routes configuration generation to either `deploySingbox` or `deployDae`.
*   **`deploySingbox()`**: Builds the massive JSON structure required for Sing-box TUN routing.
    *   **Glitch/Fix (The "Exit Status 5" Bug)**: Xynet originally tried to apply configurations by calling `systemctl restart sing-box`. This failed critically because (A) the user might not have sing-box installed, and (B) we were writing our config to `~/.config`, but the systemd daemon reads from `/etc/sing-box`.
    *   *Fix*: We abandoned systemd entirely for sing-box. We now spawn it as an isolated child process: `pkexec sing-box run -c ~/.config/xynet/sing-box-config.json`. This guarantees Xynet owns the lifecycle and works identically across all Linux distros (and sets us up for macOS/Windows portability).
*   **`StopSingbox()`**: Tears down the active proxy.
    *   **Glitch/Fix (The Deadlock Bug)**: The fallback timeout was written as `case <-make(chan struct{})`. An empty, unbuffered channel blocks forever, meaning if Sing-box hung, the Go backend deadlocked trying to kill it.
    *   *Fix*: Replaced with `case <-time.After(3 * time.Second)`, ensuring a forceful `SIGKILL` timeout.

### `wgparser.go` (WireGuard Parser)
*   **`ParseWireGuardConfig()`**: Converts an INI string into a typed `WireGuardConfig` struct.
    *   **Glitch/Fix (Regex Nightmares)**: We originally wrote a custom string-splitting parser. It broke immediately on complex `.conf` files containing inline comments, out-of-order `[Peer]` blocks, or weird spacing.
    *   *Fix*: Stripped out the custom code and integrated `gopkg.in/ini.v1`. A battle-tested library ensures we don't accidentally malform cryptography keys or drop routing IPs.

### `state.go`
*   **`SaveState()` / `LoadState()`**: Manages the `~/.config/xynet/state.json` file.
    *   **Glitch/Fix (State Corruption)**: Rapidly clicking the UI while the app was saving would corrupt the JSON file, wiping the user's entire setup.
    *   *Fix*: Introduced a `sync.Mutex` (`a.stateMu`) to lock the file during IO. Furthermore, implemented atomic writes: data is written to `state.json.tmp` and then `os.Rename` is called to atomically swap the file, ensuring a power-loss mid-save can never corrupt the original configuration.

### `systray.go`
*   **`setupSystray()`**: Creates the Linux taskbar icon.
    *   **Glitch/Fix (Wayland/DBus Incompatibility)**: The original implementation used `getlantern/systray` (based on `libayatana-appindicator`). This frequently failed to render on modern KDE/Wayland setups.
    *   *Fix*: Migrated to `fyne.io/systray`, which implements a pure DBus `StatusNotifierItem`. This resolved all tray rendering glitches across GNOME, KDE, and Sway.
    *   **Glitch/Fix (The Exit Loop)**: The tray "Quit" button called `os.Exit(0)`. This violently bypassed the Wails shutdown hooks, resulting in the Zombie Process bug mentioned above.
    *   *Fix*: Changed the Quit button to call `runtime.Quit(a.ctx)`, which gracefully triggers Wails to fire the `shutdown()` function and clean up kernel processes.

---

## 3. Major Architectural Decisions

### The IPC Leak Vulnerability & The Birth of "Strict Mode"
Initially, Xynet relied *solely* on our custom eBPF/cgroup kernel engine to route apps (what is now "Standard Mode"). 

**The Vulnerability**: 
We discovered that if you route `firefox` through the VPN via eBPF/cgroups, standard TCP/UDP traffic is safely caught. However, Firefox can communicate with a background daemon (like `systemd-resolved` or a local proxy) via **Unix Domain Sockets (IPC / D-Bus)**. Because the background daemon is *not* in the VPN cgroup, it executes DNS queries or fetches data on the host's cleartext network. The user's IP is leaked.
No I knew about this. Maybe if this fuckass antigravity claude proxy setup with 5 chinese gemini pro accs was actualy good and let met use gemini 3.1 high fuck.

**The Solution**: 
We introduced **Strict Mode** (Vopono). Vopono leverages Linux Network Namespaces (`netns`). By trapping the application inside a namespace, it is given a virtual ethernet cable tied *exclusively* to the WireGuard interface. It physically cannot reach the host's D-Bus or IPC sockets unless explicitly bridged. This guarantees 100% cryptographic isolation at the cost of requiring the app to be launched "from inside" the namespace.

### Retiring the Custom eBPF Engine for Dae & Sing-box
Xynet originally contained its own `engine/` directory with custom C-based eBPF tracepoints (`sched_process_exec`) and Netlink routing managers.

**The Problem**: 
Maintaining a bespoke kernel firewall is exceptionally dangerous. Edge cases in Linux kernel versions, conflicting iptables rules from Docker, and the complexity of parsing Hysteria2 in Go proved too heavy a burden.

**The Solution**: 
We gutted the custom engine and pivoted Xynet to be a **Visual Orchestrator** for battle-tested binaries.
*   **Sing-box** was chosen as the default because it manages its own userspace TUN interface, making it portable to Windows and macOS in the future.
*   **Dae** was included as an option because it does exactly what our custom engine did (eBPF + cgroups + fwmarks), but is maintained by a massive open-source community, guaranteeing high performance and reliability on Linux.
