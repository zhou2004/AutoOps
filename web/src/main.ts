import App from "./App.vue";
import router from "./router";
import { setupStore } from "@/store";
import { getPlatformConfig } from "./config";
import { MotionPlugin } from "@vueuse/motion";
import { createApp, type Directive } from "vue";
import { useElementPlus } from "@/plugins/elementPlus";
import { injectResponsiveStorage } from "@/utils/responsive";

import Table from "@pureadmin/table";
import ApiPlugin from "@/api/index";
import storage from "@/utils/storage";
import handleTree from "@/utils/common";

// 引入重置样式
import "./style/reset.scss";
// 导入公共样式
import "./style/index.scss";
// 导入 AutoOps 主题变量（兼容旧组件）
import "./style/theme.css";
// 导入旧组件全局样式
import "./style/global.css";
// 一定要在main.ts中导入tailwind.css，防止vite每次hmr都会请求src/style/index.scss整体css文件导致热更新慢的问题
import "./style/tailwind.css";
import "element-plus/dist/index.css";
// 导入字体图标
import "./assets/iconfont/iconfont.js";
import "./assets/iconfont/iconfont.css";

const app = createApp(App);

// 自定义指令
import * as directives from "@/directives";
Object.keys(directives).forEach(key => {
  app.directive(key, (directives as { [key: string]: Directive })[key]);
});

// 全局注册@iconify/vue图标库
import {
  IconifyIconOffline,
  IconifyIconOnline,
  FontIcon
} from "./components/ReIcon";
app.component("IconifyIconOffline", IconifyIconOffline);
app.component("IconifyIconOnline", IconifyIconOnline);
app.component("FontIcon", FontIcon);

// 全局注册按钮级别权限组件
import { Auth } from "@/components/ReAuth";
import { Perms } from "@/components/RePerms";
app.component("Auth", Auth);
app.component("Perms", Perms);

// 全局注册vue-tippy
import "tippy.js/dist/tippy.css";
import "tippy.js/themes/light.css";
import VueTippy from "vue-tippy";
app.use(VueTippy);

// 注册 AutoOps API 插件（提供 $api）
app.use(ApiPlugin);

// 全局注册所有 Element Plus 图标（兼容 Options API 组件）
import * as ElIconModules from '@element-plus/icons-vue'
for (let iconName in ElIconModules) {
  app.component(iconName, (ElIconModules as any)[iconName])
}

// 全局注入 AutoOps 工具
app.config.globalProperties.$storage = storage as any;
app.config.globalProperties.$handleTree = handleTree;

// 预加载 @/assets/image/ 下所有资源，确保 Vite 打包时将其包含在产物中
// 否则运行时动态 path 参数无法被 Rollup 静态分析，导致资源被 tree-shake
const assetImageModules = import.meta.glob<
  typeof import('*.svg') | typeof import('*.png') | typeof import('*.jpg')
>('@/assets/image/*', { eager: true, query: '?url', import: 'default' });

// 全局资源路径解析（兼容 @/ 别名在模板中无法被 Vite 解析的问题）
// 注意：Vite 编译 import.meta.glob 会将 @/ 别名解析为 /src/ 前缀，所以需要归一化
app.config.globalProperties.$getAssetUrl = (path: string): string => {
  const normalizedPath = path.replace('@/', '/src/');
  const cached = (assetImageModules as unknown as Record<string, string>)[normalizedPath];
  if (cached) return cached;
  // 兜底：其他路径的资源
  return new URL(normalizedPath, import.meta.url).href;
};

getPlatformConfig(app).then(async config => {
  setupStore(app);
  app.use(router);
  await router.isReady();
  injectResponsiveStorage(app, config);
  app.use(MotionPlugin).use(useElementPlus).use(Table);
  app.mount("#app");
});
