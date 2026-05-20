<script setup lang="ts">
import { BaseEdge, getBezierPath, useVueFlow, Position } from "@vue-flow/core";
import type { EdgeProps } from "@vue-flow/core";
import { computed, inject } from "vue";

const props = defineProps<EdgeProps>();
const { edges, findNode } = useVueFlow();

const dragWire = inject<any>("dragWire");
const startPlugDrag = inject<Function>("startPlugDrag");

// Target offset for stacking
const targetOffset = computed(() => {
    const connectedSources = edges.value
        .filter((e) => e.target === props.target)
        .map((e) => e.source);

    // If there is an active drag, and it is hovering over our target, we need to make space!
    if (dragWire?.active && dragWire?.hoveredTunnelId === props.target && dragWire?.sourceId) {
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

    const index = uniqueSources.indexOf(props.source);
    if (index === -1) return 0;
    return (index - (uniqueSources.length - 1) / 2) * 16;
});

const activeTargetX = computed(() => {
    const node = findNode(props.target);
    if (!node) return props.targetX;
    return (node.computedPosition?.x ?? node.position.x) + 1;
});
const activeTargetY = computed(() => props.targetY + targetOffset.value);

const pathParams = computed(() => ({
    sourceX: props.sourceX - 14,
    sourceY: props.sourceY,
    sourcePosition: Position.Right,
    targetX: activeTargetX.value - 12, // Wire ends at back of plug
    targetY: activeTargetY.value,
    targetPosition: Position.Left,
}));
const path = computed(() => getBezierPath(pathParams.value));

const onPointerDown = (e: PointerEvent) => {
    e.stopPropagation();
    if (startPlugDrag) {
        startPlugDrag(props.id, props.source, props.target, e);
    }
};
</script>

<script lang="ts">
export default { inheritAttrs: false };
</script>

<template>
    <g class="wire-edge-group">
        <!-- Main wire -->
        <BaseEdge :path="path[0]" :style="props.style" style="pointer-events: none;" />

        <!-- Interactive Plug -->
        <g
            :transform="`translate(${activeTargetX}, ${activeTargetY})`"
            style="pointer-events: all; cursor: grab; transition: transform 0.15s ease;"
            @pointerdown="onPointerDown"
        >
            <!-- Invisible larger hit area for easier grabbing -->
            <rect x="-20" y="-15" width="30" height="30" fill="transparent" />
            
            <rect
                x="-12"
                y="-4"
                width="12"
                height="8"
                rx="2"
                fill="var(--bg-card)"
                stroke="var(--accent-success)"
                stroke-width="1.5"
            />
            <path
                d="M 0 -2 L 4 -2 M 0 2 L 4 2"
                stroke="var(--accent-success)"
                stroke-width="1.5"
                stroke-linecap="round"
            />
        </g>
    </g>
</template>
