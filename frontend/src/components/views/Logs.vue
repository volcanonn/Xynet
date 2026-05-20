<script setup lang="ts">
import { FileText } from '@lucide/vue';
</script>

<template>
  <div class="view-container">
    <FileText :size="48" class="icon" />
    <h2>Logs</h2>
    <p>Real-time system and proxy logs will be displayed here.</p>
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
