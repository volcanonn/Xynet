<script setup lang="ts">
import { computed, ref } from 'vue';
import { Globe, X, Edit2, ChevronDown, ChevronRight, Check } from '@lucide/vue';
import { ImportWireguardConfig } from '../../../wailsjs/go/main/App';
import { useAppState } from '../../composables/useAppState';
import { useToast } from '../../composables/useToast';

const { appState, saveState } = useAppState();
const toast = useToast();

interface ProxyConfig {
  name: string;
  country: string;
  city: string;
  content: string;
}

const expandedCountries = computed(() => new Set(
  configs.value.map(c => c.country)
));
const expandedCitiesSet = computed(() => new Set(
  configs.value.map(c => `${c.country}:${c.city}`)
));

const editingName = ref('');
const editNameVal = ref('');

const startEdit = (name: string) => {
  editingName.value = name;
  editNameVal.value = name;
};

const saveEdit = (oldName: string) => {
  if (!appState.value?.proxies) return;
  const newName = editNameVal.value.trim();
  if (!newName || newName === oldName) {
    editingName.value = '';
    return;
  }
  
  // Update in proxies
  const idx = appState.value.proxies.findIndex(p => p.name === oldName);
  if (idx !== -1) {
    appState.value.proxies[idx].name = newName;
  }
  
  // Update any tunnel nodes referencing this label
  const oldLabel = oldName.replace(/\.(conf|txt)$/, '');
  const newLabel = newName.replace(/\.(conf|txt)$/, '');
  
  if (appState.value.canvasElements) {
    appState.value.canvasElements.forEach(el => {
      if (el.type === 'tunnel' && el.data?.label === oldLabel) {
        el.data.label = newLabel;
      }
    });
  }
  
  saveState();
  editingName.value = '';
};

const cancelEdit = () => {
  editingName.value = '';
};

const configs = computed<ProxyConfig[]>(() => {
  if (!appState.value?.proxies) return [];
  return appState.value.proxies.map(p => {
    let country = 'Imported';
    let city = 'Unknown';
    const parts = p.name.replace(/\.conf$/, '').split(/[-_]/);
    if (parts.length >= 2 && parts[0].length <= 3) {
      country = parts[0].toUpperCase();
      city = parts[1].toUpperCase();
    }
    return { name: p.name, country, city, content: p.content };
  });
});

const groupedConfigs = computed(() => {
  const groups: Record<string, Record<string, ProxyConfig[]>> = {};
  for (const config of configs.value) {
    if (!groups[config.country]) groups[config.country] = {};
    if (!groups[config.country][config.city]) groups[config.country][config.city] = [];
    groups[config.country][config.city].push(config);
  }
  return groups;
});

const importConfig = async () => {
  try {
    const imported = await ImportWireguardConfig();
    if (imported && imported.name) {
      if (!appState.value) return;
      if (!appState.value.proxies) appState.value.proxies = [];

      const exists = appState.value.proxies.some(p => p.name === imported.name);
      if (exists) {
        const idx = appState.value.proxies.findIndex(p => p.name === imported.name);
        appState.value.proxies[idx] = { name: imported.name, content: imported.content };
      } else {
        appState.value.proxies.push({ name: imported.name, content: imported.content });
      }
      saveState();
      toast.success('Config imported successfully');
    }
  } catch (e) {
    toast.error(`Import failed: ${e}`);
  }
};

const removeConfig = (name: string) => {
  if (!appState.value?.proxies) return;
  const idx = appState.value.proxies.findIndex(p => p.name === name);
  if (idx !== -1) {
    appState.value.proxies.splice(idx, 1);
    saveState();
  }
};
</script>

<template>
  <div class="view-container">
    <div class="header-section">
      <Globe :size="48" class="icon" />
      <h2>Proxies</h2>
      <p>Manage your imported proxies and VPNs here.</p>
      <button @click="importConfig" class="import-btn">Import Proxy Config</button>
    </div>

    <div v-if="configs.length > 0" class="proxy-list-container">
      <div v-for="(cities, country) in groupedConfigs" :key="country" class="country-group">
        <div class="group-header">
          <ChevronDown :size="18" />
          <span class="group-title">{{ country }}</span>
          <span class="badge">{{ Object.values(cities).flat().length }}</span>
        </div>

        <div class="cities-container">
          <div v-for="(serverList, city) in cities" :key="city" class="city-group">
            <div class="group-header city-header">
              <ChevronDown :size="16" />
              <span class="group-title">{{ city }}</span>
              <span class="badge">{{ serverList.length }}</span>
            </div>

            <div class="servers-container">
              <div v-for="server in serverList" :key="server.name" class="server-item">
                <div class="server-info">
                  <template v-if="editingName === server.name">
                    <input 
                      v-model="editNameVal" 
                      @keyup.enter="saveEdit(server.name)"
                      @keyup.esc="cancelEdit"
                      class="edit-input" 
                      autoFocus
                    />
                  </template>
                  <template v-else>
                    <span class="server-name">{{ server.name }}</span>
                  </template>
                </div>
                
                <div style="display: flex; gap: 0.25rem;">
                  <template v-if="editingName === server.name">
                    <button class="icon-btn" @click.stop="saveEdit(server.name)" title="Save">
                      <Check :size="16" />
                    </button>
                    <button class="icon-btn" @click.stop="cancelEdit" title="Cancel">
                      <X :size="16" />
                    </button>
                  </template>
                  <template v-else>
                    <button class="icon-btn" @click.stop="startEdit(server.name)" title="Rename">
                      <Edit2 :size="16" />
                    </button>
                    <button class="icon-btn delete-btn" @click.stop="removeConfig(server.name)" title="Remove">
                      <X :size="16" />
                    </button>
                  </template>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <div v-else class="empty-state">
      <p>No proxies imported yet. Click the button above to import a proxy config file.</p>
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

.proxy-list-container {
  width: 100%;
  max-width: 600px;
  display: flex;
  flex-direction: column;
  flex: 1;
  overflow-y: auto;
  background-color: var(--bg-card, #1e293b);
  border-radius: 8px;
  padding: 1rem;
  box-shadow: 0 4px 6px rgba(0,0,0,0.1);
}

.group-header {
  display: flex;
  align-items: center;
  padding: 0.75rem;
  cursor: pointer;
  border-radius: 4px;
  transition: background-color 0.2s;
  color: var(--text-primary, #f8fafc);
  font-weight: 600;
  user-select: none;
}

.group-header:hover {
  background-color: rgba(255, 255, 255, 0.05);
}

.city-header {
  padding-left: 2rem;
  font-weight: 500;
  font-size: 0.95em;
  color: var(--text-secondary, #94a3b8);
}

.group-title {
  margin-left: 0.5rem;
  flex: 1;
}

.badge {
  background-color: rgba(255, 255, 255, 0.1);
  padding: 0.1rem 0.5rem;
  border-radius: 12px;
  font-size: 0.8em;
}

.cities-container {
  display: flex;
  flex-direction: column;
}

.servers-container {
  display: flex;
  flex-direction: column;
  padding-left: 3.5rem;
  gap: 0.25rem;
  margin-top: 0.25rem;
  margin-bottom: 0.5rem;
}

.server-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.5rem 0.75rem;
  background-color: rgba(0, 0, 0, 0.2);
  border-radius: 4px;
  border: 1px solid transparent;
}

.server-item:hover {
  border-color: rgba(255, 255, 255, 0.1);
}

.server-info {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex: 1;
}

.server-name {
  color: var(--text-primary, #f8fafc);
  font-size: 0.9em;
}

.icon-btn {
  background: none;
  border: none;
  color: inherit;
  cursor: pointer;
  padding: 0.2rem;
  display: flex;
  align-items: center;
  opacity: 0.5;
  transition: all 0.2s;
  border-radius: 3px;
}

.icon-btn:hover {
  opacity: 1;
  background-color: rgba(255, 255, 255, 0.1);
}

.delete-btn:hover {
  color: #ef4444;
}

.edit-input {
  background: var(--bg-card);
  border: 1px solid var(--accent-primary);
  color: var(--text-primary);
  padding: 0.2rem 0.4rem;
  border-radius: 4px;
  font-size: 0.9em;
  width: 100%;
  outline: none;
}

.empty-state {
  color: var(--text-secondary);
  font-size: 0.875rem;
  text-align: center;
  max-width: 400px;
}
</style>
