<script setup lang="ts">
import { LayoutDashboard, Network, Server, Shield, Activity } from '@lucide/vue';
import { useTelemetry } from '../../composables/useTelemetry';

const { upload, download, backendRunning, backendStatus, backendName, voponoCount, deployedAppCount, deployedTunnelCount } = useTelemetry();

const formatSpeed = (bytesPerSec: number): string => {
    if (bytesPerSec >= 1_073_741_824) return (bytesPerSec / 1_073_741_824).toFixed(1) + ' GB/s';
    if (bytesPerSec >= 1_048_576) return (bytesPerSec / 1_048_576).toFixed(1) + ' MB/s';
    if (bytesPerSec >= 1024) return (bytesPerSec / 1024).toFixed(1) + ' KB/s';
    return bytesPerSec.toFixed(0) + ' B/s';
};
</script>

<template>
  <div class="view-container">
    <div class="header-section">
      <LayoutDashboard :size="32" class="icon" />
      <h2>Dashboard</h2>
    </div>

    <div class="grid-container">
      <div class="card status-card">
        <div class="card-header">
          <Activity :size="20" class="card-icon" />
          <h3>System Status</h3>
        </div>
        <div class="card-body status-body">
          <div class="status-indicator" :class="{ 
            active: backendStatus === 'active', 
            suspended: backendStatus === 'suspended' 
          }">
            <div class="dot"></div>
            <span class="status-text">
              {{ backendStatus === 'active' ? 'Active' : (backendStatus === 'suspended' ? 'Suspended' : 'Offline') }}
            </span>
          </div>
          <span class="backend-label">Engine: <strong>{{ backendName === 'dae' ? 'dae (eBPF)' : 'sing-box (TUN)' }}</strong></span>
        </div>
      </div>

      <div class="card bandwidth-card">
        <div class="card-header">
          <Network :size="20" class="card-icon" />
          <h3>Bandwidth</h3>
        </div>
        <div class="card-body bandwidth-body">
          <div class="speed-row">
            <span class="speed-label">Download</span>
            <span class="speed-value text-success">{{ formatSpeed(download) }}</span>
          </div>
          <div class="speed-row">
            <span class="speed-label">Upload</span>
            <span class="speed-value text-primary">{{ formatSpeed(upload) }}</span>
          </div>
        </div>
      </div>

      <div class="card routing-card">
        <div class="card-header">
          <Server :size="20" class="card-icon" />
          <h3>Active Routing</h3>
        </div>
        <div class="card-body">
          <div class="stat-group">
            <span class="stat-number">{{ deployedAppCount }}</span>
            <span class="stat-desc">Apps Routed</span>
          </div>
          <div class="stat-group">
            <span class="stat-number">{{ deployedTunnelCount }}</span>
            <span class="stat-desc">Active Tunnels</span>
          </div>
        </div>
      </div>

      <div class="card strict-card">
        <div class="card-header">
          <Shield :size="20" class="card-icon text-purple" />
          <h3>Strict Isolation</h3>
        </div>
        <div class="card-body">
          <div class="stat-group">
            <span class="stat-number text-purple">{{ voponoCount }}</span>
            <span class="stat-desc">Isolated Namespaces</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.view-container {
  display: flex;
  flex-direction: column;
  height: 100%;
  padding: 2rem;
  box-sizing: border-box;
}

.header-section {
  display: flex;
  align-items: center;
  gap: 1rem;
  margin-bottom: 2rem;
}

h2 {
  color: var(--text-primary);
  margin: 0;
  font-size: 1.5rem;
}

.icon {
  color: var(--text-primary);
}

.grid-container {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: 1.5rem;
}

.card {
  background-color: var(--bg-card);
  border: 1px solid var(--border-color);
  border-radius: 0.75rem;
  padding: 1.25rem;
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.card-header {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.card-header h3 {
  margin: 0;
  font-size: 1rem;
  font-weight: 600;
  color: var(--text-primary);
}

.card-icon {
  color: var(--text-secondary);
}

.card-body {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.status-body {
  align-items: flex-start;
}

.status-indicator {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  background: rgba(255, 255, 255, 0.05);
  padding: 0.5rem 1rem;
  border-radius: 9999px;
  border: 1px solid var(--border-color);
}

.status-indicator.active {
  background: rgba(34, 197, 94, 0.1);
  border-color: rgba(34, 197, 94, 0.2);
}

.dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background-color: var(--text-secondary);
}

.status-indicator.active .dot {
  background-color: var(--accent-success);
  box-shadow: 0 0 8px var(--accent-success);
}

.status-indicator.suspended {
  background: rgba(234, 179, 8, 0.1);
  border-color: rgba(234, 179, 8, 0.2);
}

.status-indicator.suspended .dot {
  background-color: #eab308;
  box-shadow: 0 0 8px #eab308;
}

.status-text {
  font-weight: 600;
  color: var(--text-secondary);
  text-transform: capitalize;
}

.status-indicator.active .status-text {
  color: var(--accent-success);
}

.status-indicator.suspended .status-text {
  color: #eab308;
}

.backend-label {
  font-size: 0.875rem;
  color: var(--text-secondary);
}

.bandwidth-body {
  gap: 0.5rem;
}

.speed-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0.5rem;
  background: rgba(0, 0, 0, 0.2);
  border-radius: 0.5rem;
}

.speed-label {
  font-size: 0.875rem;
  font-weight: 500;
  color: var(--text-secondary);
}

.speed-value {
  font-size: 1.125rem;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
}

.text-success { color: var(--accent-success); }
.text-primary { color: var(--accent-primary); }
.text-purple { color: #a855f7 !important; }

.routing-card .card-body,
.strict-card .card-body {
  flex-direction: row;
  gap: 1.5rem;
}

.stat-group {
  display: flex;
  flex-direction: column;
}

.stat-number {
  font-size: 2rem;
  font-weight: 700;
  color: var(--text-primary);
  line-height: 1.2;
}

.stat-desc {
  font-size: 0.8125rem;
  color: var(--text-secondary);
  font-weight: 500;
}
</style>
