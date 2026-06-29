<script setup lang="ts">
import { ref, watch, nextTick } from 'vue';
import { Terminal, Trash2, Download } from '@lucide/vue';
import { useTelemetry } from '../../composables/useTelemetry';

const { systemLogs: logs } = useTelemetry();
const terminalRef = ref<HTMLElement | null>(null);
const autoScroll = ref(true);

const getSourceColor = (source: string) => {
  if (source.startsWith('sing-box')) return '#3b82f6';
  if (source.startsWith('dae')) return '#22c55e';
  if (source.startsWith('vopono')) return '#a855f7';
  return '#a1a1aa';
};

const clearLogs = () => {
  logs.value = [];
};

const downloadLogs = () => {
  if (logs.value.length === 0) return;
  const content = logs.value.map(l => `[${l.timestamp}] [${l.source}] ${l.message}`).join('\n');
  const blob = new Blob([content], { type: 'text/plain' });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = `xynet_logs_${new Date().toISOString().replace(/[:.]/g, '-')}.txt`;
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  URL.revokeObjectURL(url);
};

const handleScroll = () => {
  if (!terminalRef.value) return;
  const { scrollTop, scrollHeight, clientHeight } = terminalRef.value;
  autoScroll.value = scrollHeight - scrollTop - clientHeight < 50;
};

watch(logs, () => {
  if (autoScroll.value) {
    nextTick(() => {
      if (terminalRef.value) {
        terminalRef.value.scrollTop = terminalRef.value.scrollHeight;
      }
    });
  }
}, { deep: true });
</script>

<template>
  <div class="view-container">
    <div class="header-section">
      <div class="title-group">
        <Terminal :size="24" class="icon" />
        <h2>System Logs</h2>
      </div>
      <div class="actions-group">
        <button class="btn download-btn" @click="downloadLogs">
          <Download :size="16" />
          Download
        </button>
        <button class="btn clear-btn" @click="clearLogs">
          <Trash2 :size="16" />
          Clear
        </button>
      </div>
    </div>

    <div class="terminal-container" ref="terminalRef" @scroll="handleScroll">
      <div v-if="logs.length === 0" class="empty-state">
        Waiting for routing engines to start...
      </div>
      <div v-for="log in logs" :key="log.id" class="log-line">
        <span class="timestamp">[{{ log.timestamp }}]</span>
        <span class="source" :style="{ color: getSourceColor(log.source) }">[{{ log.source }}]</span>
        <span class="message">{{ log.message }}</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.view-container {
  display: flex;
  flex-direction: column;
  height: 100%;
  padding: 1.5rem;
  box-sizing: border-box;
}

.header-section {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1rem;
}

.title-group {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

h2 {
  color: var(--text-primary);
  margin: 0;
  font-size: 1.25rem;
}

.icon {
  color: var(--text-secondary);
}

.actions-group {
  display: flex;
  gap: 0.5rem;
}

.btn {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  background-color: var(--bg-card);
  border: 1px solid var(--border-color);
  color: var(--text-secondary);
  padding: 0.5rem 1rem;
  border-radius: 0.5rem;
  cursor: pointer;
  transition: all 0.2s;
  font-size: 0.875rem;
}

.btn:hover {
  background-color: rgba(255, 255, 255, 0.05);
  color: var(--text-primary);
}

.clear-btn:hover {
  background-color: rgba(239, 68, 68, 0.1);
  color: var(--accent-danger);
  border-color: var(--accent-danger);
}

.download-btn:hover {
  background-color: rgba(59, 130, 246, 0.1);
  color: var(--accent-primary);
  border-color: var(--accent-primary);
}

.terminal-container {
  flex: 1;
  background-color: #09090b;
  border: 1px solid var(--border-color);
  border-radius: 0.5rem;
  padding: 1rem;
  overflow-y: auto;
  font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
  font-size: 0.8125rem;
  line-height: 1.5;
  user-select: text;
  -webkit-user-select: text;
}

.empty-state {
  color: var(--text-secondary);
  font-style: italic;
  opacity: 0.7;
}

.log-line {
  display: flex;
  gap: 0.5rem;
  word-break: break-all;
  white-space: pre-wrap;
}

.log-line:hover {
  background-color: rgba(255, 255, 255, 0.05);
}

.timestamp {
  color: #71717a;
  flex-shrink: 0;
}

.source {
  font-weight: bold;
  flex-shrink: 0;
}

.message {
  color: #e4e4e7;
  user-select: text;
  -webkit-user-select: text;
}
</style>
