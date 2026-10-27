<script setup lang="ts">
import { ref } from "vue";
import { MoreFilled } from "@element-plus/icons-vue";
import type { DropdownInstance } from "element-plus";

defineProps<{ label: string }>();
const emit = defineEmits<{ command: [command: string] }>();
const trigger = ref<HTMLElement>();
const dropdown = ref<DropdownInstance>();
const menuOpen = ref(false);
</script>

<template>
  <span ref="trigger" class="action-icon-trigger">
    <el-dropdown
      ref="dropdown"
      trigger="click"
      placement="bottom-end"
      :teleported="true"
      @visible-change="menuOpen = $event"
      @command="emit('command', $event)"
    >
      <el-button
        :icon="MoreFilled"
        link
        type="primary"
        class="action-icon"
        :aria-label="label"
        @keydown.esc.stop="dropdown?.handleClose()"
      />
      <template #dropdown>
        <el-dropdown-menu><slot /></el-dropdown-menu>
      </template>
    </el-dropdown>
  </span>
  <!-- Keep the hint outside the dropdown's event and focus handling. -->
  <el-tooltip
    :virtual-ref="trigger"
    virtual-triggering
    :content="label"
    placement="top"
    :disabled="menuOpen"
    :enterable="false"
    :trigger-keys="[]"
  />
</template>
