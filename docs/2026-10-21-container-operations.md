# 工作负载操作与 Docker 容器传输

日期：2026-10-21

## 界面调整

- 容器管理侧栏不再单列弹性伸缩。工作负载行提供「弹性伸缩」「编辑 YAML」「更多操作」。
- 弹性伸缩在抽屉中打开，锁定当前集群、命名空间和工作负载，列表仅显示该工作负载的策略；新建策略自动填入目标。Deployment / StatefulSet 支持 HPA、VPA，DaemonSet 提供 VPA。
- 副本调整、重启、回滚、Pod 配置、调度和删除放入更多操作。工具栏的更多操作保留完整策略管理入口，包含 Knative KPA。
- Docker 页面移除「操作审计」页签及其前端请求；服务端审计记录及历史查询接口保留。
- Docker 容器、镜像、数据卷、Compose、网络操作收拢至更多操作。容器列表增加分页、中文状态、匹配数量、空态，适配明暗主题与窄屏。

## 容器复制、迁移

入口：Docker 管理 → 容器 → 更多操作 → 复制容器 / 跨节点迁移。

复制支持同节点或跨节点，迁移必须使用不同 Docker 引擎。用户选择目标资产、已授权 SSH 档案、新容器名称、启动选项和端口策略，并输入源容器名称确认。

1. 校验源资产 Docker 操作权限、目标资产操作权限，以及双方 SSH 凭据。迁移还需要源资产 `docker_remove` 权限。
2. 检查目标名称、引擎操作系统与架构，以及目标已有网络。以实际 Docker 引擎 ID 判断同节点，防止不同资产记录指向同一引擎而误迁移。
3. 运行中的源容器先停止，使用 `docker commit` 保存可写层；镜像通过 SSH 二进制流传输给目标 `docker load`，不将镜像整体读入内存。
4. 命名卷、匿名卷、目录绑定挂载的数据使用 `docker cp -a` 复制为独立目标命名卷。使用临时停止状态容器恢复数据，随后恢复最终挂载的只读属性。源数据卷始终保留。
5. 通过 Docker Engine 创建接口保留环境变量、Entrypoint、Cmd、用户、工作目录、资源限制、重启策略、健康检查等。配置经 SSH stdin 传输，不写入 shell 参数。移除源 Compose / Swarm 管理标签，目标容器独立管理。
6. 复制完成后恢复原本运行的源容器。迁移会在目标启动并通过健康检查后删除源容器；无健康检查时确认启动后仍运行。原本停止的容器可迁移为停止状态。
7. 失败时删除本次创建的目标容器、临时容器和数据卷，恢复原本运行的源容器。若目标清理失败、状态不确定，则迁移源保持停止并返回人工恢复提示，避免双方同时写入。目标已验证但源删除失败时保留目标和停止的源，并明确报错。

## 使用边界

- 这是需要停机窗口的复制/迁移，不迁移进程内存、连接或客户端流量。共享挂载的其他写入者应一同停止；数据库可按应用要求先做一致性备份。
- 双方节点需安装 Docker CLI、curl，SSH 用户应能操作当前 Docker context 的本机 Unix socket；支持 rootless socket 路径。SSH 连接沿用平台的主机指纹验证。
- 目标需具备同名网络；不复制静态 IP，由目标重新分配。可保留原宿主机端口或分配随机端口；同节点复制并启动运行中容器时使用随机端口。
- 目录绑定挂载转换为独立 Docker 命名卷，不覆盖目标宿主机原路径。tmpfs 保留配置，不复制内容。
- 暂不支持文件型绑定挂载、设备/GPU 绑定、共享其他容器命名空间、AutoRemove 容器及暂停/重启中的容器。文件型绑定挂载的数据复制会失败并进入恢复流程。
- 自定义运行时、日志驱动等仍需由目标 Docker 支持。Docker 操作无法跨引擎形成原子事务；服务进程被强制结束时需检查源容器和带 `autoops.transfer` 标签的目标资源。
- 后端整体超时为 30 分钟，前端该请求超时为 31 分钟；现有 Nginx 代理读取超时已覆盖该时间，无需部署配置调整。

## API

`POST /api/v1/cmdb/docker/:assetType/:assetId/containers/:container/transfer`

```json
{
  "connectionId": 11,
  "targetAssetType": "host",
  "targetAssetId": 2,
  "targetConnectionId": 22,
  "name": "web-copy",
  "migrate": false,
  "start": true,
  "preservePorts": true,
  "confirmName": "web"
}
```

成功返回 `containerId`、`name`、`migrated`、`volumes`。连接列表额外返回 `canDockerOperate`、`canDockerRemove` 以及逐连接 `canDocker`，用于前端权限展示；后端始终独立校验。

## 验证

- Go 单元测试覆盖配置保留、独立数据卷、端口与网络身份处理、参数拒绝、失败恢复、目标验证后删除、部分成功及权限边界。
- 已通过 `pnpm build`、`pnpm i18n:test` 及 CMDB 全部 Go 包测试；浏览器使用模拟接口检查菜单、同节点复制和跨节点迁移请求、伸缩目标预填、明暗主题和窄屏。
- 独立 `vue-tsc` 检查受仓库现有配置阻塞：`tsconfig.json` 的 `ignoreDeprecations: "6.0"` 与当前 TypeScript 版本不兼容（TS5103）；本次未调整该全局配置。
- 已通过双引擎集成测试：本地两个 Docker 28.1.1 引擎通过临时 SSH 测试服务完成真实复制、迁移，验证可写层、数据卷、只读目录、环境变量、资源限制、健康检查与源容器清理。可使用以下命令重复测试：

```bash
cd api
CMDB_TRANSFER_SOURCE_SOCKET=/var/run/docker.sock \
CMDB_TRANSFER_TARGET_SOCKET=/path/to/isolated-test-engine/docker.sock \
CMDB_TRANSFER_TEST_IMAGE=python:3.12-alpine \
go test ./api/cmdb/service -run '^TestDockerContainerTransferLive$' -count=1 -v
```

两个 socket 必须指向不同本地引擎，测试镜像需已存在于源引擎；测试仅创建和删除唯一命名的测试资源。

实现依据：[Docker cp 文档](https://docs.docker.com/reference/cli/docker/container/cp/)、[Docker Engine API](https://docs.docker.com/reference/api/engine/)。
