<script setup lang="ts">
import { Handle, Position, useVueFlow } from "@vue-flow/core";
import type { NodeProps } from "@vue-flow/core";
import { computed } from "vue";

interface AppNodeData {
    label: string;
    icon?: string;
    mode: "Standard" | "Strict";
}
const props = defineProps<NodeProps<AppNodeData>>();

const toggleMode = () => {
    const nextMode = props.data.mode === "Standard" ? "Strict" : "Standard";
    if (
        window.confirm(
            `Are you sure you want to switch ${props.data.label} to ${nextMode} mode?`,
        )
    ) {
        props.data.mode = nextMode;
    }
};

const { getConnectedEdges } = useVueFlow();
const hasConnection = computed(() => getConnectedEdges(props.id).length > 0);
</script>

<template>
    <div class="app-node">
        <div class="node-content">
            <div class="icon-container">
                <div class="icon-placeholder"></div>
            </div>
            <div class="info">
                <span class="label">{{ data.label }}</span>
                <span
                    class="mode-badge"
                    :class="data.mode.toLowerCase()"
                    @click="toggleMode"
                    title="Click to toggle mode"
                    >{{ data.mode }}</span
                >
            </div>
        </div>

        <Handle
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
    gap: 0.125rem;
    overflow: hidden;
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
    font-size: 0.65rem;
    font-weight: 600;
    padding: 0.125rem 0.375rem;
    border-radius: 0.25rem;
    display: inline-block;
    width: fit-content;
    cursor: pointer;
    transition: filter 0.2s;
}
.mode-badge:hover {
    filter: brightness(1.2);
}
.mode-badge.standard {
    background-color: rgba(59, 130, 246, 0.2);
    color: #60a5fa;
}
.mode-badge.strict {
    background-color: rgba(239, 68, 68, 0.2);
    color: #f87171;
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
