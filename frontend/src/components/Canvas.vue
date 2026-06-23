<script setup lang="ts">
import { ref, reactive, provide, markRaw, onUnmounted, onMounted } from "vue";
import { VueFlow, useVueFlow, Position } from "@vue-flow/core";
import { Background } from "@vue-flow/background";
import { Panel } from "@vue-flow/core";
import { Plus, Trash2, Box, Globe, Upload } from "@lucide/vue";
import { getBezierPath } from "@vue-flow/core";
import type { Node, Edge, Connection } from "@vue-flow/core";
import ApplicationNode from "./ApplicationNode.vue";
import TunnelNode from "./TunnelNode.vue";
import WireEdge from "./WireEdge.vue";
import "@vue-flow/core/dist/style.css";
import "@vue-flow/core/dist/theme-default.css";
import { useAppState } from "../composables/useAppState";
import { useToast } from "../composables/useToast";
import { ListDesktopApps, ImportWireguardConfig } from "../../wailsjs/go/main/App";

const nodeTypes = {
    application: markRaw(ApplicationNode),
    tunnel: markRaw(TunnelNode),
};
const edgeTypes = { wire: markRaw(WireEdge) };

const { appState, loadState, saveState } = useAppState();

const {
    onConnect,
    addEdges,
    addNodes,
    onNodeDragStop,
    removeNodes,
    removeEdges,
    edges: currentEdges,
    getNodes,
    findNode,
    project,
    viewport,
    toObject,
    onNodesChange,
    onEdgesChange,
} = useVueFlow();

let saveTimeout: any = null;
const triggerSave = () => {
    if (saveTimeout) clearTimeout(saveTimeout);
    saveTimeout = setTimeout(() => {
        if (appState.value) {
            const obj = toObject();
            appState.value.canvasElements = [...obj.nodes, ...obj.edges];
            saveState();
        }
    }, 500);
};

onNodesChange(() => triggerSave());
onEdgesChange(() => triggerSave());

onNodeDragStop(({ node, event }) => {
    const trashEl = document.getElementById("trash-zone");
    if (!trashEl) return;

    const rect = trashEl.getBoundingClientRect();
    if (
        event.clientX >= rect.left &&
        event.clientX <= rect.right &&
        event.clientY >= rect.top &&
        event.clientY <= rect.bottom
    ) {
        // Prevent deleting default nodes
        if (node.deletable !== false) {
            removeNodes([node.id]);

            // Add back to lists
            if (node.type === "application") {
                availableInputs.value.push({
                    label: node.data.label,
                    icon: node.data.icon,
                    processName: node.data.processName,
                });
            } else if (node.type === "tunnel") {
                availableOutputs.value.push({
                    label: node.data.label,
                    type: node.data.type,
                    latency: node.data.latency,
                });
            }
        }
    }
});

onMounted(async () => {
    await loadState();

    // Load real desktop applications
    try {
        const desktopApps = await ListDesktopApps();
        availableInputs.value = desktopApps.map((app: any) => ({
            label: app.name,
            icon: app.icon,
            processName: app.processName,
        }));
    } catch (e) {
        console.error("Failed to load desktop apps:", e);
    }

    // Build tunnel list from imported proxies
    if (appState.value?.proxies) {
        availableOutputs.value = appState.value.proxies.map((p: any) => ({
            label: p.name.replace(/\.(conf|txt)$/, ""),
            type: p.content.trim().startsWith('hysteria2://') ? 'Hysteria2' : 'WireGuard',
            latency: "--",
        }));
    }

    if (appState.value && appState.value.canvasElements) {
        const savedNodes = appState.value.canvasElements.filter(
            (e: any) => e.position,
        );
        const savedEdges = appState.value.canvasElements.filter(
            (e: any) => e.source && e.target,
        );
        nodes.value = savedNodes;
        edges.value = savedEdges;

        // Filter out items that are already placed on the canvas
        availableInputs.value = availableInputs.value.filter(
            (input) =>
                !savedNodes.some(
                    (n: any) =>
                        n.type === "application" &&
                        n.data?.label === input.label,
                ),
        );
        availableOutputs.value = availableOutputs.value.filter(
            (output) =>
                !savedNodes.some(
                    (n: any) =>
                        n.type === "tunnel" && n.data?.label === output.label,
                ),
        );
    }
});

// ── Unified drag state ──
const dragWire = reactive({
    active: false,
    sourceId: "",
    sourceX: 0,
    sourceY: 0,
    mouseX: 0,
    mouseY: 0,
    originalEdgeId: "",
    originalTarget: "",
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
            if (
                fx >= nx - 60 &&
                fx <= nx + w &&
                fy >= ny - 20 &&
                fy <= ny + h + 20
            )
                tun = n.id;
        } else if (n.type === "application") {
            if (
                fx >= nx - 20 &&
                fx <= nx + w + 20 &&
                fy >= ny - 20 &&
                fy <= ny + h + 20
            )
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
        addEdges({
            id: `e-${dragWire.sourceId}-${dragWire.hoveredTunnelId}-${Date.now()}`,
            source: dragWire.sourceId,
            target: dragWire.hoveredTunnelId,
            type: "wire",
            style: { stroke: "var(--accent-success)", strokeWidth: 2 },
        });
    } else if (dragWire.originalTarget && !dragWire.hoveredAppId) {
        addEdges({
            id: `e-${dragWire.sourceId}-${dragWire.originalTarget}-${Date.now()}`,
            source: dragWire.sourceId,
            target: dragWire.originalTarget,
            type: "wire",
            style: { stroke: "var(--accent-success)", strokeWidth: 2 },
        });
    }

    dragWire.active = false;
    dragWire.sourceId = "";
    dragWire.originalEdgeId = "";
    dragWire.originalTarget = "";
    dragWire.hoveredTunnelId = null;
    dragWire.hoveredAppId = null;
};

provide("vue-flow", {
    getConnectedEdges: () => [],
    updateNodeData: () => {},
    findNode: () => null,
});

const startPlugDrag = (
    edgeId: string,
    sourceId: string,
    targetId: string,
    event: PointerEvent,
) => {
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
    if (saveTimeout) clearTimeout(saveTimeout);
    window.removeEventListener("pointermove", onDragMove);
    window.removeEventListener("pointerup", onDragEnd);
});

// ── New connection from handle ──
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
const nodes = ref<Node[]>([]);
const edges = ref<Edge[]>([]);

// ── Application & Tunnel Lists (populated on mount) ──
const availableInputs = ref<any[]>([]);
const availableOutputs = ref<any[]>([]);

const showInputsList = ref(false);
const showOutputsList = ref(false);
const toast = useToast();

const importConfig = async () => {
    try {
        const proxy = await ImportWireguardConfig();
        if (!proxy || !proxy.name) return;
        if (appState.value) {
            if (!appState.value.proxies) appState.value.proxies = [];
            appState.value.proxies.push(proxy);
            const tunnelLabel = proxy.name.replace(/\.(conf|txt)$/, "");
            availableOutputs.value.push({
                label: tunnelLabel,
                type: proxy.content.trim().startsWith('hysteria2://') ? 'Hysteria2' : 'WireGuard',
                latency: "--",
            });
            saveState();
            toast.success('Config imported successfully');
        }
    } catch (e) {
        toast.error(`Import failed: ${e}`);
    }
};

const dragPreview = reactive({
    active: false,
    type: "" as "application" | "tunnel",
    data: null as any,
    startX: 0,
    startY: 0,
    x: 0,
    y: 0,
    pulledOut: false,
});

let capturedElement: HTMLElement | null = null;

const onPointerDownItem = (
    event: PointerEvent,
    type: "application" | "tunnel",
    data: any,
) => {
    if (event.button !== 0) return; // only left click

    // Prevent the default browser drag/selection behaviors so our custom logic runs perfectly
    event.preventDefault();

    capturedElement = event.currentTarget as HTMLElement;
    capturedElement.setPointerCapture(event.pointerId);

    dragPreview.active = true;
    dragPreview.type = type;
    dragPreview.data = data;
    dragPreview.startX = event.clientX;
    dragPreview.startY = event.clientY;
    dragPreview.x = event.clientX;
    dragPreview.y = event.clientY;
    dragPreview.pulledOut = false;

    window.addEventListener("pointermove", onPointerMoveItem);
    window.addEventListener("pointerup", onPointerUpItem);
    window.addEventListener("pointercancel", onPointerUpItem);
};

const onPointerMoveItem = (event: PointerEvent) => {
    if (!dragPreview.active) return;
    dragPreview.x = event.clientX;
    dragPreview.y = event.clientY;

    if (!dragPreview.pulledOut) {
        const dist = Math.hypot(
            event.clientX - dragPreview.startX,
            event.clientY - dragPreview.startY,
        );
        if (dist > 60) {
            dragPreview.pulledOut = true;
        }
    }
};

const onPointerUpItem = (event: PointerEvent) => {
    // Release pointer capture gracefully
    if (
        capturedElement &&
        typeof capturedElement.releasePointerCapture === "function" &&
        capturedElement.hasPointerCapture(event.pointerId)
    ) {
        capturedElement.releasePointerCapture(event.pointerId);
    }
    capturedElement = null;

    // Clean up event listeners immediately
    window.removeEventListener("pointermove", onPointerMoveItem);
    window.removeEventListener("pointerup", onPointerUpItem);
    window.removeEventListener("pointercancel", onPointerUpItem);

    if (!dragPreview.active) return;

    const canvasEl = document.querySelector(".vue-flow");
    const rect = canvasEl?.getBoundingClientRect();

    let droppedOnCanvas = false;
    let droppedOnToolbar = false;

    const toolbarEl = document.querySelector(".toolbar-panel");
    const tbRect = toolbarEl?.getBoundingClientRect();
    if (
        tbRect &&
        event.clientX >= tbRect.left &&
        event.clientX <= tbRect.right &&
        event.clientY >= tbRect.top &&
        event.clientY <= tbRect.bottom
    ) {
        droppedOnToolbar = true;
    }

    if (
        rect &&
        event.clientX >= rect.left &&
        event.clientX <= rect.right &&
        event.clientY >= rect.top &&
        event.clientY <= rect.bottom &&
        !droppedOnToolbar
    ) {
        droppedOnCanvas = true;
    }

    if (droppedOnCanvas && dragPreview.pulledOut) {
        const position = screenToFlow(event.clientX, event.clientY);
        // Center the node on cursor
        position.x -= 100;
        position.y -= 40;

        const newNode = {
            id: `node-${Date.now()}`,
            type: dragPreview.type,
            position,
            data:
                dragPreview.type === "application"
                    ? { ...dragPreview.data, mode: "Standard" }
                    : dragPreview.data,
            deletable: true,
        };

        // Remove item from available lists
        if (dragPreview.type === "application") {
            availableInputs.value = availableInputs.value.filter(
                (i) => i.label !== dragPreview.data.label,
            );
        } else {
            availableOutputs.value = availableOutputs.value.filter(
                (i) => i.label !== dragPreview.data.label,
            );
        }

        addNodes([newNode]);
        triggerSave();

        showInputsList.value = false;
        showOutputsList.value = false;
    }

    dragPreview.active = false;
    dragPreview.data = null;
    dragPreview.pulledOut = false;
};

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
    <div class="canvas-wrapper" :class="{ 'is-dragging': dragPreview.active }">
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

            <Panel position="top-center" class="toolbar-panel">
                <div class="toolbar-controls">
                    <div
                        id="trash-zone"
                        class="toolbar-btn trash-btn"
                        title="Drag nodes here to delete"
                    >
                        <Trash2 :size="20" />
                    </div>

                    <div class="dropdown-wrapper">
                        <button
                            class="toolbar-btn"
                            @click="
                                showInputsList = !showInputsList;
                                showOutputsList = false;
                            "
                        >
                            <Box :size="20" />
                            <span>Applications</span>
                        </button>
                        <transition-group
                            name="list"
                            tag="div"
                            v-if="showInputsList"
                            class="dropdown-menu"
                        >
                            <div
                                v-if="availableInputs.length === 0"
                                key="empty-inputs"
                                class="empty-msg"
                            >
                                No more applications
                            </div>
                            <div
                                v-for="item in availableInputs"
                                :key="item.label"
                                class="dropdown-item"
                                :class="{
                                    'ghost-hidden':
                                        dragPreview.active &&
                                        dragPreview.data?.label ===
                                            item.label &&
                                        !dragPreview.pulledOut,
                                    'collapsed-hidden':
                                        dragPreview.active &&
                                        dragPreview.data?.label ===
                                            item.label &&
                                        dragPreview.pulledOut,
                                }"
                                @pointerdown="
                                    onPointerDownItem(
                                        $event,
                                        'application',
                                        item,
                                    )
                                "
                            >
                                <img v-if="item.icon" :src="item.icon" class="dropdown-icon" />
                                <div v-else class="dropdown-icon-placeholder"></div>
                                {{ item.label }}
                            </div>
                        </transition-group>
                    </div>

                    <div class="dropdown-wrapper">
                        <button
                            class="toolbar-btn"
                            @click="
                                showOutputsList = !showOutputsList;
                                showInputsList = false;
                            "
                        >
                            <Globe :size="20" />
                            <span>Tunnels</span>
                        </button>
                        <transition-group
                            name="list"
                            tag="div"
                            v-if="showOutputsList"
                            class="dropdown-menu"
                        >
                            <div
                                v-if="availableOutputs.length === 0"
                                key="empty-outputs"
                                class="empty-msg"
                            >
                                No more tunnels
                            </div>
                            <div
                                v-for="item in availableOutputs"
                                :key="item.label"
                                class="dropdown-item"
                                :class="{
                                    'ghost-hidden':
                                        dragPreview.active &&
                                        dragPreview.data?.label ===
                                            item.label &&
                                        !dragPreview.pulledOut,
                                    'collapsed-hidden':
                                        dragPreview.active &&
                                        dragPreview.data?.label ===
                                            item.label &&
                                        dragPreview.pulledOut,
                                }"
                                @pointerdown="
                                    onPointerDownItem($event, 'tunnel', item)
                                "
                            >
                                {{ item.label }}
                            </div>
                        </transition-group>
                        <button
                            v-if="showOutputsList"
                            class="toolbar-btn import-btn"
                            @click="importConfig"
                        >
                            <Upload :size="16" />
                            <span>Import Config</span>
                        </button>
                    </div>
                </div>
            </Panel>

            <template #connection-line>
                <g style="opacity: 0.8; pointer-events: none">
                    <path
                        :d="dragPath()"
                        fill="none"
                        stroke="var(--accent-success)"
                        stroke-width="2"
                    />
                    <g
                        :transform="`translate(${dragWire.mouseX}, ${dragWire.mouseY})`"
                    >
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
                        <g
                            v-if="dragWire.hoveredAppId"
                            transform="translate(8, -12)"
                        >
                            <path
                                d="M -4 -4 L 4 4 M -4 4 L 4 -4"
                                stroke="var(--accent-danger)"
                                stroke-width="2"
                                stroke-linecap="round"
                            />
                        </g>
                    </g>
                </g>
            </template>
        </VueFlow>

        <div
            v-if="dragPreview.active"
            class="drag-preview-node"
            :style="{
                transform: `translate(calc(${dragPreview.x}px - 50%), calc(${dragPreview.y}px - 50%))`,
            }"
        >
            <div
                v-if="!dragPreview.pulledOut"
                class="dropdown-item"
                style="
                    margin: 0;
                    min-width: 140px;
                    box-shadow: 0 10px 15px rgba(0, 0, 0, 0.3);
                "
            >
                {{ dragPreview.data?.label }}
            </div>
            <template v-else>
                <ApplicationNode
                    v-if="dragPreview.type === 'application'"
                    id="preview"
                    type="application"
                    :position="{ x: 0, y: 0 }"
                    :dimensions="{ width: 200, height: 80 }"
                    :selected="false"
                    :dragging="true"
                    :connectable="false"
                    :resizing="false"
                    :events="{}"
                    :z-index="1"
                    :target-position="Position.Left"
                    :source-position="Position.Right"
                    :data="dragPreview.data"
                />
                <TunnelNode
                    v-if="dragPreview.type === 'tunnel'"
                    id="preview"
                    type="tunnel"
                    :position="{ x: 0, y: 0 }"
                    :dimensions="{ width: 200, height: 80 }"
                    :selected="false"
                    :dragging="true"
                    :connectable="false"
                    :resizing="false"
                    :events="{}"
                    :z-index="1"
                    :target-position="Position.Left"
                    :source-position="Position.Right"
                    :data="dragPreview.data"
                />
            </template>
        </div>

        <svg
            v-if="dragWire.active && dragWire.originalEdgeId"
            class="drag-overlay"
        >
            <g
                :transform="`translate(${viewport.x}, ${viewport.y}) scale(${viewport.zoom})`"
            >
                <path
                    :d="dragPath()"
                    fill="none"
                    stroke="var(--accent-success)"
                    stroke-width="2"
                    opacity="0.8"
                />
                <g
                    :transform="`translate(${dragWire.mouseX}, ${dragWire.mouseY})`"
                >
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
                    <g
                        v-if="dragWire.hoveredAppId"
                        transform="translate(8, -12)"
                    >
                        <path
                            d="M -4 -4 L 4 4 M -4 4 L 4 -4"
                            stroke="var(--accent-danger)"
                            stroke-width="2"
                            stroke-linecap="round"
                        />
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
.drag-preview-node {
    position: fixed;
    left: 0;
    top: 0;
    z-index: 9999;
    pointer-events: none;
    will-change: transform;
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

.toolbar-panel {
    pointer-events: all;
    z-index: 10;
}

.toolbar-controls {
    display: flex;
    gap: 0.5rem;
    background: var(--bg-app);
    padding: 0.5rem;
    border-radius: 0.75rem;
    border: 1px solid var(--border-color);
    box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1);
}

.toolbar-btn {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    background-color: var(--bg-card);
    border: 1px solid var(--border-color);
    color: var(--text-primary);
    padding: 0.5rem 0.75rem;
    border-radius: 0.5rem;
    cursor: pointer;
    font-size: 0.875rem;
    font-weight: 500;
    transition: all 0.2s;
}

.toolbar-btn:hover {
    background-color: #3f3f46;
    border-color: #52525b;
}

.trash-btn {
    color: var(--text-secondary);
}

.trash-btn:hover {
    color: var(--accent-danger);
    border-color: var(--accent-danger);
    background-color: rgba(239, 68, 68, 0.1);
}

.dropdown-wrapper {
    position: relative;
}

.dropdown-menu {
    position: absolute;
    top: calc(100% + 0.5rem);
    left: 0;
    background: var(--bg-card);
    border: 1px solid var(--border-color);
    border-radius: 0.5rem;
    min-width: 160px;
    padding: 0.5rem;
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
    box-shadow: 0 10px 15px -3px rgba(0, 0, 0, 0.1);
}

.dropdown-item {
    padding: 0.5rem 0.75rem;
    border-radius: 0.25rem;
    font-size: 0.875rem;
    color: var(--text-primary);
    cursor: grab;
    background: var(--bg-app);
    border: 1px solid transparent;
    user-select: none;
    touch-action: none;
    display: flex;
    align-items: center;
    gap: 0.5rem;
}

.dropdown-icon {
    width: 20px;
    height: 20px;
    object-fit: contain;
    border-radius: 3px;
    flex-shrink: 0;
}

.dropdown-icon-placeholder {
    width: 20px;
    height: 20px;
    background: var(--border-color);
    border-radius: 3px;
    flex-shrink: 0;
}

.import-btn {
    margin-top: 0.25rem;
    font-size: 0.75rem;
    justify-content: center;
}

.dropdown-menu {
    max-height: 300px;
    overflow-y: auto;
}

.dropdown-item:hover {
    background: #3f3f46;
    border-color: #52525b;
}

.dropdown-item:active {
    cursor: grabbing;
}

.empty-msg {
    padding: 0.5rem;
    font-size: 0.75rem;
    color: var(--text-secondary);
    text-align: center;
}

.ghost-hidden {
    opacity: 0;
}

.collapsed-hidden {
    opacity: 0;
    height: 0;
    min-height: 0;
    padding-top: 0;
    padding-bottom: 0;
    margin: 0;
    border: none;
    overflow: hidden;
    /* REMOVED: pointer-events: none; - this was forcefully dropping pointer capture! */
    transition: all 0.2s ease;
}

.list-move,
.list-enter-active,
.list-leave-active {
    transition: all 0.2s ease;
}
.list-enter-from,
.list-leave-to {
    opacity: 0;
    transform: scaleY(0.1);
    margin-top: -10px;
}
</style>
