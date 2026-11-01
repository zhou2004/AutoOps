<template>
  <img
    v-if="icon"
    :src="icon"
    :alt="name"
    :title="name"
    class="database-engine-icon"
  />
</template>
<script setup lang="ts">
import { computed } from "vue";
import mysql from "@/assets/image/mysql.svg?url";
import redis from "@/assets/image/redis.svg?url";
import postgresql from "@/assets/image/PostgreSQL.svg?url";
const props = defineProps<{ engine: string | number }>();
const engines: Record<string, string> = {
  "1": "mysql",
  "2": "postgresql",
  "3": "redis"
};
const icons: Record<string, string> = { mysql, redis, postgresql };
const names: Record<string, string> = {
  mysql: "MySQL",
  redis: "Redis",
  postgresql: "PostgreSQL"
};
const key = computed(
  () => engines[String(props.engine)] || String(props.engine).toLowerCase()
);
const icon = computed(() => icons[key.value]);
const name = computed(() => names[key.value] || key.value);
</script>
<style scoped>
.database-engine-icon {
  width: 32px;
  height: 32px;
  object-fit: contain;
  flex: 0 0 32px;
}
</style>
