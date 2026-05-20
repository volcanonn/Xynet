import { computed, ref, watch, onUnmounted } from "vue";
import { useVueFlow } from "@vue-flow/core";

export function useWireStacking(
    targetId: string | (() => string),
    sourceId: string | (() => string | undefined),
    dragWire: any
) {
    const { edges, findNode } = useVueFlow();

    const rawOffset = computed(() => {
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

        if (!sId) return null;
        const index = uniqueSources.indexOf(sId);
        if (index === -1) return null;
        return (index - (uniqueSources.length - 1) / 2) * 16;
    });

    const animatedOffset = ref<number | null>(rawOffset.value);
    let animationFrame: number | null = null;
    let startTime: number | null = null;
    let startValue: number | null = rawOffset.value;

    watch(rawOffset, (newVal, oldVal) => {
        if (animationFrame) cancelAnimationFrame(animationFrame);
        
        if (newVal === null) {
            animatedOffset.value = null;
            return;
        }
        
        if (oldVal === null || animatedOffset.value === null) {
            animatedOffset.value = newVal;
            return;
        }

        startValue = animatedOffset.value;
        startTime = performance.now();
        
        const animate = (time: number) => {
            if (!startTime) startTime = time;
            const elapsed = time - startTime;
            const progress = Math.min(elapsed / 150, 1);
            
            const easeOutCubic = 1 - Math.pow(1 - progress, 3);
            animatedOffset.value = startValue! + (newVal - startValue!) * easeOutCubic;

            if (progress < 1) {
                animationFrame = requestAnimationFrame(animate);
            }
        };
        animationFrame = requestAnimationFrame(animate);
    });

    onUnmounted(() => {
        if (animationFrame) cancelAnimationFrame(animationFrame);
    });

    return animatedOffset;
}
