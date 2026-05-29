📋 PROJECT CONTEXT: NodeNet (Visual Proxy & VPN Manager)

1. Project Overview

NodeNet is a visual, node-based VPN and proxy manager built for Linux (specifically CachyOS/Arch). It replaces list-based configurations with a drag-and-drop "Blender-style" canvas. Users drag lines from Applications (Inputs) to Network Tunnels (Outputs) to effortlessly achieve advanced process-level split-tunneling.

2. Tech Stack

Backend Framework: Wails v2 (Go). Used to bridge the OS and the UI, execute system commands, and spawn background daemon processes.

Frontend UI: Vue.js 3 + TypeScript.

Node Library: Vue Flow (for the visual drag-and-drop canvas).

Package Manager & Security: Deno (replaces Node/NPM to prevent supply chain attacks).

Primary Engine (Userspace): Sing-box (Universal proxy daemon for 95% of everyday apps).

Secondary Engine (Kernel): Vopono / Network Namespaces (For strict, isolated "Paranoid Mode" routing).

IDE: Zed (Configured with Deno and Vue.js LSPs).

3. The "Dual-Engine" Architecture

NodeNet does not route traffic directly. It acts as a visual JSON generator and process manager for two parallel routing engines:

The Sing-box Engine (Flexible Mode): Uses a TUN interface, FakeIP DNS, and process-sniffing to seamlessly route everyday apps (Browsers, Games).

The Vopono Engine (Paranoid Mode): Uses Linux kernel netns (Network Namespaces) to create mathematically isolated "bubbles" with pure kernel-level WireGuard. Used for apps where IPC leaks would be disastrous (e.g., Torrenting).

Note on overlap: Sing-box is configured to bypass (ignore) encrypted WireGuard UDP packets coming out of the Vopono namespaces, preventing double-encryption.

4. Key Network Features & Capabilities Supported

Process-Based Split Tunneling (Sing-box): Routing specific apps to specific outbounds effortlessly.

Absolute Namespace Isolation (Vopono): Using $PATH wrappers or .desktop file overrides so apps launch natively into kernel-isolated VPN bubbles with flawless Kill Switches.

Hysteria2 Support: Used for stealth, low-latency, DPI-bypassing connections to a Home Server (critical for Sunshine game streaming over hostile school/work Wi-Fi).

WireGuard Support: Native support for standard VPNs (like AirVPN) in both Sing-box and Vopono.

Direct / Bypass: Native support for ignoring local network traffic or apps that need naked internet (e.g., Moonlight at an unblocked coffee shop).

Port Forwarding (Inbound): Support for routing traffic coming IN from an AirVPN forwarded port directly to a local process (like localhost:47984 for Sunshine).

5. Development Environment Quirks (CRITICAL FOR LLM)

OS: Arch Linux (CachyOS).

Wails Compilation: Must ALWAYS be run with the WebKit 4.1 tag due to Arch Linux deprecating 4.0.

Command: wails dev -tags webkit2_41

Deno over NPM: Node.js/NPM is banned in this project.

wails.json has been modified so frontend:Install uses deno install.

deno.json exists in the frontend folder.

Do NOT generate npm, yarn, or pnpm commands. Generate deno commands (e.g., deno add npm:@vue-flow/core).

6. The UI Layout Concept

Left Column (Inputs): Draggable nodes representing Applications (e.g., Firefox, qBittorrent, Sunshine) and a "Default System" node.

Feature: Input nodes can be toggled as [Standard] (handled by Sing-box) or [Strict/Namespace] (handled by Vopono).

Right Column (Outputs): Draggable nodes representing Outbound Tunnels (AirVPN, Hysteria2 Home Server, Bypass/Direct, Block/Killswitch).

Interaction: Connecting a line between an Input and an Output writes a rule in the Sing-box JSON array OR triggers Go to execute a vopono exec wrapper script.

7. Instructions for the LLM

When acting as my coding assistant for this project:

Assume I have a working Wails/Vue/Deno setup running.

Provide modular, clean Go code for the Wails backend (app.go) for writing JSON files, restarting Sing-box, and executing vopono CLI commands via os/exec.

Provide Vue 3 <script setup lang="ts"> code using Vue Flow to render the UI.

If asked to generate Sing-box configurations, ensure they utilize the tun interface, fakeip for DNS, and process_name rules.

Prioritize performance and security. Do not suggest adding Electron, Node.js, or heavy web dependencies unless strictly necessary.

