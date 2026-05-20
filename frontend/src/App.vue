<script lang="ts" setup>
import { ref } from 'vue';
import Sidebar from './components/Sidebar.vue';
import Header from './components/Header.vue';
import Canvas from './components/Canvas.vue';
import Dashboard from './components/views/Dashboard.vue';
import Proxies from './components/views/Proxies.vue';
import Logs from './components/views/Logs.vue';
import Settings from './components/views/Settings.vue';

const activeTab = ref('routing');
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
</style>
