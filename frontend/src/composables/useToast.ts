import { ref } from 'vue';

export interface Toast {
    id: number;
    message: string;
    type: 'success' | 'error' | 'info';
}

const toasts = ref<Toast[]>([]);
let nextId = 0;

export function useToast() {
    const show = (message: string, type: 'success' | 'error' | 'info' = 'info', duration = 3000) => {
        const id = nextId++;
        toasts.value.push({ id, message, type });
        setTimeout(() => {
            remove(id);
        }, duration);
    };

    const remove = (id: number) => {
        const index = toasts.value.findIndex(t => t.id === id);
        if (index !== -1) toasts.value.splice(index, 1);
    };

    return {
        toasts,
        show,
        error: (msg: string) => show(msg, 'error', 4000),
        success: (msg: string) => show(msg, 'success', 3000),
        info: (msg: string) => show(msg, 'info', 3000)
    };
}
