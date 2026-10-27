import request from "@/utils/request";
const root = "/api/v1/cmdb/database-deployments";
export const databaseAutomation = {
  access: (id: number) =>
    request({ url: `${root}/${id}/orchestrator-access`, method: "get" }),
  list: () => request({ url: root, method: "get" }),
  create: (data: unknown) => request({ url: root, method: "post", data }),
  runs: (id: number) => request({ url: `${root}/${id}/runs`, method: "get" }),
  action: (id: number, data: unknown) =>
    request({ url: `${root}/${id}/actions`, method: "post", data }),
  view: (id: number, view: string) =>
    request({ url: `${root}/${id}/ha/${view}`, method: "get" }),
  ha: (id: number, data: unknown) =>
    request({ url: `${root}/${id}/ha`, method: "post", data, timeout: 60000 })
};
