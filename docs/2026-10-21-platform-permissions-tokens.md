# 统一权限与用户 API Token

> 此文记录早期继承模式。当前入口、分配请求与授权规则请参阅 [API Key 独立权限与用户组](2026-10-24-api-key-scopes-user-groups.md)：角色权限入口已移除，新 Key 必须提交独立 scopes，旧 Key 保留本文所述兼容规则。

## 使用方式

1. 管理员进入 **基础管理 → 角色权限**，选择真实角色，配置各模块的查看、修改与执行能力。保存立即生效；“清空授权”会保存明确的空权限集。
2. 在 **资产授权** 和 **容器授权** 中配置具体资产、凭据、集群、命名空间及 Kubernetes RBAC 资源/操作范围。
3. 在 **用户信息 → 分配 API Token**，或 **基础管理 → API Token 管理** 中选择所属用户、名称和 1–365 天有效期。
4. 将仅显示一次的 Token 交付给所属用户或该用户授权的 Harness 等外部系统。调用方式：`Authorization: Bearer <TOKEN>`。

API Key 就是用户的 API Token。它绑定用户身份，不独立授予权限；每个用户可由管理员分配多个独立撤销的 Token。普通用户只能查看自己的记录，不能自行分配、撤销，也不能查询其他用户的 Token。管理员可按用户筛选所有记录。列表仅显示前缀，数据库仅保存 SHA-256 摘要。

登录 JWT 与 API Token 经过同一个认证、模块授权和资源授权链。用户禁用/删除、角色禁用、模块权限撤回会影响后续请求；Token 到期或撤销后不能发起新请求。已经执行的操作不会回滚；已建立的长连接不因此自动断开。Harness 工具每次调用还会重新核验账号、模块权限、会话授权、资产/凭据和发起 Token。

## 权限范围

| 模块 | 权限 |
| --- | --- |
| 数据看板 | `dashboard:read` |
| 资产 / Docker / 数据库 | `cmdb` / `docker` / `database` 的 `read`、`write`、`execute` |
| Kubernetes / 监控 / 任务 / 应用发布 | `k8s` / `monitor` / `task` / `app` 的 `read`、`write`、`execute` |
| 配置 / 运维工具 | `config` / `tool` 的 `read`、`write`、`execute` |
| AI 运维 | `aiops:read`、`aiops:chat`、`aiops:operate` |
| 基础管理 / 日志 | `system` / `audit` 的 `read`、`write` |

`write` 对应新增、修改、删除；`execute` 对应运行、启停、部署、迁移、远程终端等操作。任务定义可能启用定时执行，因此创建/修改可执行任务也要求 `task:execute`。通常同时分配该模块的 `read`，供页面及资源选择器使用。例如 Docker 页面还需要 `cmdb:read` 读取已授权资产，Harness 操作 Docker 需要 AI 与 Docker 对应权限及会话限时授权。

管理员身份只由启用的 `admin` 角色认定。用户、角色、菜单、部门、岗位、授权规则、Token 分配、集群接入、共享密钥和 Ansible 配置、全局调度管理等入口仅管理员可用；给普通角色勾选 `system:write` 不会提升为管理员。Agent 全局列表和统计仍由管理员查看，具体主机的监控和运维操作按资产范围授权。其他没有对象授权模型的模块使用模块级权限。

前端按当前权限过滤菜单、阻止直接打开无权访问的页面；已有 `v-authority` 按钮使用同一组模块权限。服务端始终独立校验，浏览器隐藏/禁用按钮不是安全边界。

## 补齐的检查

- 基础管理接口由仅校验登录改为管理员检查；个人资料接口强制读取当前用户，账号返回值不再包含密码摘要。
- 角色菜单授权同步提交并返回数据库错误，不再提前返回成功后异步写库。
- 任务日志 WebSocket 必须认证，支持与 HTTP/SSE 相同的用户 Token；按任务实际主机范围校验访问。
- 监控主机详情/批量查询、Agent 部署/重启/卸载、服务部署、任务执行检查主机范围；SSH 操作同时检查凭据使用权。
- K8s 命名空间读权限不能执行终端或文件操作；这些操作要求 `pods/exec:create`。Pod 日志要求 `pods/log:get`；节点修改按集群级 `nodes:update` 校验，排空还要求集群级 `pods:list/delete`（现有排空实现会删除 Pod）。
- K8s 资源校验匹配 API group、resource、verb 和绑定范围；通用 YAML 更新解析工作负载类型，聚合列表只查询被授权的类型，CRD 实例按实际 group/resource 校验。
- 禁用的 K8s 用户组不再贡献授权。跨命名空间存储迁移/切换属于管理员操作。
- JWT 使用随机签名材料和内置 24 小时过期时间；即使没有 Redis，也不接受无过期时间的会话。只接受 HS256。

## 升级和部署

启动自动迁移 `sys_role_permission` 和 `sys_api_key.issued_by`。角色尚未保存新权限配置时，从启用的旧菜单/按钮权限推导权限；菜单存在本身不会授予写权限。新模块未被旧授权覆盖时，由管理员在角色权限页明确授权。新配置保存后替代该角色的旧菜单推导结果，多角色取并集。

此前创建的 `aiops:*` Key 保留原来的接口范围，不会自动扩权。管理员需要撤销并重新分配，才能获得继承用户权限的 Token（内部标记 `user:inherit`）。旧登录 JWT 因签名和过期校验调整，需要重新登录。

生产环境为所有 API 副本设置相同的、至少 32 字节的高熵环境变量 `AUTOOPS_JWT_SECRET`。Compose 已接入同名变量；Kubernetes Deployment 从 `autoops-auth` Secret 的 `jwt-secret` 字段读取。不要将真实值提交到仓库。

```bash
# 为当前部署生成独立的签名材料；后续启动复用安全保存的同一值。
export AUTOOPS_JWT_SECRET="$(openssl rand -hex 32)"
# Docker Compose 会读取该变量。
# Kubernetes 使用相同材料：
kubectl create secret generic autoops-auth --from-literal=jwt-secret="$AUTOOPS_JWT_SECRET"
```

未设置时使用进程随机密钥，适合单进程本地开发；重启会使登录会话失效，多副本必须设置一致的环境变量。数据库中的 API Token 不受 JWT 签名密钥轮换影响。此次修改没有部署或改动运行中的服务。

## 接口

| 方法与路径（前缀 `/api/v1`） | 作用 |
| --- | --- |
| `GET /system/permissions/me` | 当前用户身份、管理员标记、有效模块权限 |
| `GET /system/permissions/catalog` | 管理员读取模块目录和角色列表 |
| `GET /system/permissions/roles/:id` | 管理员读取角色权限及旧授权推导标记 |
| `PUT /system/permissions/roles/:id` | 管理员提交 `{"permissions":["cmdb:read"]}`，可传空数组撤销全部 |
| `GET /system/api-keys?page=1&userId=2` | 管理员查询所有/指定用户；普通用户始终限于自己，每页 50 条 |
| `GET /system/api-keys/users?q=ops` | 管理员搜索所属用户 |
| `GET /system/api-keys/users/:id/permissions` | 管理员预览用户当前权限 |
| `POST /system/api-keys` | 管理员分配：`{"userId":2,"name":"Harness","days":30}` |
| `DELETE /system/api-keys/:id` | 管理员撤销 |

推荐请求头传递 Token。为兼容浏览器下载、SSE、WebSocket，统一认证也支持现有 `token` 参数和 WebSocket 协议头；不要将带 Token 的地址复制到聊天或日志。

## 验证

- 真实 Gin 路由注册覆盖 **670 个受保护接口**：全部具有已知权限映射，未认证请求均被拒绝；登录、验证码、Agent 心跳、告警 Webhook、远程连接票据和 Harness 回调保留各自的入口认证方式。
- Go 测试覆盖管理员分配、所属用户隔离、旧 Key 不扩权、角色清空/禁用、账号停用、Token 撤销/到期、JWT 算法和有效期、K8s 资源范围、跨模块主机/凭据授权和 Harness 权限撤回。
- 前端构建及国际化检查；浏览器使用测试数据验证角色保存/清空、按用户分配、一次展示、撤销、普通用户限制、直接路由拒绝、明暗主题和窄屏。

```bash
(cd api && go test ./pkg/authorization ./pkg/apikey ./pkg/jwt ./api/system/... ./middleware ./api/aiops/... ./api/task/... ./api/k8s/... ./router/... -count=1)
(cd web && pnpm build && pnpm i18n:test && pnpm i18n:audit)
```

全量 `vue-tsc` 仍受仓库已有 cmdb/k8s 等类型错误影响；使用 `--ignoreDeprecations 5.0` 验证时，本次新增 API Token、角色权限页面及公共权限 API/工具文件未出现在错误列表中。

界面示例（浏览器测试数据）：[角色权限](images/role-permissions-light.png)、[用户 API Token](images/user-api-tokens-dark.png)。
