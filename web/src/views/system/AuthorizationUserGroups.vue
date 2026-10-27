<script setup lang="ts">
import { computed, defineAsyncComponent } from "vue";
import { useRoute, useRouter } from "vue-router";
const route = useRoute();
const router = useRouter();
const AssetGroups = defineAsyncComponent(
  () => import("@/views/cmdb/cmdbUserGroup.vue")
);
const ContainerGroups = defineAsyncComponent(
  () => import("@/views/K8s/k8s-permission.vue")
);
const groupType = computed({
  get: () => (route.query.groupType === "containers" ? "containers" : "assets"),
  set: groupType => router.replace({ query: { ...route.query, groupType } })
});
</script>
<template>
  <div class="authorization-groups">
    <el-tabs v-model="groupType">
      <el-tab-pane name="assets" :label="$tr('资产用户组')" />
      <el-tab-pane name="containers" :label="$tr('容器用户组')" />
    </el-tabs>
    <AssetGroups v-if="groupType === 'assets'" embedded />
    <ContainerGroups v-else embedded section="groups" />
  </div>
</template>

<style scoped>
.authorization-groups :deep(> .user-group-management),
.authorization-groups :deep(> .k8s-permission) {
  padding: 0;
  margin: 0;
  min-height: 0;
  background: transparent;
}
</style>
