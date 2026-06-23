import { ref } from 'vue';
import { EventsOn } from '../../wailsjs/runtime/runtime';

export const upload = ref(0);
export const download = ref(0);
export const backendRunning = ref(false);
export const backendName = ref('singbox');
export const voponoCount = ref(0);
export const deployedAppCount = ref(0);
export const deployedTunnelCount = ref(0);

let initialized = false;

export function useTelemetry() {
    if (!initialized) {
        EventsOn('net-stats', (stats: any) => {
            upload.value = stats.upload;
            download.value = stats.download;
        });

        EventsOn('service-status', (status: any) => {
            backendRunning.value = status.backendRunning;
            backendName.value = status.backendName;
            voponoCount.value = status.voponoCount;
        });
        


        initialized = true;
    }

    return {
        upload,
        download,
        backendRunning,
        backendName,
        voponoCount,
        deployedAppCount,
        deployedTunnelCount
    };
}
