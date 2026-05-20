<script setup lang="ts">
import { ref, reactive, provide, markRaw, onUnmounted } from "vue";
import { VueFlow, useVueFlow, Position } from "@vue-flow/core";
import { Background } from "@vue-flow/background";
import { getBezierPath } from "@vue-flow/core";
import type { Node, Edge, Connection } from "@vue-flow/core";
import ApplicationNode from "./ApplicationNode.vue";
import TunnelNode from "./TunnelNode.vue";
import WireEdge from "./WireEdge.vue";
import "@vue-flow/core/dist/style.css";
import "@vue-flow/core/dist/theme-default.css";

const nodeTypes = {
    application: markRaw(ApplicationNode),
    tunnel: markRaw(TunnelNode),
};
const edgeTypes = { wire: markRaw(WireEdge) };

const {
    onConnect,
    addEdges,
    removeEdges,
    edges: currentEdges,
    getNodes,
    findNode,
    project,
    viewport,
} = useVueFlow();

// ── Unified drag state ──
// Covers BOTH new connections (from handle) and reconnections (grabbing a plug)
const dragWire = reactive({
    active: false,
    sourceId: "",
    sourceX: 0,
    sourceY: 0,
    mouseX: 0,
    mouseY: 0,
    // Only set during reconnections (grabbing existing plug)
    originalEdgeId: "",
    originalTarget: "",
    // Hit-test results
    hoveredTunnelId: null as string | null,
    hoveredAppId: null as string | null,
});
provide("dragWire", dragWire);

// ── Coordinate helpers ──
const screenToFlow = (clientX: number, clientY: number) => {
    const el = document.querySelector(".vue-flow");
    if (!el) return { x: 0, y: 0 };
    const rect = el.getBoundingClientRect();
    return project({ x: clientX - rect.left, y: clientY - rect.top });
};

const getSourceHandlePos = (nodeId: string) => {
    const node = findNode(nodeId);
    if (!node) return { x: 0, y: 0 };
    const x =
        (node.computedPosition?.x ?? node.position.x) +
        (node.dimensions?.width ?? 200) +
        14;
    const y =
        (node.computedPosition?.y ?? node.position.y) +
        (node.dimensions?.height ?? 80) / 2;
    return { x, y };
};

// ── Hit testing ──
const hitTest = (fx: number, fy: number) => {
    let tun: string | null = null;
    let app: string | null = null;
    for (const n of getNodes.value) {
        const nx = n.computedPosition?.x ?? n.position.x;
        const ny = n.computedPosition?.y ?? n.position.y;
        const w = n.dimensions?.width ?? 200;
        const h = n.dimensions?.height ?? 80;
        if (n.type === "tunnel") {
            if (fx >= nx - 60 && fx <= nx + w && fy >= ny - 20 && fy <= ny + h + 20)
                tun = n.id;
        } else if (n.type === "application") {
            if (fx >= nx - 20 && fx <= nx + w + 20 && fy >= ny - 20 && fy <= ny + h + 20)
                app = n.id;
        }
    }
    dragWire.hoveredTunnelId = tun;
    dragWire.hoveredAppId = app;
};

// ── Plug drag (reconnection) ──
const onDragMove = (e: PointerEvent) => {
    const pos = screenToFlow(e.clientX, e.clientY);
    dragWire.mouseX = pos.x;
    dragWire.mouseY = pos.y;
    hitTest(pos.x, pos.y);
};

const onDragEnd = () => {
    window.removeEventListener("pointermove", onDragMove);
    window.removeEventListener("pointerup", onDragEnd);

    if (dragWire.hoveredTunnelId) {
        // Drop on a tunnel → create new edge
        addEdges({
            id: `e-${dragWire.sourceId}-${dragWire.hoveredTunnelId}-${Date.now()}`,
            source: dragWire.sourceId,
            target: dragWire.hoveredTunnelId,
            type: "wire",
            style: { stroke: "var(--accent-success)", strokeWidth: 2 },
        });
    } else if (dragWire.hoveredAppId) {
        // Drop on an app → delete (don't recreate)
    } else {
        // Drop in void → snap back to original target
        if (dragWire.originalTarget) {
            addEdges({
                id: `e-${dragWire.sourceId}-${dragWire.originalTarget}-${Date.now()}`,
                source: dragWire.sourceId,
                target: dragWire.originalTarget,
                type: "wire",
                style: { stroke: "var(--accent-success)", strokeWidth: 2 },
            });
        }
    }

    dragWire.active = false;
    dragWire.sourceId = "";
    dragWire.originalEdgeId = "";
    dragWire.originalTarget = "";
    dragWire.hoveredTunnelId = null;
    dragWire.hoveredAppId = null;
};

// Exposed to WireEdge via provide
const startPlugDrag = (
    edgeId: string,
    sourceId: string,
    targetId: string,
    event: PointerEvent,
) => {
    // Remove the existing edge
    removeEdges([edgeId]);

    const srcPos = getSourceHandlePos(sourceId);
    const mousePos = screenToFlow(event.clientX, event.clientY);

    dragWire.active = true;
    dragWire.sourceId = sourceId;
    dragWire.sourceX = srcPos.x;
    dragWire.sourceY = srcPos.y;
    dragWire.mouseX = mousePos.x;
    dragWire.mouseY = mousePos.y;
    dragWire.originalEdgeId = edgeId;
    dragWire.originalTarget = targetId;
    dragWire.hoveredTunnelId = null;
    dragWire.hoveredAppId = null;

    hitTest(mousePos.x, mousePos.y);

    window.addEventListener("pointermove", onDragMove);
    window.addEventListener("pointerup", onDragEnd);
};
provide("startPlugDrag", startPlugDrag);

onUnmounted(() => {
    window.removeEventListener("pointermove", onDragMove);
    window.removeEventListener("pointerup", onDragEnd);
});

// ── New connection from handle ──
// Vue Flow's built-in connection system handles the visual wire via #connection-line.
// We just need to track hover state for ghost plugs.
const onConnectStart = (event: any) => {
    dragWire.active = true;
    dragWire.sourceId = event?.nodeId ?? "";
    const srcPos = getSourceHandlePos(dragWire.sourceId);
    dragWire.sourceX = srcPos.x;
    dragWire.sourceY = srcPos.y;
    dragWire.originalEdgeId = "";
    dragWire.originalTarget = "";
    
    if (event?.event instanceof MouseEvent) {
        const pos = screenToFlow(event.event.clientX, event.event.clientY);
        dragWire.mouseX = pos.x;
        dragWire.mouseY = pos.y;
    }
    window.addEventListener("pointermove", onDragMove);
    window.addEventListener("pointerup", onDragEnd);
};

// When Vue Flow fires a successful new connection
onConnect((connection: Connection) => {
    const existingEdge = currentEdges.value.find(
        (e) => e.source === connection.source,
    );
    if (existingEdge) removeEdges([existingEdge.id]);
    addEdges({
        id: `e-${connection.source}-${connection.target}-${Date.now()}`,
        source: connection.source,
        target: connection.target,
        type: "wire",
        style: { stroke: "var(--accent-success)", strokeWidth: 2 },
    });
});

// ── Node data ──
const nodes = ref<Node[]>([
    { id: "app-firefox", type: "application", position: { x: 100, y: 100 }, data: { label: "Firefox", mode: "Standard" } },
    { id: "app-qbit", type: "application", position: { x: 100, y: 250 }, data: { label: "qBittorrent", mode: "Strict" } },
    { id: "app-steam", type: "application", position: { x: 100, y: 400 }, data: { label: "Steam", mode: "Standard" } },
    { id: "tun-airvpn", type: "tunnel", position: { x: 600, y: 100 }, data: { label: "AirVPN-US", type: "WireGuard", latency: "34ms" } },
    { id: "tun-hysteria", type: "tunnel", position: { x: 600, y: 250 }, data: { label: "Home Server", type: "Hysteria2", latency: "12ms" } },
    { id: "tun-bypass", type: "tunnel", position: { x: 600, y: 400 }, data: { label: "Direct / Bypass", type: "Bypass" } },
]);

const edges = ref<Edge[]>([
    { id: "e-firefox-airvpn", source: "app-firefox", target: "tun-airvpn", type: "wire", style: { stroke: "var(--accent-success)", strokeWidth: 2 } },
    { id: "e-qbit-airvpn", source: "app-qbit", target: "tun-airvpn", type: "wire", style: { stroke: "var(--accent-success)", strokeWidth: 2 } },
]);

// ── Drag wire bezier (for reconnection overlay) ──
const dragPath = () => {
    return getBezierPath({
        sourceX: dragWire.sourceX,
        sourceY: dragWire.sourceY,
        sourcePosition: Position.Right,
        targetX: dragWire.mouseX - 12,
        targetY: dragWire.mouseY,
        targetPosition: Position.Left,
    })[0];
};
</script>

<template>
    <div class="canvas-wrapper">
        <VueFlow
            :nodes="nodes"
            :edges="edges"
            :node-types="nodeTypes"
            :edge-types="edgeTypes"
            @connect-start="onConnectStart"
            @connect-end="onDragEnd"
            :default-zoom="1"
            :min-zoom="0.5"
            :max-zoom="2"
            fit-view-on-init
            class="xynet-theme"
        >
            <Background pattern-color="#27272a" />

            <!-- Connection line for NEW connections from app handle -->
            <template #connection-line>
                <g style="opacity: 0.8; pointer-events: none">
                    <path
                        :d="dragPath()"
                        fill="none" stroke="var(--accent-success)" stroke-width="2"
                    />
                    <g :transform="`translate(${dragWire.mouseX}, ${dragWire.mouseY})`">
                        <rect x="-12" y="-4" width="12" height="8" rx="2" fill="var(--bg-card)" stroke="var(--accent-success)" stroke-width="1.5" />
                        <path d="M 0 -2 L 4 -2 M 0 2 L 4 2" stroke="var(--accent-success)" stroke-width="1.5" stroke-linecap="round" />
                        <!-- Show Red X if hovering an app (top right offset) -->
                        <g v-if="dragWire.hoveredAppId" transform="translate(8, -12)">
                            <path d="M -4 -4 L 4 4 M -4 4 L 4 -4" stroke="var(--accent-danger)" stroke-width="2" stroke-linecap="round" />
                        </g>
                    </g>
                </g>
            </template>
        </VueFlow>

        <!-- Custom drag overlay for reconnections (raw pointer events, not Vue Flow) -->
        <svg
            v-if="dragWire.active && dragWire.originalEdgeId"
            class="drag-overlay"
        >
            <g :transform="`translate(${viewport.x}, ${viewport.y}) scale(${viewport.zoom})`">
                <path
                    :d="dragPath()"
                    fill="none" stroke="var(--accent-success)" stroke-width="2" opacity="0.8"
                />
                <!-- Show plug and optionally the Red X if hovering an app -->
                <g :transform="`translate(${dragWire.mouseX}, ${dragWire.mouseY})`">
                    <rect x="-12" y="-4" width="12" height="8" rx="2" fill="var(--bg-card)" stroke="var(--accent-success)" stroke-width="1.5" />
                    <path d="M 0 -2 L 4 -2 M 0 2 L 4 2" stroke="var(--accent-success)" stroke-width="1.5" stroke-linecap="round" />
                    <g v-if="dragWire.hoveredAppId" transform="translate(8, -12)">
                        <path d="M -4 -4 L 4 4 M -4 4 L 4 -4" stroke="var(--accent-danger)" stroke-width="2" stroke-linecap="round" />
                    </g>
                </g>
            </g>
        </svg>
    </div>
</template>

<style>
.canvas-wrapper {
    width: 100%;
    height: 100%;
    position: absolute;
    top: 0;
    left: 0;
}
.drag-overlay {
    position: absolute;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    pointer-events: none;
    z-index: 5;
}
.xynet-theme .vue-flow__edge-path {
    stroke: var(--border-color);
    stroke-width: 2;
}
.xynet-theme .vue-flow__edge.selected .vue-flow__edge-path {
    stroke: var(--accent-primary);
}
.xynet-theme .vue-flow__connection-path {
    stroke: var(--text-secondary);
    stroke-width: 2;
}
.vue-flow__node {
    user-select: none;
}
</style>
