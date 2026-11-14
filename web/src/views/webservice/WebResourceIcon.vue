<template>
  <span
    class="web-resource-icon"
    :class="iconKind"
    :title="label"
    role="img"
    :aria-label="label"
  >
    <component :is="icon" />
  </span>
</template>
<script setup lang="ts">
import { computed } from "vue";
import Nginx from "~icons/simple-icons/nginx";
import Cloudflare from "~icons/simple-icons/cloudflare";
import Lock from "~icons/ep/lock";
import Document from "~icons/ep/document";
import Globe from "~icons/ep/monitor";
import Connection from "~icons/ep/connection";
import { tr } from "@/locales";
const props = withDefaults(
  defineProps<{ kind?: string; provider?: string; engine?: string }>(),
  { kind: "sources", engine: "nginx" }
);
const iconKind = computed(() =>
  props.kind === "dns-accounts" && props.provider === "cloudflare"
    ? "cloudflare"
    : ["sources", "clusters", "instances"].includes(props.kind) &&
        props.engine === "nginx"
      ? "nginx"
      : "resource"
);
const icon = computed(() =>
  iconKind.value === "nginx"
    ? Nginx
    : iconKind.value === "cloudflare"
      ? Cloudflare
      : props.kind.includes("certificate") || props.kind === "acme-accounts"
        ? Lock
        : props.kind === "sites"
          ? Globe
          : props.kind === "upstreams" || props.kind === "dns-accounts"
            ? Connection
            : Document
);
const label = computed(() =>
  iconKind.value === "nginx"
    ? "Nginx"
    : iconKind.value === "cloudflare"
      ? "Cloudflare"
      : tr("Web 资源")
);
</script>
<style scoped>
.web-resource-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  flex: 0 0 28px;
  color: var(--el-color-primary);
}
.web-resource-icon svg {
  width: 26px;
  height: 26px;
}
.nginx {
  color: #009639;
}
.cloudflare {
  color: #f48120;
}
</style>
