<script setup lang="ts">
import { BaseEdge, getBezierPath, useVueFlow, Position } from "@vue-flow/core";
import type { EdgeProps } from "@vue-flow/core";
import { computed, inject } from "vue";
import { useWireStacking } from "../composables/useWireStacking";
import { useActiveDeployment } from "../composables/useActiveDeployment";
import { useAppState } from "../composables/useAppState";

const props = defineProps<EdgeProps>();
const { edges } = useVueFlow();

const dragWire = inject<any>("dragWire");
const startPlugDrag = inject<Function>("startPlugDrag");
const { activeRules, voponoApps } = useActiveDeployment();
const { appState } = useAppState();

const strokeColor = computed(() => {
    const sourceNode = appState.value?.canvasElements?.find((el: any) => el.id === props.source);
    const targetNode = appState.value?.canvasElements?.find((el: any) => el.id === props.target);

    const processName = sourceNode?.data?.processName;
    const tunnelLabel = targetNode?.data?.label;
    const mode = sourceNode?.data?.mode || 'Standard';

    if (mode === 'Strict' && processName && tunnelLabel) {
        const isStrictActive = voponoApps.value.some((p: any) =>
            p.appName === processName && p.configName === tunnelLabel
        );
        if (isStrictActive) return '#a855f7';
    }

    let isStandardActive = false;
    if (processName) {
        isStandardActive = activeRules.value.some((r: any) =>
            r.tunnelId === props.target && r.processName === processName
        );
    } else {
        isStandardActive = activeRules.value.some((r: any) => r.tunnelId === props.target);
    }

    if (isStandardActive) return 'var(--accent-success)';

    return 'var(--border-color)';
});

const targetOffset = useWireStacking(
    () => props.target,
    () => props.source,
    dragWire
);

const activeTargetX = computed(() => props.targetX + 6);
const activeTargetY = computed(() => props.targetY + (targetOffset.value ?? 0));

const pathParams = computed(() => ({
    sourceX: props.sourceX,
    sourceY: props.sourceY,
    sourcePosition: Position.Right,
    targetX: activeTargetX.value - 12,
    targetY: activeTargetY.value,
    targetPosition: Position.Left,
}));
const path = computed(() => getBezierPath(pathParams.value));

const onPointerDown = (e: PointerEvent) => {
    e.stopPropagation();
    e.preventDefault();
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
        <BaseEdge :path="path[0]" :style="{ ...props.style, stroke: strokeColor }" style="pointer-events: none;" />

        <g
            :transform="`translate(${activeTargetX}, ${activeTargetY})`"
            style="pointer-events: all; cursor: grab;"
            @pointerdown="onPointerDown"
        >
            <rect x="-20" y="-15" width="30" height="30" fill="transparent" />

            <rect
                x="-12"
                y="-4"
                width="12"
                height="8"
                rx="2"
                fill="var(--bg-card)"
                :stroke="strokeColor"
                stroke-width="1.5"
            />
            <path
                d="M 0 -2 L 4 -2 M 0 2 L 4 2"
                :stroke="strokeColor"
                stroke-width="1.5"
                stroke-linecap="round"
            />
        </g>
    </g>
</template>