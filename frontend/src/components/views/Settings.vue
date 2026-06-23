<script setup lang="ts">
import { computed, ref, onMounted } from 'vue';
import { useAppState } from '../../composables/useAppState';
import { GetInterfaces, InstallDae } from '../../../wailsjs/go/main/App';
import { useToast } from '../../composables/useToast';

const { appState, saveState } = useAppState();
const toast = useToast();
const isInstalling = ref(false);

const installDae = async () => {
    isInstalling.value = true;
    toast.info("Downloading dae from GitHub...");
    try {
        await InstallDae();
        toast.success("Successfully installed dae!");
    } catch (e) {
        toast.error(`Install failed: ${e}`);
    } finally {
        isInstalling.value = false;
    }
};

const backend = computed({
    get: () => appState.value?.settings?.backend || 'singbox',
    set: (val: string) => {
        if (!appState.value) return;
        if (!appState.value.settings) appState.value.settings = {} as any;
        (appState.value.settings as any).backend = val;
        saveState();
    },
});

const theme = computed({
    get: () => appState.value?.settings?.theme || 'dark',
    set: (val: string) => {
        if (!appState.value) return;
        if (!appState.value.settings) appState.value.settings = {} as any;
        (appState.value.settings as any).theme = val;
        saveState();
        document.body.className = val === 'light' ? 'light-theme' : '';
    },
});

const defaultInterface = computed({
    get: () => appState.value?.settings?.defaultInterface || 'eth0',
    set: (val: string) => {
        if (!appState.value) return;
        if (!appState.value.settings) appState.value.settings = {} as any;
        (appState.value.settings as any).defaultInterface = val;
        saveState();
    },
});

const availableInterfaces = ref<string[]>([]);

onMounted(async () => {
    try {
        const interfaces = await GetInterfaces();
        if (interfaces) availableInterfaces.value = interfaces;
    } catch (e) {
        console.error("Failed to fetch interfaces", e);
    }
});

const backends = [
    {
        value: 'singbox',
        label: 'sing-box',
        tag: 'Recommended',
        description: 'Userspace TUN proxy with native WireGuard and Hysteria2 support. Cross-platform — will support Windows and macOS in the future.',
    },
    {
        value: 'dae',
        label: 'dae',
        tag: 'High Performance',
        description: 'eBPF-based kernel-level routing. Bypassed traffic never enters userspace. Linux only, requires the dae package to be installed.',
    },
];

const themes = [
    { value: 'dark', label: 'Dark Theme' },
    { value: 'light', label: 'Light Theme' },
];
</script>

<template>
  <div class="settings-page">
    <h2>Settings</h2>

    <section class="settings-section">
      <h3>Theme</h3>
      <p class="section-desc">Choose the application's visual appearance.</p>
      
      <div class="theme-options">
        <label v-for="t in themes" :key="t.value" class="radio-label">
          <input type="radio" :value="t.value" v-model="theme" />
          <span>{{ t.label }}</span>
        </label>
      </div>
    </section>

    <section class="settings-section">
      <h3>Network Interface</h3>
      <p class="section-desc">Select your primary physical network interface (e.g., eth0, wlan0). Used for bandwidth telemetry.</p>
      
      <select v-model="defaultInterface" class="interface-select">
        <option v-for="iface in availableInterfaces" :key="iface" :value="iface">
          {{ iface }}
        </option>
      </select>
    </section>

    <section class="settings-section">
      <h3>Routing Backend</h3>
      <p class="section-desc">Choose how traffic is intercepted and routed through tunnels.</p>

      <div class="backend-options">
        <label
          v-for="b in backends"
          :key="b.value"
          class="backend-card"
          :class="{ selected: backend === b.value }"
        >
          <input
            type="radio"
            :value="b.value"
            v-model="backend"
            class="radio-input"
          />
          <div class="card-content">
            <div class="card-header">
              <span class="card-label">{{ b.label }}</span>
              <span class="card-tag" :class="b.value">{{ b.tag }}</span>
            </div>
            <p class="card-desc">{{ b.description }}</p>
            <button 
              v-if="b.value === 'dae'" 
              @click.stop="installDae" 
              class="install-btn" 
              :disabled="isInstalling"
            >
              {{ isInstalling ? 'Installing...' : 'Download dae (GitHub)' }}
            </button>
          </div>
        </label>
      </div>
    </section>
  </div>
</template>

<style scoped>
.settings-page {
  padding: 2rem;
  max-width: 640px;
}

h2 {
  color: var(--text-primary);
  font-size: 1.5rem;
  font-weight: 600;
  margin: 0 0 2rem;
}

.settings-section {
  margin-bottom: 2rem;
}

h3 {
  color: var(--text-primary);
  font-size: 1rem;
  font-weight: 600;
  margin: 0 0 0.25rem;
}

.section-desc {
  color: var(--text-secondary);
  font-size: 0.875rem;
  margin: 0 0 1rem;
}

.theme-options {
  display: flex;
  gap: 1.5rem;
}

.radio-label {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  cursor: pointer;
  color: var(--text-primary);
}

.interface-select {
  padding: 0.5rem;
  border-radius: 0.5rem;
  background-color: var(--bg-card);
  color: var(--text-primary);
  border: 1px solid var(--border-color);
  width: 100%;
  max-width: 300px;
  outline: none;
}

.backend-options {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.backend-card {
  display: flex;
  align-items: flex-start;
  gap: 0.75rem;
  padding: 1rem;
  background-color: var(--bg-card);
  border: 1px solid var(--border-color);
  border-radius: 0.75rem;
  cursor: pointer;
  transition: border-color 0.2s, box-shadow 0.2s;
}

.backend-card:hover {
  border-color: #52525b;
}

.backend-card.selected {
  border-color: var(--accent-primary);
  box-shadow: 0 0 0 1px var(--accent-primary);
}

.radio-input {
  margin-top: 0.25rem;
  accent-color: var(--accent-primary);
}

.card-content {
  flex: 1;
}

.card-header {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin-bottom: 0.375rem;
}

.card-label {
  font-size: 0.9375rem;
  font-weight: 600;
  color: var(--text-primary);
}

.card-tag {
  padding: 0.125rem 0.5rem;
  font-size: 0.625rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  border-radius: 9999px;
}

.card-tag.singbox {
  background-color: rgba(34, 197, 94, 0.15);
  color: #22c55e;
}

.card-tag.dae {
  background-color: rgba(59, 130, 246, 0.15);
  color: #3b82f6;
}

.card-desc {
  font-size: 0.8125rem;
  color: var(--text-secondary);
  line-height: 1.5;
  margin: 0;
  margin-bottom: 0.5rem;
}

.install-btn {
  background-color: var(--bg-app);
  color: var(--text-primary);
  border: 1px solid var(--border-color);
  padding: 0.25rem 0.75rem;
  border-radius: 0.375rem;
  font-size: 0.75rem;
  cursor: pointer;
  transition: all 0.2s;
  margin-top: 0.5rem;
}

.install-btn:hover:not(:disabled) {
  background-color: rgba(255, 255, 255, 0.1);
}

.install-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
</style>
