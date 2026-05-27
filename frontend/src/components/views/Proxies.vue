<script setup lang="ts">
import { ref } from 'vue';
import { Globe } from '@lucide/vue';
import { ImportWireguardConfig } from '../../../wailsjs/go/main/App';

const configContent = ref('');

const importConfig = async () => {
  try {
    const content = await ImportWireguardConfig();
    if (content) configContent.value = content;
  } catch (e) {
    console.error("Failed to import config", e);
  }
};
</script>

<template>
  <div class="view-container">
    <Globe :size="48" class="icon" />
    <h2>Proxies</h2>
    <p>Proxy group and node management will be displayed here.</p>
    <button @click="importConfig" class="import-btn">Import WireGuard Config</button>
    <pre v-if="configContent" class="config-display">{{ configContent }}</pre>
  </div>
</template>

<style scoped>
.view-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: var(--text-secondary);
  gap: 1rem;
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
}

.import-btn:hover {
  opacity: 0.9;
}

.config-display {
  background-color: #1e293b;
  padding: 1rem;
  border-radius: 4px;
  max-width: 80%;
  overflow-x: auto;
  text-align: left;
}
</style>
