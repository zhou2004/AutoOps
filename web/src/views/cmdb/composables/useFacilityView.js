import { ref, onMounted, onBeforeUnmount, onActivated, nextTick } from "vue";

export function useFacilityAppearance() {
  const dark = ref(false);
  const read = () => {
    const root = document.documentElement;
    dark.value =
      root.classList.contains("dark") || root.dataset.theme === "dark";
  };
  read();
  let observer;
  onMounted(() => {
    observer = new MutationObserver(read);
    observer.observe(document.documentElement, {
      attributes: true,
      attributeFilter: ["class", "data-theme"]
    });
    read();
  });
  onBeforeUnmount(() => observer?.disconnect());
  return { dark };
}

// Use the actual page offset rather than assuming a fixed application header.
export function useFacilityViewport(element) {
  const height = ref("calc(100dvh - 110px)");
  let observer,
    frame = 0;
  function measure() {
    frame = 0;
    const el = element.value;
    if (!el) return;
    const viewport = window.visualViewport;
    const bottom =
      (viewport?.height || innerHeight) + (viewport?.offsetTop || 0);
    const wrap = el.closest(".el-scrollbar__wrap");
    const fullscreen = document.fullscreenElement === el;
    const top = fullscreen
      ? 0
      : Math.max(0, el.getBoundingClientRect().top + (wrap?.scrollTop || 0));
    height.value = `${Math.max(200, Math.floor(bottom - top - (fullscreen ? 0 : 12)))}px`;
  }
  function schedule() {
    cancelAnimationFrame(frame);
    frame = requestAnimationFrame(measure);
  }
  onMounted(() => {
    const wrap = element.value?.closest(".el-scrollbar__wrap");
    if (wrap) wrap.scrollTop = 0;
    observer = new ResizeObserver(schedule);
    if (element.value?.parentElement)
      observer.observe(element.value.parentElement);
    if (wrap) observer.observe(wrap);
    window.addEventListener("resize", schedule);
    window.visualViewport?.addEventListener("resize", schedule);
    document.addEventListener("fullscreenchange", schedule);
    nextTick(schedule);
  });
  onActivated(() => nextTick(schedule));
  onBeforeUnmount(() => {
    observer?.disconnect();
    cancelAnimationFrame(frame);
    window.removeEventListener("resize", schedule);
    window.visualViewport?.removeEventListener("resize", schedule);
    document.removeEventListener("fullscreenchange", schedule);
  });
  return { height };
}
