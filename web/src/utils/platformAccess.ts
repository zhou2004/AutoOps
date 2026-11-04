import { reactive } from "vue";
import { getToken } from "@/utils/auth";
import { myPermissions } from "@/api/platformPermissions";
export const platformAccess = reactive({
  ready: false,
  isAdmin: false,
  permissions: [] as string[]
});
let loadedToken = "";
export async function refreshPlatformAccess() {
  const token = getToken();
  if (loadedToken !== token) {
    platformAccess.ready = false;
    platformAccess.permissions = [];
    platformAccess.isAdmin = false;
  }
  try {
    const data = await myPermissions();
    if (getToken() !== token) throw new Error("登录用户已切换，请重新加载");
    Object.assign(platformAccess, {
      ready: true,
      isAdmin: data.isAdmin,
      permissions: data.permissions
    });
    loadedToken = token;
  } catch (e) {
    Object.assign(platformAccess, {
      ready: false,
      isAdmin: false,
      permissions: []
    });
    throw e;
  }
}
export function canPlatform(permission: string) {
  return (
    platformAccess.ready &&
    (platformAccess.isAdmin || platformAccess.permissions.includes(permission))
  );
}
export function canVisitPlatform(path: string): boolean {
  if (["/not-found", "/access-denied", "/server-error"].includes(path))
    return true;
  if (/^\/(error|login|redirect)(\/|$)/.test(path)) return true;
  if (!platformAccess.ready) return false;
  if (platformAccess.isAdmin) return true;
  if (["/system/api-keys", "/system/personal"].includes(path)) return true;
  if (path === "/host-terminal") return canPlatform("cmdb:execute");
  if (path === "/monitor/dblog") return canPlatform("database:read");
  if (path === "/system/machine") return canPlatform("cmdb:read");
  if (
    /^\/system(\/|$)/.test(path) ||
    [
      "/cmdb/asset-permission",
      "/cmdb/user-group",
      "/cmdb/credential-permission",
      "/task/config",
      "/aiops/models",
      "/aiops/skills",
      "/aiops/workflows",
      "/config/keymanage",
      "/tools/agent"
    ].includes(path)
  )
    return false;
  if (
    ["/monitor/loginlog", "/monitor/operator", "/monitor/dblog"].includes(path)
  )
    return canPlatform("audit:read");
  if (/^\/(cmdb|k8s)\/docker(\/|$)/.test(path))
    return canPlatform(
      path.includes("/terminal/") ? "docker:execute" : "docker:read"
    );
  if (path.startsWith("/cmdb/db") || /^\/database(\/|$)/.test(path))
    return canPlatform("database:read");
  if (
    path.startsWith("/k8s/terminal") ||
    path === "/cmdb/remote" ||
    path === "/cmdb/ssh"
  )
    return canPlatform(
      path.startsWith("/k8s") ? "k8s:execute" : "cmdb:execute"
    );
  const prefix = path.split("/")[1];
  const module = (
    {
      cmdb: "cmdb",
      k8s: "k8s",
      monitor: "monitor",
      task: "task",
      app: "app",
      config: "config",
      configcenter: "config",
      tools: "tool",
      tool: "tool",
      aiops: "aiops",
      dashboard: "dashboard",
      welcome: "dashboard",
      home: "dashboard"
    } as Record<string, string>
  )[prefix];
  return !!module && canPlatform(`${module}:read`);
}
export function filterPlatformMenus(routes: any[]): any[] {
  return routes.flatMap(route => {
    // Compatibility redirects and terminal/detail routes are never menu entries.
    if (route.meta?.showLink === false) return [];
    const children = route.children?.length
      ? filterPlatformMenus(route.children)
      : undefined;
    if (children ? !children.length : !canVisitPlatform(route.path)) return [];
    const copy = { ...route };
    if (children) {
      copy.children = children;
      if (
        copy.redirect &&
        typeof copy.redirect === "string" &&
        !canVisitPlatform(copy.redirect)
      )
        copy.redirect = children[0].path;
    }
    return [copy];
  });
}
