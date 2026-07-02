import { ref } from 'vue';

export function useRunProcess({ graph, cancelOnError = true }) {
    const isRunning = ref(false);
    const run = async (nodes: any) => { isRunning.value = true; };
    const stop = async () => { isRunning.value = false; };
    const reset = (nodes: any) => { isRunning.value = false; };
    return { run, stop, reset, isRunning };
}
