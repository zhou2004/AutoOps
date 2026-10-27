<script setup lang="ts">
import { locale, setLocale, tr, type AppLocale } from "@/locales";
</script>
<template>
  <el-dropdown
    trigger="click"
    @command="(value: AppLocale) => setLocale(value)"
  >
    <el-button text :aria-label="tr('语言')" class="language-switch"
      >{{ locale === "zh-CN" ? "中文" : "EN" }} ▾</el-button
    >
    <template #dropdown>
      <el-dropdown-menu>
        <el-dropdown-item command="zh-CN" :disabled="locale === 'zh-CN'"
          >简体中文</el-dropdown-item
        >
        <el-dropdown-item command="en-US" :disabled="locale === 'en-US'"
          >English</el-dropdown-item
        >
      </el-dropdown-menu>
    </template>
  </el-dropdown>
</template>
<style scoped>
.language-switch {
  min-width: 52px;
  color: var(--text-primary);
}
</style>
