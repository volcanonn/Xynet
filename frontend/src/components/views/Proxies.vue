<script setup lang="ts">
import { ref } from 'vue';
import { Globe, X } from '@lucide/vue';
import { ImportWireguardConfig } from '../../../wailsjs/go/main/App';

interface ConfigItem {
  id: number;
  name: string;
  content: string;
}

const configs = ref<ConfigItem[]>([]);
const activeTabId = ref<number | null>(null);
let nextId = 1;

const importConfig = async () => {
  try {
    const content = await ImportWireguardConfig();
    if (content) {
      const newConfig = {
        id: nextId++,
        name: `Config ${nextId - 1}`,
        content: content
      };
      configs.value.push(newConfig);
      activeTabId.value = newConfig.id;
    }
  } catch (e) {
    console.error("Failed to import config", e);
  }
};

const closeTab = (id: number) => {
  const index = configs.value.findIndex(c => c.id === id);
  if (index !== -1) {
    configs.value.splice(index, 1);
    if (activeTabId.value === id) {
      if (configs.value.length > 0) {
        activeTabId.value = configs.value[Math.max(0, index - 1)].id;
      } else {
        activeTabId.value = null;
      }
    }
  }
};
</script>

<template>
  <div class="view-container">
    <div class="header-section">
      <Globe :size="48" class="icon" />
      <h2>Proxies</h2>
      <p>Proxy group and node management will be displayed here.</p>
      <button @click="importConfig" class="import-btn">Import WireGuard Config</button>
    </div>

    <div v-if="configs.length > 0" class="tabs-container">
      <transition-group name="tab-list" tag="div" class="tabs-header">
        <div 
          v-for="config in configs" 
          :key="config.id"
          class="tab"
          :class="{ active: activeTabId === config.id }"
          @click="activeTabId = config.id"
        >
          <span>{{ config.name }}</span>
          <button class="close-btn" @click.stop="closeTab(config.id)">
            <X :size="14" />
          </button>
        </div>
      </transition-group>

      <div class="tab-content-wrapper">
        <transition name="fade" mode="out-in">
          <pre 
            v-if="activeTabId" 
            :key="activeTabId" 
            class="config-display"
          >{{ configs.find(c => c.id === activeTabId)?.content }}</pre>
        </transition>
      </div>
    </div>
  </div>
</template>

<style scoped>
.view-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  height: 100%;
  color: var(--text-secondary);
  padding: 2rem;
  box-sizing: border-box;
  overflow: hidden;
}

.header-section {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 1rem;
  margin-bottom: 2rem;
}

.icon {
  opacity: 0.5;
}

h2 {
  color: var(--text-primary);
  margin: 0;
}

.import-btn {
  background-color: #3b82f6;
  color: white;
  border: none;
  padding: 0.5rem 1rem;
  border-radius: 4px;
  cursor: pointer;
  font-weight: 500;
  transition: opacity 0.2s;
}

.import-btn:hover {
  opacity: 0.9;
}

.tabs-container {
  width: 100%;
  max-width: 800px;
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
}

.tabs-header {
  display: flex;
  gap: 0.5rem;
  margin-bottom: 1rem;
  overflow-x: auto;
  padding-bottom: 0.5rem;
}

.tab {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 1rem;
  background-color: var(--bg-card, #1e293b);
  border-radius: 4px;
  cursor: pointer;
  transition: all 0.3s ease;
  border: 1px solid transparent;
  user-select: none;
}

.tab:hover {
  background-color: var(--bg-card-hover, #334155);
}

.tab.active {
  background-color: #3b82f6;
  color: white;
  border-color: #2563eb;
}

.close-btn {
  background: none;
  border: none;
  color: inherit;
  cursor: pointer;
  padding: 0;
  display: flex;
  align-items: center;
  opacity: 0.7;
  transition: opacity 0.2s;
}

.close-btn:hover {
  opacity: 1;
}

.tab-content-wrapper {
  flex: 1;
  position: relative;
  overflow: auto;
  background-color: var(--bg-card, #1e293b);
  border-radius: 4px;
}

.config-display {
  margin: 0;
  padding: 1rem;
  width: 100%;
  box-sizing: border-box;
  text-align: left;
}

/* Animations */
.tab-list-enter-active,
.tab-list-leave-active {
  transition: all 0.4s ease;
}
.tab-list-enter-from,
.tab-list-leave-to {
  opacity: 0;
  transform: translateY(10px);
}
.tab-list-leave-active {
  position: absolute;
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.3s ease, transform 0.3s ease;
}
.fade-enter-from {
  opacity: 0;
  transform: translateY(10px);
}
.fade-leave-to {
  opacity: 0;
  transform: translateY(-10px);
}
</style>
