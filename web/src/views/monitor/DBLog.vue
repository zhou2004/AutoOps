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
    </el-tabs>
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
import DatabaseManagement from "@/views/database/Management.vue";
const route = useRoute(),
  router = useRouter();
const canDatabase = computed(() => canPlatform("database:read"));
const activeTab = computed({
  get() {
    if (route.query.tab === "database-audit" && canDatabase.value)
      return "database-audit";
    return platform.isAdmin ? "sql" : "database-audit";
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
