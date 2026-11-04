import * as ElementIcons from "@element-plus/icons-vue";
import { markRaw, type Component } from "vue";

// Route metadata and dynamically supplied menus share local, offline icons.
export const menuIconNames: Record<string, keyof typeof ElementIcons> = {
  "/": "HomeFilled",
  "/dashboard": "DataBoard",
  "/aiops": "ChatDotRound",
  "/aiops/diagnose": "FirstAidKit",
  "/aiops/models": "Cpu",
  "/aiops/skills": "MagicStick",
  "/aiops/workflows": "Share",
  "/aiops/history": "DocumentChecked",
  "/cmdb": "Monitor",
  "/cmdb/ecs": "Monitor",
  "/cmdb/group": "FolderOpened",
  "/cmdb/physical": "Cpu",
  "/cmdb/network": "Connection",
  "/cmdb/my-assets": "Key",
  "/database": "Coin",
  "/database/sources": "Coin",
  "/database/clusters": "Connection",
  "/database/storage": "Files",
  "/database/backups": "RefreshLeft",
  "/k8s": "Box",
  "/k8s/docker": "Box",
  "/k8s/list": "Connection",
  "/k8s/node": "Cpu",
  "/k8s/namespace": "CollectionTag",
  "/k8s/workload": "Grid",
  "/k8s/network": "Connection",
  "/k8s/config": "SetUp",
  "/k8s/storage": "Files",
  "/k8s/crd": "DocumentAdd",
  "/k8s/monitoring": "Odometer",
  "/monitor": "AlarmClock",
  "/monitor/domain": "Lock",
  "/monitor/api-endpoint": "Link",
  "/monitor/alarm/rules": "Bell",
  "/monitor/alarm/notify": "Message",
  "/monitor/datasource": "Coin",
  "/monitor/incident": "Warning",
  "/config": "SetUp",
  "/config/ecskey": "Key",
  "/config/accountauth": "UserFilled",
  "/config/keymanage": "Lock",
  "/task": "Operation",
  "/task/template": "Tickets",
  "/task/job": "List",
  "/task/ansible": "Connection",
  "/task/config": "SetUp",
  "/tools": "Tools",
  "/tools/ops": "Tools",
  "/tools/agent": "Cpu",
  "/app": "Collection",
  "/app/application": "Grid",
  "/app/quick-release": "Promotion",
  "/system": "Setting",
  "/system/authorization": "Lock",
  "/system/api-keys": "Key",
  "/system/admin": "User",
  "/system/role": "UserFilled",
  "/system/menu": "Menu",
  "/system/dept": "OfficeBuilding",
  "/system/post": "Postcard",
  "/system/machine": "House",
  "/monitor/loginlog": "UserFilled",
  "/monitor/operator": "Document",
  "/monitor/dblog": "DocumentCopy"
};

export function menuIcon(path: string, configured?: any): Component | string {
  if (
    typeof configured === "function" ||
    typeof configured?.render === "function"
  )
    return configured;
  if (configured && typeof configured === "object") return configured;
  if (typeof configured === "string" && configured.startsWith("IF-"))
    return configured;
  const key =
    typeof configured === "string"
      ? configured
          .replace(/^ep[/:]/, "")
          .replace(/(^|-)([a-z])/g, (_, _prefix, letter) =>
            letter.toUpperCase()
          )
      : "";
  const local =
    (ElementIcons as Record<string, Component>)[key] ||
    ElementIcons[menuIconNames[path]];
  return markRaw(local || ElementIcons.Menu);
}
