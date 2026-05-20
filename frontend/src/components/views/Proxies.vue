<script setup lang="ts">
import { Globe } from '@lucide/vue';
</script>

<template>
  <div class="view-container">
    <Globe :size="48" class="icon" />
    <h2>Proxies</h2>
    <p>Proxy group and node management will be displayed here.</p>
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
</style>
