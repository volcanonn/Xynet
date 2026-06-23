<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue';
import { Activity } from '@lucide/vue';
import { useAppState } from '../composables/useAppState';
import { generateRoutingRules } from '../composables/routeGenerator';
import { Deploy, GetBackendStatus, ListVoponoProcesses } from '../../wailsjs/go/main/App';
import { EventsOn } from '../../wailsjs/runtime/runtime';

const { appState } = useAppState();

const upload = ref(0);
const download = ref(0);
const backendRunning = ref(false);
const backendName = ref('singbox');
const voponoCount = ref(0);
const deploying = ref(false);

let cleanupNetStats: (() => void) | null = null;
let cleanupVoponoStart: (() => void) | null = null;
let cleanupVoponoEnd: (() => void) | null = null;
let statusPoll: ReturnType<typeof setInterval> | null = null;

const formatSpeed = (bytesPerSec: number): string => {
    if (bytesPerSec >= 1_073_741_824) return (bytesPerSec / 1_073_741_824).toFixed(1) + ' GB/s';
    if (bytesPerSec >= 1_048_576) return (bytesPerSec / 1_048_576).toFixed(1) + ' MB/s';
    if (bytesPerSec >= 1024) return (bytesPerSec / 1024).toFixed(1) + ' KB/s';
    return bytesPerSec.toFixed(0) + ' B/s';
};

const backendDisplayName = (name: string): string => {
    if (name === 'dae') return 'dae';
    return 'sing-box';
};

const checkStatus = async () => {
    try {
        const status = await GetBackendStatus();
        backendRunning.value = status.running;
        backendName.value = status.backend;
    } catch {
        backendRunning.value = false;
    }
};

const refreshVoponoCount = async () => {
    try {
        const procs = await ListVoponoProcesses();
        voponoCount.value = procs?.length || 0;
    } catch {
        voponoCount.value = 0;
    }
};

onMounted(() => {
    cleanupNetStats = EventsOn('net-stats', (stats: any) => {
        upload.value = stats.upload;
        download.value = stats.download;
    });
    cleanupVoponoStart = EventsOn('vopono-process-started', () => refreshVoponoCount());
    cleanupVoponoEnd = EventsOn('vopono-process-ended', () => refreshVoponoCount());

    checkStatus();
    refreshVoponoCount();
    statusPoll = setInterval(() => {
        checkStatus();
        refreshVoponoCount();
    }, 10000);
});

onUnmounted(() => {
    cleanupNetStats?.();
    cleanupVoponoStart?.();
    cleanupVoponoEnd?.();
    if (statusPoll) clearInterval(statusPoll);
});

const deployConfig = async () => {
    if (!appState.value) return;
    deploying.value = true;
    try {
        const rules = generateRoutingRules(appState.value);
        await Deploy(rules);
        await checkStatus();
    } catch (e) {
        console.error("Failed to deploy:", e);
        alert(`Failed to deploy: ${e}`);
    } finally {
        deploying.value = false;
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
      <button class="deploy-btn" @click="deployConfig" :disabled="deploying">
        <Activity class="icon" :size="16" />
        {{ deploying ? 'Deploying...' : 'Deploy' }}
      </button>
    </div>

    <div class="status-indicators">
      <div class="status-pill" :class="{ active: backendRunning }">
        <div class="status-dot"></div>
        <span>{{ backendDisplayName(backendName) }}: {{ backendRunning ? 'Active' : 'Offline' }}</span>
      </div>
      <div v-if="voponoCount > 0" class="status-pill active">
        <div class="status-dot"></div>
        <span>Strict: {{ voponoCount }}</span>
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

.deploy-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.status-indicators {
  display: flex;
  gap: 0.75rem;
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
