import { canPlatform, platformAccess } from "@/utils/platformAccess";
/** Legacy button permission names resolve to the same module actions as the API. */
export function checkAuthority(
  permissionCode: string | string[],
  _legacyPermissions?: string[]
) {
  if (!platformAccess.ready) return false;
  if (platformAccess.isAdmin) return true;
  const codes = Array.isArray(permissionCode)
    ? permissionCode
    : [permissionCode];
  return codes.some(code => {
    if (!code) return false;
    const parts = code.split(":");
    if (parts.length === 2) return canPlatform(code);
    let module = (
      {
        base: "system",
        config: "config",
        monitor: "monitor",
        cmdb: "cmdb",
        task: "task",
        app: "app",
        tool: "tool",
        k8s: "k8s",
        aiops: "aiops"
      } as Record<string, string>
    )[parts[0]];
    if (!module || module === "system" || module === "config") return false;
    if (["operator", "loginLog"].includes(parts[1])) module = "audit";
    if (["database", "db", "sql"].includes(parts[1])) module = "database";
    const action = parts[parts.length - 1].toLowerCase();
    if (["list", "get", "info", "query", "view"].includes(action))
      return canPlatform(`${module}:read`);
    if (
      /start|stop|run|exec|connect|deploy|sync|restart|pause|resume|rollback/.test(
        action
      )
    )
      return canPlatform(`${module}:execute`);
    if (
      ["add", "create", "edit", "update", "delete", "clean", "import"].includes(
        action
      )
    )
      return canPlatform(`${module}:write`);
    return false;
  });
}
