import { ref, watch, toRaw } from 'vue';
import { LoadState, SaveState } from '../../wailsjs/go/main/App';
import { main } from '../../wailsjs/go/models';

const appState = ref<main.AppState | null>(null);
const isLoading = ref(true);

export function useAppState() {
    const loadState = async () => {
        try {
            isLoading.value = true;
            const state = await LoadState();
            appState.value = state;

            // Set defaults if empty
            if (!appState.value.canvasElements || appState.value.canvasElements.length === 0) {
                appState.value.canvasElements = [
                    { id: "app-other", type: "application", position: { x: 100, y: 100 }, data: { label: "Other", mode: "Standard" }, deletable: false },
                    { id: "tun-direct", type: "tunnel", position: { x: 600, y: 100 }, data: { label: "Direct", type: "Bypass" }, deletable: false },
                    { id: "tun-block", type: "tunnel", position: { x: 600, y: 300 }, data: { label: "Block", type: "Block" }, deletable: false },
                    { id: "e-other-direct", source: "app-other", target: "tun-direct", type: "wire", style: { stroke: "var(--accent-success)", strokeWidth: 2 }, deletable: false }
                ];
            }
            if (!appState.value.proxies) appState.value.proxies = [];
            if (!appState.value.settings) {
                appState.value.settings = new main.AppSettings({ theme: 'dark', defaultInterface: 'eth0' });
            }
        } catch (e) {
            console.error("Failed to load state", e);
        } finally {
            isLoading.value = false;
        }
    };

    const saveState = async () => {
        if (!appState.value) return;
        try {
            // Must convert Proxies & AppSettings back into class objects if they are raw proxies
            // Wails bindings handle this, we just need to pass the raw value
            // after stripping vue reactives.
            const raw = JSON.parse(JSON.stringify(toRaw(appState.value)));
            await SaveState(raw);
        } catch (e) {
            console.error("Failed to save state", e);
        }
    };

    return {
        appState,
        isLoading,
        loadState,
        saveState
    };
}
