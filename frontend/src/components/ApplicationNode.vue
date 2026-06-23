<script setup lang="ts">
import { Handle, Position, useVueFlow } from "@vue-flow/core";
import type { NodeProps } from "@vue-flow/core";
import { computed, ref } from "vue";
import { Play } from "@lucide/vue";
import { useAppState } from "../composables/useAppState";
import { LaunchStrict } from "../../wailsjs/go/main/App";
import { useToast } from "../composables/useToast";

interface AppNodeData {
    label: string;
    icon?: string;
    processName?: string;
    mode?: "Standard" | "Strict";
}
const props = defineProps<NodeProps<AppNodeData>>();

let vueFlow: any = null;
try {
    vueFlow = useVueFlow();
} catch (e) {
    // Ignore error when rendered outside VueFlow context (drag preview)
}

const getConnectedEdges = vueFlow?.getConnectedEdges || (() => []);
const findNode = vueFlow?.findNode || (() => null);
const updateNodeData = vueFlow?.updateNodeData;
const hasConnection = computed(() => getConnectedEdges(props.id).length > 0);
const isInFlow = computed(() => !!findNode(props.id));

const mode = computed(() => props.data.mode || "Standard");
const launching = ref(false);
const toast = useToast();

const connectedTunnel = computed(() => {
    const edges = getConnectedEdges(props.id);
    if (edges.length === 0) return null;
    const edge = edges[0];
    const targetId = edge.source === props.id ? edge.target : edge.source;
    return findNode(targetId);
});

const canLaunch = computed(() => {
    if (mode.value !== "Strict") return false;
    const tunnel = connectedTunnel.value;
    if (!tunnel) return false;
    return tunnel.data?.type === "WireGuard";
});

const toggleMode = () => {
    if (!updateNodeData) return;
    const newMode = mode.value === "Standard" ? "Strict" : "Standard";
    if (newMode === "Strict") {
        if (!confirm("Switch to Strict mode?\n\nStrict mode launches the app inside a network namespace for complete isolation. You must use the Launch button to start it.")) return;
    }
    updateNodeData(props.id, { ...props.data, mode: newMode });
};

const launchApp = async () => {
    const tunnel = connectedTunnel.value;
    if (!tunnel || !props.data.processName) return;

    launching.value = true;
    try {
        await LaunchStrict(props.data.processName, tunnel.data.label);
    } catch (e) {
        toast.error(`Failed to launch: ${e}`);
    } finally {
        launching.value = false;
    }
};
</script>

<template>
    <div class="app-node" :class="{ 'strict-mode': mode === 'Strict' }">
        <div class="node-content">
            <div class="icon-container">
                <img v-if="data.icon" :src="data.icon" class="icon-img" />
                <div v-else class="icon-placeholder"></div>
            </div>
            <div class="info">
                <span class="label">{{ data.label }}</span>
                <button
                    v-if="isInFlow"
                    class="mode-badge"
                    :class="mode.toLowerCase()"
                    @click.stop="toggleMode"
                    :title="mode === 'Standard' ? 'Auto-routed via backend' : 'Launches inside network namespace'"
                >
                    {{ mode }}
                </button>
            </div>
            <button
                v-if="isInFlow && canLaunch"
                class="launch-btn"
                @click.stop="launchApp"
                :disabled="launching"
                title="Launch inside network namespace"
            >
                <Play :size="14" />
            </button>
        </div>

        <Handle
            v-if="isInFlow"
            id="source"
            type="source"
            :position="Position.Right"
            class="handle source-handle"
            :class="{ 'is-connected': hasConnection }"
            :connectable="!hasConnection"
        >
            <div class="plug-base"></div>
        </Handle>
    </div>
</template>

<style scoped>
.app-node {
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
.app-node:hover {
    border-color: #52525b;
}
.app-node.strict-mode {
    border-color: #a855f7;
}
.vue-flow__node-application.selected .app-node {
    border-color: var(--accent-primary);
    box-shadow: 0 0 0 1px var(--accent-primary);
}

.node-content {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    position: relative;
    z-index: 2;
}
.icon-container {
    width: 32px;
    height: 32px;
    border-radius: 0.5rem;
    background-color: #3f3f46;
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    overflow: hidden;
}
.icon-img {
    width: 24px;
    height: 24px;
    object-fit: contain;
}
.icon-placeholder {
    width: 16px;
    height: 16px;
    border-radius: 50%;
    background-color: #71717a;
}
.info {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
    overflow: hidden;
    flex: 1;
}
.label {
    font-size: 0.875rem;
    font-weight: 500;
    color: var(--text-primary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
}
.mode-badge {
    display: inline-block;
    width: fit-content;
    padding: 0.125rem 0.375rem;
    font-size: 0.625rem;
    font-weight: 600;
    font-family: inherit;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    border-radius: 0.25rem;
    border: none;
    cursor: pointer;
    transition: filter 0.2s;
}
.mode-badge:hover {
    filter: brightness(1.3);
}
.mode-badge.standard {
    background-color: rgba(34, 197, 94, 0.15);
    color: #22c55e;
}
.mode-badge.strict {
    background-color: rgba(168, 85, 247, 0.15);
    color: #a855f7;
}

.launch-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 28px;
    height: 28px;
    border-radius: 0.375rem;
    border: 1px solid #a855f7;
    background: rgba(168, 85, 247, 0.1);
    color: #a855f7;
    cursor: pointer;
    flex-shrink: 0;
    transition: all 0.2s;
}
.launch-btn:hover {
    background: rgba(168, 85, 247, 0.25);
}
.launch-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
}

.handle {
    width: 14px;
    height: 24px;
    border-radius: 0 12px 12px 0;
    background-color: var(--bg-card);
    border: 1px solid var(--border-color);
    border-left: none;
    transition: border-color 0.2s;
    right: -14px;
    top: 50%;
    transform: translateY(-50%);
    display: flex;
    align-items: center;
    justify-content: flex-end;
    z-index: 3;
}
.plug-base {
    width: 8px;
    height: 10px;
    background-color: var(--text-secondary);
    border-radius: 2px;
    margin-right: -4px;
    transition: background-color 0.2s;
}
.handle.is-connected {
    border-color: var(--accent-success);
    pointer-events: none;
}
.handle.is-connected .plug-base {
    background-color: var(--accent-success);
}
.handle:hover {
    border-color: var(--accent-primary);
}
.handle:hover .plug-base {
    background-color: var(--accent-primary);
}
</style>
