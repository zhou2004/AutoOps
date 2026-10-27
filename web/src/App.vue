<script setup lang="ts">
import { computed, watch } from "vue";
import { useRoute } from "vue-router";
import { ElConfigProvider } from "element-plus";
import { ReDialog } from "@/components/ReDialog";
import zhCn from "element-plus/es/locale/lang/zh-cn";
import en from "element-plus/es/locale/lang/en";
import { locale, tr } from "@/locales";
import { getConfig } from "@/config";
const route = useRoute();
const currentLocale = computed(() => (locale.value === "en-US" ? en : zhCn));
watch(
  [locale, () => route.meta.title],
  () => {
    document.title = [tr(route.meta.title), getConfig()?.Title || "AutoOps"]
      .filter(Boolean)
      .join(" | ");
  },
  { immediate: true }
);
</script>
<template>
  <el-config-provider :locale="currentLocale">
    <router-view />
    <ReDialog />
  </el-config-provider>
</template>
