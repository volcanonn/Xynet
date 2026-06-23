export function generateRoutingRules(state: any): any[] {
    if (!state?.canvasElements) return [];

    const nodes = state.canvasElements.filter((e: any) => e.type === 'application' || e.type === 'tunnel');
    const edges = state.canvasElements.filter((e: any) => e.source && e.target);

    const rules: any[] = [];

    edges.forEach((edge: any) => {
        const sourceNode = nodes.find((n: any) => n.id === edge.source);
        const targetNode = nodes.find((n: any) => n.id === edge.target);

        if (sourceNode && targetNode && sourceNode.data?.mode !== 'Strict') {
            let processName = sourceNode.data.processName;
            
            rules.push({
                processName: processName,
                tunnelId: targetNode.id,
                tunnelLabel: targetNode.data.label,
                tunnelType: targetNode.data.type || 'WireGuard',
            });
        }
    });

    return rules;
}
