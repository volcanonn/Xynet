import { main } from "../../wailsjs/go/models";

// Sing-box config generation according to the dual-engine architecture rules.
// TUN interface, FakeIP, and process_name routing rules.
export function generateSingboxConfig(state: main.AppState): string {
    const nodes = state.canvasElements.filter((e: any) => e.position);
    const edges = state.canvasElements.filter((e: any) => e.source && e.target);

    // Filter standard apps connected to tunnels
    const standardApps = nodes.filter((n: any) => n.type === 'application' && n.data?.mode === 'Standard');

    // We only need to route apps that have connections
    const outbounds: any[] = [];
    const routeRules: any[] = [];

    // Base outbounds (Direct/Block)
    outbounds.push({
        type: "direct",
        tag: "direct"
    });
    outbounds.push({
        type: "block",
        tag: "block"
    });

    // Map each tunnel node to an outbound
    const tunnels = nodes.filter((n: any) => n.type === 'tunnel');
    for (const tun of tunnels) {
        // Here we'd map the config. For Wireguard imported proxies, we parse the config.
        // For MVP, we'll just mock the outbound based on type if we don't have a parser.
        // We will output a basic wireguard stub for tun-airvpn or bypass/direct.
        if (tun.data?.type === 'Bypass') {
            // "direct" already exists
        } else if (tun.data?.type === 'Block') {
            // "block" already exists
        } else if (tun.data?.type === 'WireGuard') {
            outbounds.push({
                type: "wireguard",
                tag: tun.id,
                server: "1.1.1.1", // Placeholder, requires parsing WireGuard .conf
                server_port: 51820,
                local_address: ["10.0.0.2/32"],
                private_key: "...",
                peer_public_key: "..."
            });
        } else if (tun.data?.type === 'Hysteria2') {
            outbounds.push({
                type: "hysteria2",
                tag: tun.id,
                server: "2.2.2.2", // Placeholder
                server_port: 443,
                password: "...",
            });
        }
    }

    // Map edges for Standard mode apps
    for (const app of standardApps) {
        const appEdges = edges.filter((e: any) => e.source === app.id);
        if (appEdges.length > 0) {
            const edge = appEdges[0]; // Simple 1-to-1 routing
            const targetTun = tunnels.find((t: any) => t.id === edge.target);

            let outboundTag = "direct"; // Default
            if (targetTun) {
                if (targetTun.data?.type === 'Bypass') outboundTag = "direct";
                else if (targetTun.data?.type === 'Block') outboundTag = "block";
                else outboundTag = targetTun.id;
            }

            routeRules.push({
                process_name: [app.data.processName],
                outbound: outboundTag
            });
        }
    }

    // Default rule for anything not explicitly routed
    routeRules.push({
        outbound: "direct"
    });

    const config = {
        log: {
            level: "info"
        },
        inbounds: [
            {
                type: "tun",
                tag: "tun-in",
                interface_name: "tun0",
                inet4_address: "172.19.0.1/30",
                auto_route: true,
                strict_route: true,
                sniff: true,
                sniff_override_destination: true
            }
        ],
        outbounds: outbounds,
        route: {
            rules: routeRules,
            auto_detect_interface: true
        },
        dns: {
            servers: [
                {
                    tag: "dns-fakeip",
                    address: "fakeip"
                },
                {
                    tag: "dns-remote",
                    address: "https://1.1.1.1/dns-query",
                    detour: "direct"
                }
            ],
            rules: [
                {
                    outbound: ["any"],
                    server: "dns-fakeip"
                }
            ],
            fakeip: {
                enabled: true,
                inet4_range: "198.18.0.0/15",
                inet6_range: "fc00::/18"
            }
        }
    };

    return JSON.stringify(config, null, 2);
}
