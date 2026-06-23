import { ref } from 'vue';

export const activeRules = ref<any[]>([]);
export const voponoApps = ref<any[]>([]);

export function useActiveDeployment() {
    return {
        activeRules,
        voponoApps
    };
}
