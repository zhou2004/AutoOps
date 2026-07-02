import { ref, reactive, computed, createApp } from 'vue'
import k8sApi from '@/api/k8s'

/**
 * K8s 权限组合式函数
 * 获取当前用户在 K8s 模块的有效权限（直接授权 + 用户组继承）
 * 提供权限检查方法供各个页面使用
 */

// 单例：全局共享权限状态
const permissions = ref([])
const loaded = ref(false)
const loading = ref(false)
let loadPromise = null

// 权限级别权重
const LEVEL_RANK = {
    readonly: 1,
    write: 2,
    admin: 3
}

// 权限级别名称
const LEVEL_NAMES = {
    readonly: '只读',
    write: '读写',
    admin: '管理员'
}

/**
 * 获取当前用户的最大权限级别
 * @param {number} clusterId
 * @param {string} namespace
 * @returns {{ level: string, rank: number }}
 */
function getEffectiveLevel(clusterId, namespace) {
    let highestLevel = ''
    let highestRank = 0

    for (const perm of permissions.value) {
        // 系统管理员（clusterId=0, namespace='*'）
        if (perm.clusterId === 0 && perm.namespace === '*' && perm.permissionType === 'admin') {
            return { level: 'admin', rank: 3 }
        }

        let match = false
        if (namespace && clusterId) {
            // perm.namespace 为空字符串代表集群级别权限（适用于所有命名空间）
            const nsMatch = perm.namespace === '*' || perm.namespace === '' || perm.namespace === namespace
            match = perm.clusterId === clusterId && nsMatch
        } else if (clusterId) {
            match = perm.clusterId === clusterId
        } else {
            match = true
        }

        if (match) {
            const rank = LEVEL_RANK[perm.permissionType] || 0
            if (rank > highestRank) {
                highestRank = rank
                highestLevel = perm.permissionType
            }
        }
    }

    return { level: highestLevel, rank: highestRank }
}

/**
 * 加载当前用户权限（只加载一次，后续复用）
 */
export function useK8sPermission() {
    const loadPermissions = async (force = false) => {
        if (loaded.value && !force) return permissions.value
        if (loading.value && loadPromise) return loadPromise

        loading.value = true
        loaded.value = false

        loadPromise = (async () => {
            try {
                const res = await k8sApi.getMyPermissions()
                const data = res.data
                if (data.code === 200) {
                    permissions.value = (data.data || []).map(p => ({
                        clusterId: Number(p.clusterId),
                        clusterName: p.clusterName || '',
                        namespace: p.namespace || '',
                        permissionType: p.permissionType || 'readonly',
                        source: p.source || 'direct',
                        rules: p.rules || []
                    }))
                    loaded.value = true
                } else {
                    permissions.value = []
                }
            } catch (err) {
                console.error('获取K8s权限失败:', err)
                permissions.value = []
            } finally {
                loading.value = false
                loadPromise = null
            }
        })()

        return loadPromise
    }

    /**
     * 检查用户是否有指定集群命名空间的访问权限
     */
    const hasPermission = (clusterId, namespace) => {
        const { rank } = getEffectiveLevel(clusterId, namespace)
        return rank > 0
    }

    /**
     * 检查用户是否有指定集群命名空间的写权限 (write 或 admin)
     */
    const hasWritePermission = (clusterId, namespace) => {
        const { rank } = getEffectiveLevel(clusterId, namespace)
        return rank >= LEVEL_RANK.write
    }

    /**
     * 检查用户是否有指定集群命名空间的管理员权限
     */
    const hasAdminPermission = (clusterId, namespace) => {
        const { rank } = getEffectiveLevel(clusterId, namespace)
        return rank >= LEVEL_RANK.admin
    }

    /**
     * 检查用户是否有指定集群的任意权限（不限命名空间）
     */
    const hasClusterPermission = (clusterId) => {
        const { rank } = getEffectiveLevel(clusterId, null)
        return rank > 0
    }

    /**
     * 获取用户在指定集群命名空间的权限级别名称
     */
    const getPermissionLevelName = (clusterId, namespace) => {
        const { level } = getEffectiveLevel(clusterId, namespace)
        return LEVEL_NAMES[level] || '无权限'
    }

    /**
     * 获取权限级别对应的 El-Tag 类型
     */
    const getPermissionTagType = (clusterId, namespace) => {
        const { level } = getEffectiveLevel(clusterId, namespace)
        const tagMap = {
            readonly: 'info',
            write: 'warning',
            admin: 'danger'
        }
        return tagMap[level] || 'info'
    }

    /**
     * 检查用户是否对指定的资源和动作有细粒度权限
     * @param {number} clusterId
     * @param {string} namespace
     * @param {string} resource 资源名称，如 'deployments', 'secrets' 等
     * @param {string} verb 操作，如 'get', 'list', 'create', 'update', 'delete' 等
     * @returns {boolean}
     */
    const hasResourcePermission = (clusterId, namespace, resource, verb) => {
        for (const perm of permissions.value) {
            // 系统管理员拥有所有权限
            if (perm.clusterId === 0 && perm.namespace === '*' && perm.permissionType === 'admin') {
                return true;
            }

            // 匹配集群和命名空间
            let match = false;
            // perm.clusterId 为 0 代表全部集群
            if (perm.clusterId === clusterId || perm.clusterId === 0) {
                // perm.namespace 为 "*" 或 ""(集群级别) 代表全部命名空间
                if (perm.namespace === '*' || perm.namespace === '' || perm.namespace === namespace) {
                    match = true;
                }
            }

            if (!match) continue;

            // 如果有 roles 并且有 rules (新版细粒度权限)
            if (perm.rules && perm.rules.length > 0) {
                for (const rule of perm.rules) {
                    const matchResource = rule.resources && (rule.resources.includes('*') || rule.resources.includes(resource));
                    const matchVerb = rule.verbs && (rule.verbs.includes('*') || rule.verbs.includes(verb));
                    if (matchResource && matchVerb) {
                        return true;
                    }
                }
            } else {
                // 回退到基于权限级别 (rank) 的判断 (兼容直接授权或只提供 permissionType 的情况)
                if (perm.permissionType === 'admin' || perm.permissionType === 'write') return true;
                if (perm.permissionType === 'readonly' && ['get', 'list', 'watch'].includes(verb)) return true;
            }
        }
        return false;
    }

    /**
     * 刷新权限
     */
    const refreshPermissions = async () => {
        return loadPermissions(true)
    }

    return {
        permissions,
        loaded,
        loading,
        loadPermissions,
        hasPermission,
        hasWritePermission,
        hasAdminPermission,
        hasClusterPermission,
        hasResourcePermission,
        getPermissionLevelName,
        getPermissionTagType,
        refreshPermissions
    }
}

export default useK8sPermission
