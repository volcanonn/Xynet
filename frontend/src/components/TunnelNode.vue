<script setup lang="ts">
import { Handle, Position, useVueFlow } from "@vue-flow/core";
import type { NodeProps } from "@vue-flow/core";
import { Shield } from "@lucide/vue";
import { computed, inject, onMounted, ref } from "vue";
import { useWireStacking } from "../composables/useWireStacking";
import { useAppState } from "../composables/useAppState";
import { useActiveDeployment } from "../composables/useActiveDeployment";
import { MeasureLatency } from "../../wailsjs/go/main/App";

interface TunnelNodeData {
    label: string;
    latency?: string;
    type: "WireGuard" | "Hysteria2" | "Bypass" | "Block";
}
const props = defineProps<NodeProps<TunnelNodeData>>();
let vueFlow: any = null;
try {
    vueFlow = useVueFlow();
} catch (e) {
    // Ignore error when rendered outside VueFlow context (drag preview)
}

const edges = vueFlow?.edges || { value: [] };
const findNode = vueFlow?.findNode || (() => null);
const dragWire = inject<any>("dragWire");
const getLatencyColor = (latency?: string) => {
    if (!latency) return "var(--text-secondary)";
    const ms = parseInt(latency.replace("ms", ""));
    if (isNaN(ms)) return "var(--text-secondary)";
    if (ms < 50) return "var(--accent-success)";
    if (ms < 150) return "var(--accent-warning)";
    return "var(--accent-danger)";
};

// Directly uses our math state. If mouse leaves the node area, this becomes FALSE instantly.
const isHovered = computed(() => dragWire?.hoveredTunnelId === props.id);

const connectedEdges = computed(() =>
    edges.value.filter(
        (e) => e.target === props.id && e.id !== dragWire?.originalEdgeId,
    ),
);
const isEmpty = computed(() => connectedEdges.value.length === 0);
const isInFlow = computed(() => !!findNode(props.id));


const ghostOffset = useWireStacking(
    () => props.id,
    () => (isHovered.value ? dragWire?.sourceId : undefined),
    dragWire
);

const { appState } = useAppState();
const measuredLatency = ref<string>("");
const { activeRules } = useActiveDeployment();
const isActive = computed(() => {
    return activeRules.value.some((r: any) => r.tunnelId === props.id);
});

onMounted(async () => {
    if (props.data.type === 'WireGuard' || props.data.type === 'Hysteria2') {
        const proxy = appState.value?.proxies?.find(p => p.name.replace(/\.(conf|txt)$/, '') === props.data.label);
        if (proxy) {
            try {
                measuredLatency.value = await MeasureLatency(proxy.content);
            } catch (e) {
                measuredLatency.value = "Error";
            }
        }
    }
});;
</script>

<template>
    <div class="tunnel-node" :class="{ 'pulse-animate': isHovered && isEmpty, 'is-active-route': isActive }">
        <Handle
            v-if="isInFlow"
            id="target"
            type="target"
            :position="Position.Left"
            class="target-handle"
            :connectable-start="false"
        />

        <div
            class="ghost-plug"
            v-if="isHovered"
            :style="{ transform: `translateY(calc(-50% + ${ghostOffset}px))` }"
        >
            <svg width="16" height="8" style="overflow: visible">
                <rect
                    x="0"
                    y="0"
                    width="12"
                    height="8"
                    rx="2"
                    fill="var(--bg-card)"
                    stroke="var(--text-secondary)"
                    stroke-width="1.5"
                    stroke-dasharray="2 2"
                />
                <path
                    d="M 12 2 L 16 2 M 12 6 L 16 6"
                    stroke="var(--text-secondary)"
                    stroke-width="1.5"
                    stroke-linecap="round"
                />
            </svg>
        </div>

        <div class="node-content">
            <div class="header">
                <div class="icon-container" :class="data.type.toLowerCase()">
                    <Shield :size="14" />
                </div>
                <span class="type-label">{{ data.type }}</span>
            </div>

            <div class="info">
                <span class="label">{{ data.label }}</span>
                <span
                    class="latency"
                    :style="{ color: getLatencyColor(measuredLatency || data.latency) }"
                    >{{ measuredLatency || data.latency || "--" }}</span
                >
            </div>
        </div>
    </div>
</template>

<style scoped>
.tunnel-node {
    width: 200px;
    background-color: var(--bg-card);
    border: 1px solid var(--border-color);
    border-radius: 0.75rem;
    padding: 0.75rem;
    transition:
        border-color 0.2s,
        box-shadow 0.2s;
    position: relative;
}
.tunnel-node:hover {
    border-color: #52525b;
}
.vue-flow__node-tunnel.selected .tunnel-node {
    border-color: var(--accent-primary);
    box-shadow: 0 0 0 1px var(--accent-primary);
}

@keyframes pulse-glow {
    0% {
        box-shadow: 0 0 0 0 rgba(34, 197, 94, 0.4);
        border-color: var(--accent-success);
    }
    70% {
        box-shadow: 0 0 0 8px rgba(34, 197, 94, 0);
        border-color: var(--accent-success);
    }
    100% {
        box-shadow: 0 0 0 0 rgba(34, 197, 94, 0);
        border-color: var(--border-color);
    }
}
.tunnel-node.pulse-animate {
    animation: pulse-glow 1.5s infinite;
    border-color: var(--accent-success);
}
.tunnel-node.is-active-route {
    border-color: var(--accent-success);
    box-shadow: 0 0 0 1px var(--accent-success);
}

.target-handle {
    width: 10px;
    height: 10px;
    left: -5px;
    top: 50%;
    transform: translateY(-50%);
    background: transparent;
    border: none;
    border-radius: 0;
    opacity: 0;
    pointer-events: none !important;
}

.ghost-plug {
    position: absolute;
    left: -12px;
    top: 50%;
    pointer-events: none;
    z-index: 10;
}
.node-content {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
}
.header {
    display: flex;
    align-items: center;
    gap: 0.5rem;
}
.icon-container {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 20px;
    height: 20px;
    border-radius: 0.25rem;
    background-color: #3f3f46;
    color: var(--text-primary);
}
.icon-container.wireguard {
    color: #3b82f6;
}
.icon-container.hysteria2 {
    color: #a855f7;
}
.icon-container.bypass {
    color: #64748b;
}
.icon-container.block {
    color: #ef4444;
}
.type-label {
    font-size: 0.65rem;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--text-secondary);
}
.info {
    display: flex;
    align-items: center;
    justify-content: space-between;
}
.label {
    font-size: 0.875rem;
    font-weight: 500;
    color: var(--text-primary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
}
.latency {
    font-size: 0.75rem;
    font-weight: 600;
    font-variant-numeric: tabular-nums;
}
</style>
