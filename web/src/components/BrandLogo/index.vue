<script setup lang="ts">
import { onMounted, onBeforeUnmount } from "vue";
defineProps({ size: { type: [String, Number], default: 40 } });
let observer: MutationObserver;
onMounted(() => {
  const update = () => {
    const dark =
      document.documentElement.classList.contains("dark") ||
      document.documentElement.dataset.theme === "dark";
    const icon = document.querySelector<HTMLLinkElement>("#app-favicon");
    if (icon) icon.href = `/assets/brand/mark-${dark ? "dark" : "light"}.svg`;
  };
  observer = new MutationObserver(update);
  observer.observe(document.documentElement, {
    attributes: true,
    attributeFilter: ["class", "data-theme"]
  });
  update();
});
onBeforeUnmount(() => observer?.disconnect());
</script>
<template>
  <span
    class="autoops-brand-mark"
    :style="{ width: `${size}px`, height: `${size}px` }"
    role="img"
    aria-label="AutoOps"
  >
    <img
      class="brand-light"
      src="/assets/brand/mark-light.svg"
      alt=""
      width="128"
      height="128"
    />
    <img
      class="brand-dark"
      src="/assets/brand/mark-dark.svg"
      alt=""
      width="128"
      height="128"
    />
  </span>
</template>
<style>
.autoops-brand-mark {
  display: inline-flex;
  flex: none;
  vertical-align: middle;
}
.autoops-brand-mark img {
  width: 100%;
  height: 100%;
  object-fit: contain;
}
.autoops-brand-mark .brand-dark {
  display: none;
}
html.dark .autoops-brand-mark .brand-light,
html[data-theme="dark"] .autoops-brand-mark .brand-light {
  display: none;
}
html.dark .autoops-brand-mark .brand-dark,
html[data-theme="dark"] .autoops-brand-mark .brand-dark {
  display: block;
}
</style>
