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
  <div class="wg-tabs-container" v-if="configs.length > 0">
    <div 
      v-for="(conf, index) in configs" 
      :key="index"
      class="wg-tab"
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
  display: flex;
  flex-wrap: wrap;
  gap: 1rem;
  margin: 0 auto 2rem auto;
  width: 100%;
  max-width: 800px;
  justify-content: center;
  align-items: center;
  position: relative;
  z-index: 10;
}

.wg-tab {
  width: 240px;
  height: 60px;
  background-color: var(--bg-card);
  border: 1px solid var(--border-color, #334155);
  border-radius: 8px;
  box-shadow: 2px 2px 8px rgba(0,0,0,0.2);
  display: flex;
  align-items: center;
  transition: transform 0.2s ease, box-shadow 0.2s ease;
  cursor: pointer;
  position: relative;
}

.wg-tab:hover {
  transform: translateY(-2px);
  box-shadow: 4px 4px 12px rgba(0,0,0,0.3);
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
