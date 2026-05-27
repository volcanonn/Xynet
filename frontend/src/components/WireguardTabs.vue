<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { Network } from '@lucide/vue';

interface WgConfig {
  name: string;
  isAirvpn: boolean;
  usageData?: string;
}

const configs = ref<WgConfig[]>([]);

onMounted(async () => {
  try {
    const w = window as any;
    if (w.go && w.go.main && w.go.main.App && w.go.main.App.GetWireguardConfigs) {
      configs.value = await w.go.main.App.GetWireguardConfigs();
    } else {
      // Fallback dummy data if wails backend is not available
      configs.value = [
        { name: 'wg0', isAirvpn: false },
        { name: 'airvpn_nl', isAirvpn: true, usageData: '12.4 GB / 50 GB' },
        { name: 'airvpn_us', isAirvpn: true, usageData: '3.1 GB / 50 GB' },
      ];
    }
  } catch (e) {
    console.error("Failed to load wireguard configs", e);
  }
});
</script>

<template>
  <div class="wg-tabs-container">
    <div 
      v-for="(conf, index) in configs" 
      :key="index"
      class="wg-tab"
      :style="{ top: `${20 + index * 70}px` }"
    >
      <div class="wg-tab-content">
        <Network class="wg-icon" :size="20" />
        <div class="wg-info">
          <div class="wg-name">{{ conf.name }}</div>
          <div v-if="conf.isAirvpn" class="wg-usage">{{ conf.usageData }}</div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.wg-tabs-container {
  position: absolute;
  left: 0;
  top: 0;
  bottom: 0;
  width: 0;
  z-index: 1000;
  pointer-events: none;
}

.wg-tab {
  position: absolute;
  left: -200px;
  width: 240px;
  height: 60px;
  background-color: var(--bg-card);
  border: 1px solid var(--border-color);
  border-left: none;
  border-radius: 0 8px 8px 0;
  box-shadow: 2px 2px 8px rgba(0,0,0,0.2);
  display: flex;
  align-items: center;
  transition: transform 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  pointer-events: auto;
  cursor: pointer;
}

.wg-tab:hover {
  transform: translateX(200px);
}

.wg-tab-content {
  display: flex;
  align-items: center;
  padding: 0 15px;
  width: 100%;
  gap: 12px;
}

.wg-icon {
  color: var(--text-primary);
  flex-shrink: 0;
}

.wg-info {
  display: flex;
  flex-direction: column;
  justify-content: center;
  flex: 1;
  min-width: 0;
}

.wg-name {
  color: var(--text-primary);
  font-weight: 600;
  font-size: 0.9rem;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.wg-usage {
  color: var(--text-secondary);
  font-size: 0.75rem;
  margin-top: 2px;
}
</style>
