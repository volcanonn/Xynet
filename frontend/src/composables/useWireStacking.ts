import { computed } from "vue";
import { useVueFlow } from "@vue-flow/core";

export function useWireStacking(
    targetId: string | (() => string),
    sourceId: string | (() => string | undefined),
    dragWire: any
) {
    const { edges, findNode } = useVueFlow();

    return computed(() => {
        const tId = typeof targetId === "function" ? targetId() : targetId;
        const sId = typeof sourceId === "function" ? sourceId() : sourceId;

        const connectedSources = edges.value
            .filter((e) => e.target === tId && e.id !== dragWire?.originalEdgeId)
            .map((e) => e.source);

        if (dragWire?.active && dragWire?.hoveredTunnelId === tId && dragWire?.sourceId) {
            if (!connectedSources.includes(dragWire.sourceId)) {
                connectedSources.push(dragWire.sourceId);
            }
        }

        const uniqueSources = Array.from(new Set(connectedSources));
        uniqueSources.sort((a, b) => {
            const nodeA = findNode(a);
            const nodeB = findNode(b);
            const yA = nodeA?.computedPosition?.y ?? nodeA?.position?.y ?? 0;
            const yB = nodeB?.computedPosition?.y ?? nodeB?.position?.y ?? 0;
            return yA - yB;
        });

        if (!sId) return 0;
        const index = uniqueSources.indexOf(sId);
        if (index === -1) return 0;
        return (index - (uniqueSources.length - 1) / 2) * 16;
    });
}
