import { main } from "../../wailsjs/go/models";

export interface RoutingRule {
    processName: string;
    tunnelId: string;
    tunnelLabel: string;
    tunnelType: string;
}

export function generateRoutingRules(state: main.AppState): RoutingRule[] {
    const nodes = state.canvasElements.filter((e: any) => e.position);
    const edges = state.canvasElements.filter((e: any) => e.source && e.target);

    const appNodes = nodes.filter((n: any) => n.type === 'application');
    const tunnelNodes = nodes.filter((n: any) => n.type === 'tunnel');
    const tunnelMap = new Map(tunnelNodes.map((t: any) => [t.id, t]));

    const rules: RoutingRule[] = [];

    for (const app of appNodes) {
        if (app.data?.mode === 'Strict') continue;

        const edge = edges.find((e: any) => e.source === app.id);
        if (!edge) continue;

        const tunnel = tunnelMap.get(edge.target);
        if (!tunnel) continue;

        const tunnelType = tunnel.data?.type || 'Bypass';
        if (tunnelType === 'Bypass') continue;

        const processName = app.data?.processName;
        if (!processName) continue;

        rules.push({
            processName,
            tunnelId: tunnel.id,
            tunnelLabel: tunnel.data?.label || tunnel.id,
            tunnelType,
        });
    }

    return rules;
}
