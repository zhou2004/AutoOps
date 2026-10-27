<script setup lang="ts">
import { computed, defineAsyncComponent } from "vue";
import { useRoute, useRouter } from "vue-router";

defineOptions({ name: "AuthorizationManagement" });
const route = useRoute();
const router = useRouter();
const panels = {
  assets: defineAsyncComponent(
    () => import("@/views/cmdb/assetPermission.vue")
  ),
  containers: defineAsyncComponent(
    () => import("@/views/K8s/k8s-permission.vue")
  ),
  groups: defineAsyncComponent(() => import("./AuthorizationUserGroups.vue"))
};
const activeTab = computed({
  get: () => {
    const tab = String(route.query.tab || "assets");
    return Object.hasOwn(panels, tab) ? tab : "assets";
  },
  set: tab => router.replace({ query: { ...route.query, tab } })
});
</script>

<template>
  <div class="authorization-page">
    <header class="page-header">
      <div>
        <h1>{{ $tr("授权管理") }}</h1>
        <p>{{ $tr("集中管理资产范围、容器权限和用户组。") }}</p>
      </div>
    </header>
    <section class="authorization-surface">
      <el-tabs v-model="activeTab" class="authorization-tabs">
        <el-tab-pane name="assets" :label="$tr('资产授权')" />
        <el-tab-pane name="containers" :label="$tr('容器授权')" />
        <el-tab-pane name="groups" :label="$tr('用户组')" />
      </el-tabs>
      <p class="authorization-help">
        <template v-if="activeTab === 'assets'">
          {{
            $tr(
              "管理主机、物理机、网络设备与数据库的操作权限，包含远程连接和 Docker 操作。"
            )
          }}
        </template>
        <template v-else-if="activeTab === 'containers'">
          {{
            $tr(
              "管理 Kubernetes 集群、命名空间及 RBAC 角色绑定；Docker 操作按所属资产授权。"
            )
          }}
        </template>
        <template v-else>
          {{
            $tr("集中维护资产用户组、容器用户组及成员，按资源类型分配权限。")
          }}
        </template>
      </p>
      <div class="authorization-panel" :data-panel="activeTab">
        <component :is="panels[activeTab]" :key="activeTab" embedded />
      </div>
    </section>
  </div>
</template>

<style scoped>
.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 16px;
  margin-bottom: 20px;
}
.page-header h1 {
  margin: 0;
}
.page-header p,
.authorization-help {
  color: var(--text-secondary);
  font-size: 13px;
  line-height: 1.7;
  margin: 8px 0 0;
}
.authorization-surface {
  padding: 20px;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--ui-radius-panel);
}
.authorization-help {
  margin: 0 0 20px;
}
.authorization-panel :deep(> div) {
  padding: 0 !important;
  margin: 0 !important;
  min-height: 0;
  background: transparent;
}
.authorization-panel :deep(.main-card) {
  border: 0 !important;
  box-shadow: none !important;
}
.authorization-panel :deep(.main-card > .el-card__body) {
  padding: 0;
}
@media (max-width: 768px) {
  .authorization-surface {
    padding: 12px;
  }
}
</style>
