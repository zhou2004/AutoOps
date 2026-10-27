# Docker 管理增强（Compose / 镜像 / 终端 / 数据卷）

日期：2026-10-16

在原有「资产管理 → Docker 管理」基础上扩展，仍然复用资产的 SSH 连接档案执行远端 `docker` 命令，凭据不落浏览器。

## 新增能力

### 容器
- 按名称 / 镜像 / 端口 / ID 搜索，支持「全部 / 运行中 / 已停止 / 已暂停」状态分类查询。
- WebSocket 终端连接（与 K8s 容器终端一致，`docker exec -it`）。
- 容器内文件管理（列表 / 查看 / 编辑 / 上传 / 下载 / 新建目录 / 删除）。

### 镜像
- **镜像仓库**：内置镜像市场清单（约 60 个常用镜像，含分类/描述/Stars），支持关键词搜索、分页展示（每页 10），可一键「下载」到当前选中资产。
- 下载：通过镜像站前缀（渡渡鸟 / DaoCloud / dockerproxy / hub.rat.dev）拉取。
- 删除 / 导入（`docker load`）/ 导出（`docker save`）。
- **跨机复制/迁移**：目标从平台主机/物理机资产中选择（复用其 SSH 连接档案与凭据）；源机器 `docker save` → 后端中转 → 目标机器 `docker load`；迁移模式成功后删除源镜像。
- 复制（打标签）、重命名（打标签 + 删除旧标签）。
- 勾选批量操作：批量删除、批量导出下载。

### 数据卷
- 创建、查看详情（`docker volume inspect`）、删除、查看映射容器（`docker ps --filter volume=`）。
- 创建容器时支持数据卷映射（`-v 卷名:容器路径[:ro]`）。

### 网络
- 创建、查看详情（`docker network inspect`）、删除、容器连接/断开网络。

### Compose
- 面板内直接编辑 YML 并保存到远端 `/opt/autoops/compose/<project>/docker-compose.yml`。
- `up -d` / `down` / `restart` / `pull` / `ps` / `config` 生命周期操作。
- 项目列表以配置目录为数据源（不再依赖 `docker compose ls --format`，兼容旧版 compose）。

## 后端接口

统一前缀 `GET/POST... /api/v1/cmdb/docker/:assetType/:assetId`（`assetType` 为 `host` 或 `physical`）。

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/containers/:container/terminal` | WebSocket 容器终端（`shell`、`connectionId` 查询参数） |
| GET | `/containers/:container/inspect` | 容器详情 |
| GET | `/containers/:container/files` | 文件列表（`path`） |
| GET | `/containers/:container/files/content` | 文件内容 |
| PUT | `/containers/:container/files/content` | 覆盖写入文件 |
| POST | `/containers/:container/files/directory` | 新建目录 |
| POST | `/containers/:container/files/upload` | 上传文件（multipart，`connectionId` 走 query） |
| GET | `/containers/:container/files/download` | 下载文件 |
| DELETE | `/containers/:container/files` | 删除文件 |
| GET | `/docker/registry/repositories` | 镜像仓库列表（`q`、`page`、`pageSize`，内置清单分页） |
| POST | `/images/pull` | 下载/拉取镜像 |
| DELETE | `/images` | 批量删除（`images`、`force`） |
| POST | `/images/tag` | 打标签 |
| POST | `/images/rename` | 重命名 |
| POST | `/images/transfer` | 跨机复制/迁移（`image`、`targetAssetType`、`targetAssetId`、`migrate`） |
| GET | `/images/export` | 批量导出下载（`images` 逗号分隔） |
| POST | `/images/import` | 导入 tar（multipart） |
| POST | `/volumes` | 创建数据卷 |
| GET | `/volumes/:volume` | 详情 + 映射容器 |
| DELETE | `/volumes/:volume` | 删除（`confirmName`） |
| GET | `/volumes/:volume/containers` | 映射容器 |
| POST | `/networks` | 创建网络 |
| GET | `/networks/:network` | 详情 + 连接容器 |
| DELETE | `/networks/:network` | 删除（`confirmName`） |
| POST | `/networks/:network/connect` | 连接容器 |
| POST | `/networks/:network/disconnect` | 断开容器 |
| GET | `/compose` | 项目列表 |
| GET | `/compose/:project` | 读取 YML |
| POST | `/compose` | 保存（`project`、`content`、`up`） |
| DELETE | `/compose/:project` | 删除（`confirmName`） |
| POST | `/compose/:project/action` | `up/down/restart/pull/ps/config` |

## 关键实现

- `api/api/cmdb/service/dockerOperation.go`：`runWithTimeout` 抽象，`runLong`（15 分钟）用于 pull/save/load/push/compose up 等耗时操作。
- `api/api/cmdb/service/dockerContainerFile.go`：容器 exec 与 WebSocket 终端桥接（`docker exec -it`）。
- `api/api/cmdb/service/dockerImage.go`：镜像全生命周期 + SFTP 文件传输。
- `api/api/cmdb/service/dockerVolume.go`、`dockerCompose.go`：数据卷与 Compose。
- 权限沿用资产授权动作：`docker_view` / `docker_logs` / `docker_operate` / `docker_remove`。

## 前端

- `web/src/views/cmdb/DockerManagement.vue`：容器 / 镜像 / 数据卷 / Compose / 网络 / 审计 六个页签。
- `web/src/views/cmdb/DockerTerminal.vue`：WebSocket 终端（路由 `/cmdb/docker/terminal/:assetType/:assetId/:container`）。
- `web/src/views/cmdb/DockerFileManager.vue`：容器内文件管理弹窗组件。
- `web/src/api/cmdb.ts`：新增对应 API 封装。
