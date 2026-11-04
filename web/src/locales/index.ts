import { createI18n } from "vue-i18n";
import { watch } from "vue";
import english from "./en-US.json";

export type AppLocale = "zh-CN" | "en-US";
const entries = Object.entries(english);
const ids = new Map(entries.map(([source], index) => [source, `m${index}`]));
const zh = Object.fromEntries(
  entries.map(([source], index) => [`m${index}`, () => source])
);
const en = Object.fromEntries(
  entries.map(([, target], index) => [`m${index}`, () => target])
);
const stored = localStorage.getItem("autoops-locale");
export const i18n = createI18n({
  legacy: false,
  globalInjection: true,
  locale: stored === "en-US" ? "en-US" : "zh-CN",
  fallbackLocale: "zh-CN",
  missingWarn: false,
  fallbackWarn: false,
  messages: { "zh-CN": zh, "en-US": en }
});
export const locale = i18n.global.locale;
export function setLocale(value: AppLocale) {
  locale.value = value;
}
// Source-language keys preserve business values and support legacy Vue components.
// Message functions keep literal JSON, Go templates, @mentions and pipes intact.
// All translated strings live in one auditable catalog. Unknown/server text stays intact.
export function tr(source: unknown, params?: Record<string, unknown>): string {
  const text = String(source ?? "");
  const key = ids.get(text);
  // Reading locale also makes fallback values reactive in computed/template callers.
  void locale.value;
  const translated = key ? i18n.global.t(key) : text;
  return params
    ? translated.replace(/\{([A-Za-z][\w]*)\}/g, (placeholder, name) =>
        Object.prototype.hasOwnProperty.call(params, name)
          ? String(params[name] ?? "")
          : placeholder
      )
    : translated;
}
export function localizeError(message: string): string {
  const parts = message.split(" / ");
  return parts.length === 2
    ? parts[locale.value === "en-US" ? 0 : 1]
    : tr(message);
}
watch(
  locale,
  value => {
    localStorage.setItem("autoops-locale", value);
    document.documentElement.lang = value;
  },
  { immediate: true }
);

declare module "vue" {
  interface ComponentCustomProperties {
    $tr: typeof tr;
  }
}
