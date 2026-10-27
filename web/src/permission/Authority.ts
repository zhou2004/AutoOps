import { watchEffect } from "vue";
import { checkAuthority } from "@/utils/authority";
import { ElMessage } from "element-plus";
const stops = new WeakMap<HTMLElement, () => void>();
const handlers = new WeakMap<HTMLElement, (e: Event) => void>();
export default {
  mounted(el: HTMLElement, binding: { value: string | string[] }) {
    const deny = (e: Event) => {
      if (!checkAuthority(binding.value)) {
        e.preventDefault();
        e.stopImmediatePropagation();
        ElMessage.warning("您没有该操作权限，请联系管理员");
      }
    };
    handlers.set(el, deny);
    el.addEventListener("click", deny, true);
    stops.set(
      el,
      watchEffect(() => {
        const allowed = checkAuthority(binding.value);
        el.classList.toggle("no-permission", !allowed);
        el.setAttribute("aria-disabled", String(!allowed));
        el.style.opacity = allowed ? "" : "0.5";
        el.style.cursor = allowed ? "" : "not-allowed";
      })
    );
  },
  unmounted(el: HTMLElement) {
    stops.get(el)?.();
    const handler = handlers.get(el);
    if (handler) el.removeEventListener("click", handler, true);
    stops.delete(el);
    handlers.delete(el);
  }
};
