<script setup lang="ts">
import { Activity, Server } from '@lucide/vue';
</script>

<template>
  <header class="header">
    <div class="telemetry">
      <div class="stat">
        <span class="stat-label">Upload</span>
        <span class="stat-value">30.8 MB/s</span>
        <!-- Placeholder for sparkline graph -->
        <div class="sparkline-placeholder"></div>
      </div>
      <div class="stat">
        <span class="stat-label">Download</span>
        <span class="stat-value">24.8 MB/s</span>
        <!-- Placeholder for sparkline graph -->
        <div class="sparkline-placeholder"></div>
      </div>
    </div>

    <div class="status-indicators">
      <div class="status-pill active">
        <div class="status-dot"></div>
        <span>Sing-box: Active</span>
      </div>
      <div class="status-pill active">
        <div class="status-dot"></div>
        <span>Vopono: Ready</span>
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

.sparkline-placeholder {
  height: 16px;
  width: 60px;
  margin-top: 2px;
  border-bottom: 1px dashed var(--border-color);
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
