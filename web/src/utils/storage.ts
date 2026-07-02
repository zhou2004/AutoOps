/**
 * AutoOps Storage 工具
 * 兼容旧版 $storage API
 */
import { storageLocal } from "@pureadmin/utils";

export default {
    getItem(key: string): any {
        return storageLocal().getItem(key);
    },
    setItem(key: string, value: any): void {
        storageLocal().setItem(key, value);
    },
    removeItem(key: string): void {
        storageLocal().removeItem(key);
    },
    clearAll(): void {
        storageLocal().clear();
    }
};
