<script setup lang="ts">
import { LayoutDashboard, Network, Globe, FileText, Settings } from '@lucide/vue';
import { ref } from 'vue';

const menuItems = [
  { id: 'dashboard', label: 'Dashboard', icon: LayoutDashboard },
  { id: 'routing', label: 'Routing Canvas', icon: Network },
  { id: 'proxies', label: 'Proxies', icon: Globe },
  { id: 'logs', label: 'Logs', icon: FileText },
];

defineProps<{
  modelValue: string;
}>();

const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void;
}>();
</script>

<template>
  <aside class="sidebar">
    <div class="logo">
      <Network class="logo-icon" />
      <span class="logo-text">Xynet</span>
    </div>

    <nav class="nav-menu">
      <button
        v-for="item in menuItems"
        :key="item.id"
        class="nav-item"
        :class="{ active: modelValue === item.id }"
        @click="emit('update:modelValue', item.id)"
      >
        <component :is="item.icon" class="nav-icon" :size="18" />
        <span>{{ item.label }}</span>
      </button>
    </nav>

    <div class="spacer"></div>

    <div class="nav-menu">
      <button class="nav-item" :class="{ active: modelValue === 'settings' }" @click="emit('update:modelValue', 'settings')">
        <Settings class="nav-icon" :size="18" />
        <span>Settings</span>
      </button>
    </div>
  </aside>
</template>

<style scoped>
.sidebar {
  width: 240px;
  background-color: var(--bg-sidebar);
  border-right: 1px solid var(--border-color);
  display: flex;
  flex-direction: column;
  padding: 1rem 0;
  flex-shrink: 0;
}

.logo {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0 1.5rem 1.5rem;
  margin-bottom: 0.5rem;
}

.logo-icon {
  color: var(--text-primary);
}

.logo-text {
  font-size: 1.25rem;
  font-weight: 600;
  letter-spacing: 0.025em;
}

.nav-menu {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  padding: 0 0.75rem;
}

.nav-item {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  width: 100%;
  padding: 0.5rem 0.75rem;
  border: none;
  background: transparent;
  color: var(--text-secondary);
  border-radius: 0.375rem;
  cursor: pointer;
  font-family: inherit;
  font-size: 0.875rem;
  font-weight: 500;
  transition: all 0.2s ease;
  text-align: left;
}

.nav-item:hover {
  background-color: var(--bg-card);
  color: var(--text-primary);
}

.nav-item.active {
  background-color: var(--bg-card-hover);
  color: var(--text-primary);
}

.nav-icon {
  opacity: 0.7;
}

.nav-item:hover .nav-icon,
.nav-item.active .nav-icon {
  opacity: 1;
}

.spacer {
  flex-grow: 1;
}
</style>
