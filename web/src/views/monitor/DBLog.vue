<template>
  <div class="data-log-page">
    <el-tabs v-model="activeTab">
      <el-tab-pane
        v-if="platform.isAdmin"
        :label="$tr('SQL 操作日志')"
        name="sql"
      />
      <el-tab-pane
        v-if="canDatabase"
        :label="$tr('数据库操作审计')"
        name="database-audit"
      />
      <el-tab-pane v-if="canWeb" :label="$tr('Web 操作审计')" name="web-audit" />
    </el-tabs>
    <WebManagement v-if="activeTab === 'web-audit' && canWeb" section="audit" embedded />
    <SQLDataLog v-if="activeTab === 'sql' && platform.isAdmin" />
    <DatabaseManagement
      v-else-if="activeTab === 'database-audit' && canDatabase"
      section="audit"
      embedded
    />
  </div>
</template>
<script setup lang="ts">
import { computed } from "vue";
import { useRoute, useRouter } from "vue-router";
import {
  platformAccess as platform,
  canPlatform
} from "@/utils/platformAccess";
import SQLDataLog from "./components/SQLDataLog.vue";
import WebManagement from "@/views/webservice/Management.vue";
import DatabaseManagement from "@/views/database/Management.vue";
const route = useRoute(),
  router = useRouter();
const canWeb = computed(() => canPlatform("web:read"));
const canDatabase = computed(() => canPlatform("database:read"));
const activeTab = computed({
  get() {
    if (route.query.tab === "web-audit" && canWeb.value) return "web-audit";
    if (route.query.tab === "database-audit" && canDatabase.value)
      return "database-audit";
    return platform.isAdmin ? "sql" : canDatabase.value ? "database-audit" : "web-audit";
  },
  set(tab: string) {
    router.replace({ query: { ...route.query, tab } });
  }
});
</script>
<style scoped>
.data-log-page {
  padding: 0 4px;
}
:deep(.embedded-management .db-hero) {
  padding-top: 8px;
}
</style>
