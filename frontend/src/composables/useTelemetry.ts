import { ref } from 'vue';
import { EventsOn } from '../../wailsjs/runtime/runtime';

export const upload = ref(0);
export const download = ref(0);
export const backendRunning = ref(false);
export const backendStatus = ref('offline');
export const backendName = ref('singbox');
export const voponoCount = ref(0);
export const deployedAppCount = ref(0);
export const deployedTunnelCount = ref(0);

export interface LogEntry {
  id: number;
  timestamp: string;
  source: string;
  message: string;
}
export const systemLogs = ref<LogEntry[]>([]);

let initialized = false;
let nextLogId = 0;

export function useTelemetry() {
    if (!initialized) {
        EventsOn('net-stats', (stats: any) => {
            upload.value = stats.upload;
            download.value = stats.download;
        });

        EventsOn('app-log', (entry: { timestamp: string; source: string; message: string }) => {
            // Assign a stable monotonic id so Logs.vue can key on it. Index-based
            // keys break when the buffer shift()s from the front (cap 5000):
            // every shift would change every index and force a full re-render.
            systemLogs.value.push({ id: nextLogId++, ...entry });
            if (systemLogs.value.length > 5000) {
                systemLogs.value.shift();
            }
        });

        EventsOn('service-status', (status: any) => {
            backendRunning.value = status.backendRunning;
            backendStatus.value = status.backendStatus;
            backendName.value = status.backendName;
            voponoCount.value = status.voponoCount;
        });
        


        initialized = true;
    }

    return {
        upload,
        download,
        backendRunning,
        backendStatus,
        backendName,
        voponoCount,
        deployedAppCount,
        deployedTunnelCount,
        systemLogs
    };
}
