<script lang="ts" setup>
import { ref } from 'vue';
import Sidebar from './components/Sidebar.vue';
import Header from './components/Header.vue';
import Canvas from './components/Canvas.vue';
import Dashboard from './components/views/Dashboard.vue';
import Proxies from './components/views/Proxies.vue';
import Logs from './components/views/Logs.vue';
import Settings from './components/views/Settings.vue';
import { useToast } from './composables/useToast';
import { AlertCircle, CheckCircle, Info } from '@lucide/vue';

import { useAppState } from './composables/useAppState';
import { onMounted } from 'vue';

const activeTab = ref('routing');
const { toasts } = useToast();
const { appState } = useAppState();

onMounted(() => {
  if (appState.value?.settings?.theme === 'light') {
    document.body.className = 'light-theme';
  }
});
</script>

<template>
  <div class="app-layout">
    <Sidebar v-model="activeTab" />
    <div class="main-content">
      <Header />
      <main class="canvas-area">
        <Canvas v-if="activeTab === 'routing'" />
        <Dashboard v-if="activeTab === 'dashboard'" />
        <Proxies v-if="activeTab === 'proxies'" />
        <Logs v-if="activeTab === 'logs'" />
        <Settings v-if="activeTab === 'settings'" />
      </main>
    </div>

    <!-- Global Toast Container -->
    <div class="toast-container">
      <transition-group name="toast">
        <div v-for="toast in toasts" :key="toast.id" class="toast-item" :class="toast.type">
          <CheckCircle v-if="toast.type === 'success'" :size="18" />
          <AlertCircle v-else-if="toast.type === 'error'" :size="18" />
          <Info v-else :size="18" />
          <span>{{ toast.message }}</span>
        </div>
      </transition-group>
    </div>
  </div>
</template>

<style scoped>
.app-layout {
  display: flex;
  width: 100vw;
  height: 100vh;
  overflow: hidden;
}

.main-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0; /* Important for flex children to allow shrinking */
}

.canvas-area {
  flex: 1;
  background-color: var(--bg-app);
  position: relative;
  overflow: hidden;
}

.toast-container {
  position: fixed;
  bottom: 1.5rem;
  right: 1.5rem;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  z-index: 9999;
  pointer-events: none;
}

.toast-item {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.75rem 1rem;
  border-radius: 0.5rem;
  font-size: 0.875rem;
  font-weight: 500;
  color: #fff;
  background-color: var(--bg-card);
  box-shadow: 0 10px 15px -3px rgba(0, 0, 0, 0.3), 0 4px 6px -4px rgba(0, 0, 0, 0.3);
  border-left: 4px solid transparent;
  pointer-events: auto;
}

.toast-item.error { border-left-color: var(--accent-danger); }
.toast-item.success { border-left-color: var(--accent-success); }
.toast-item.info { border-left-color: var(--accent-primary); }

.toast-enter-active,
.toast-leave-active {
  transition: all 0.3s ease;
}
.toast-enter-from {
  opacity: 0;
  transform: translateX(100%);
}
.toast-leave-to {
  opacity: 0;
  transform: scale(0.9);
}
</style>
