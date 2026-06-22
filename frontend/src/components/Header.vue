<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue';
import { Activity } from '@lucide/vue';
import { useAppState } from '../composables/useAppState';
import { generateSingboxConfig } from '../composables/singboxGenerator';
import { WriteSingboxConfig, RestartSingbox } from '../../wailsjs/go/main/App';
import { EventsOn } from '../../wailsjs/runtime/runtime';

const { appState } = useAppState();

const upload = ref(0);
const download = ref(0);
const singboxRunning = ref(false);
const voponoCount = ref(0);

let cleanupNetStats: (() => void) | null = null;
let cleanupServiceStatus: (() => void) | null = null;

const formatSpeed = (bytesPerSec: number): string => {
    if (bytesPerSec >= 1_073_741_824) return (bytesPerSec / 1_073_741_824).toFixed(1) + ' GB/s';
    if (bytesPerSec >= 1_048_576) return (bytesPerSec / 1_048_576).toFixed(1) + ' MB/s';
    if (bytesPerSec >= 1024) return (bytesPerSec / 1024).toFixed(1) + ' KB/s';
    return bytesPerSec.toFixed(0) + ' B/s';
};

onMounted(() => {
    cleanupNetStats = EventsOn('net-stats', (stats: any) => {
        upload.value = stats.upload;
        download.value = stats.download;
    });
    cleanupServiceStatus = EventsOn('service-status', (status: any) => {
        singboxRunning.value = status.singboxRunning;
        voponoCount.value = status.voponoCount;
    });
});

onUnmounted(() => {
    cleanupNetStats?.();
    cleanupServiceStatus?.();
});

const deployConfig = async () => {
    if (!appState.value) return;
    try {
        const configJson = generateSingboxConfig(appState.value);
        console.log("Deploying config:", configJson);
        await WriteSingboxConfig(configJson);
        await RestartSingbox();
        alert("Deployed & restarted Sing-box successfully.");
    } catch (e) {
        console.error("Failed to deploy:", e);
        alert(`Failed to deploy: ${e}`);
    }
};
</script>

<template>
  <header class="header">
    <div class="telemetry">
      <div class="stat">
        <span class="stat-label">Upload</span>
        <span class="stat-value">{{ formatSpeed(upload) }}</span>
      </div>
      <div class="stat">
        <span class="stat-label">Download</span>
        <span class="stat-value">{{ formatSpeed(download) }}</span>
      </div>
    </div>

    <div class="actions">
      <button class="deploy-btn" @click="deployConfig">
        <Activity class="icon" :size="16" />
        Deploy Routes
      </button>
    </div>

    <div class="status-indicators">
      <div class="status-pill" :class="{ active: singboxRunning }">
        <div class="status-dot"></div>
        <span>Sing-box: {{ singboxRunning ? 'Active' : 'Inactive' }}</span>
      </div>
      <div class="status-pill" :class="{ active: voponoCount > 0 }">
        <div class="status-dot"></div>
        <span>Vopono: {{ voponoCount > 0 ? voponoCount + ' Running' : 'Ready' }}</span>
      </div>
    </div>
  </header>
</template>

<style scoped>
.header {
  height: 64px;
  background-color: var(--bg-header);
  border-bottom: 1px solid var(--border-color);
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 1.5rem;
  flex-shrink: 0;
}

.telemetry {
  display: flex;
  gap: 2rem;
}

.stat {
  display: flex;
  flex-direction: column;
}

.stat-label {
  font-size: 0.75rem;
  color: var(--text-secondary);
}

.stat-value {
  font-size: 0.875rem;
  font-weight: 600;
  color: var(--text-primary);
}

.actions {
  display: flex;
  align-items: center;
  margin-right: 1.5rem;
}

.deploy-btn {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  background-color: var(--accent-primary);
  color: white;
  border: none;
  border-radius: 0.5rem;
  padding: 0.5rem 1rem;
  font-size: 0.875rem;
  font-weight: 600;
  cursor: pointer;
  transition: filter 0.2s;
}

.deploy-btn:hover {
  filter: brightness(1.2);
}

.status-indicators {
  display: flex;
  gap: 1rem;
}

.status-pill {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.375rem 0.75rem;
  background-color: var(--bg-card);
  border: 1px solid var(--border-color);
  border-radius: 9999px;
  font-size: 0.75rem;
  font-weight: 500;
  color: var(--text-secondary);
}

.status-pill.active {
  color: var(--text-primary);
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background-color: var(--text-secondary);
}

.status-pill.active .status-dot {
  background-color: var(--accent-success);
  box-shadow: 0 0 8px rgba(34, 197, 94, 0.4);
}
</style>
