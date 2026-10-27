import request from "@/utils/request";
export async function databaseResult(promise) {
  const response = await promise;
  if (response.data?.code !== 200)
    throw new Error(response.data?.message || "数据库操作失败");
  return response.data.data;
}
const root = id => `cmdb/databases/${id}/mysql`;
export const mysqlWorkbench = {
  capabilities: id =>
    databaseResult(request({ url: `${root(id)}/capabilities`, method: "get" })),
  table: (id, params) =>
    databaseResult(
      request({
        url: `${root(id)}/table`,
        method: "get",
        params,
        timeout: 40000
      })
    ),
  data: (id, params) =>
    databaseResult(
      request({
        url: `${root(id)}/data`,
        method: "get",
        params,
        timeout: 40000
      })
    ),
  row: (id, data) =>
    databaseResult(
      request({ url: `${root(id)}/rows`, method: "post", data, timeout: 40000 })
    ),
  schema: (id, data) =>
    databaseResult(
      request({
        url: `${root(id)}/schema`,
        method: "post",
        data,
        timeout: 40000
      })
    ),
  export: (id, params) =>
    databaseResult(
      request({
        url: `${root(id)}/export`,
        method: "get",
        params,
        timeout: 40000
      })
    ),
  import: (id, data) =>
    databaseResult(
      request({
        url: `${root(id)}/import`,
        method: "post",
        data,
        headers: { "Content-Type": "multipart/form-data" },
        timeout: 45000
      })
    ),
  variables: id =>
    databaseResult(request({ url: `${root(id)}/variables`, method: "get" })),
  setVariable: (id, data) =>
    databaseResult(
      request({ url: `${root(id)}/variables`, method: "put", data })
    )
};
