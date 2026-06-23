<script setup lang="ts">
import { ref, onMounted, onUnmounted, nextTick } from 'vue';
import { Terminal, Trash2 } from '@lucide/vue';
import { EventsOn } from '../../../wailsjs/runtime/runtime';

interface LogEntry {
  timestamp: string;
  source: string;
  message: string;
}

const logs = ref<LogEntry[]>([]);
const terminalRef = ref<HTMLElement | null>(null);
let cleanupLogs: (() => void) | null = null;
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

const handleScroll = () => {
  if (!terminalRef.value) return;
  const { scrollTop, scrollHeight, clientHeight } = terminalRef.value;
  // If user scrolls up significantly, disable autoscroll
  autoScroll.value = scrollHeight - scrollTop - clientHeight < 50;
};

onMounted(() => {
  cleanupLogs = EventsOn('app-log', (entry: LogEntry) => {
    logs.value.push(entry);
    if (logs.value.length > 5000) {
      logs.value.shift(); // Keep last 5000 lines
    }
    
    if (autoScroll.value) {
      nextTick(() => {
        if (terminalRef.value) {
          terminalRef.value.scrollTop = terminalRef.value.scrollHeight;
        }
      });
    }
  });
});

onUnmounted(() => {
  cleanupLogs?.();
});
</script>

<template>
  <div class="view-container">
    <div class="header-section">
      <div class="title-group">
        <Terminal :size="24" class="icon" />
        <h2>System Logs</h2>
      </div>
      <button class="clear-btn" @click="clearLogs">
        <Trash2 :size="16" />
        Clear
      </button>
    </div>

    <div class="terminal-container" ref="terminalRef" @scroll="handleScroll">
      <div v-if="logs.length === 0" class="empty-state">
        Waiting for routing engines to start...
      </div>
      <div v-for="(log, idx) in logs" :key="idx" class="log-line">
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

.clear-btn {
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

.clear-btn:hover {
  background-color: rgba(239, 68, 68, 0.1);
  color: var(--accent-danger);
  border-color: var(--accent-danger);
}

.terminal-container {
  flex: 1;
  background-color: #09090b; /* Very dark background for terminal */
  border: 1px solid var(--border-color);
  border-radius: 0.5rem;
  padding: 1rem;
  overflow-y: auto;
  font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
  font-size: 0.8125rem;
  line-height: 1.5;
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
}
</style>
