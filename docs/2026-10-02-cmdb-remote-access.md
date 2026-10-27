# 资产连接、网络配置与细粒度授权

## 使用入口

主机、物理机、网络设备列表增加「连接中心」。一台资产可以保存多个独立连接，每个连接都有自己的地址、端口、账户、凭据、启停状态和协议设置。

| 资产 | 可用连接 | 用途 |
| --- | --- | --- |
| Linux 主机 | SSH、VNC、RDP | 终端与独立桌面服务 |
| Windows 主机 | RDP、VNC、SSH | 默认采用 RDP；可先登记资产，稍后添加连接 |
| 物理机 | SSH、VNC、RDP | 业务系统访问、独立管理地址或标准 VNC 控制台 |
| 路由器、交换机等网络设备 | SSH | 终端、配置备份和版本管理 |

主机创建表单选择系统类型；启用「稍后配置连接」可不填 SSH 信息。Windows 登记不再尝试运行 Linux 信息采集脚本。旧主机 SSH 字段和入口继续保留，新增连接档案单独管理。

在配置中心创建凭据后，在连接中心选择已有凭据。新接口仅返回凭据名称、类型、用户名和 ID。SSH 档案必须获取并确认 SHA256 主机指纹，后续指纹变化会阻止连接。VNC/RDP 使用密码类型凭据；RDP 默认 NLA，证书校验默认开启。

连接检测中，SSH 会实际认证；VNC/RDP 检测 TCP 端口可达性，完整认证结果由会话确认。网页地址只包含资产和连接 ID。后端签发有效期 60 秒的一次性票据，数据库仅保存其哈希；跨 API 副本也只能兑换一次。连接过程中每 20 秒复查资产动作、凭据授权与档案状态，最长会话为 4 小时。

浏览器通过登录认证申请票据时，后端同时绑定其 `Origin`（协议、域名和端口）。后续 WebSocket 必须来自同一页面来源，所以前端与 API 使用不同端口或域名时也可直接连接。其他来源会被拒绝，且不会消耗该票据。SSH 终端只有收到后端的 `ready` 确认后才显示「已连接」，后端建立会话失败时返回 `error` 事件，页面显示失败信息。

### SSH 故障时的连接条件

VNC/RDP 连接直接访问各自服务，不依赖 SSH 隧道。需要提前在目标系统启用服务并放通访问路径。

操作系统中的 VNC 服务通常不能在系统关机或内核故障时继续工作。此类故障需要 BMC/iDRAC/iLO 等独立控制器，且其控制台必须提供兼容的标准 VNC 服务。本次没有实现厂商专有 HTML5、Java KVM、IPMI 电源控制或 Redfish 接口。业务 IP 和 BMC IP 应分别配置。

## 网络设备配置管理

网络设备列表的「配置管理」提供平台能力、版本历史、脱敏预览、两个版本对比及操作审计。

| 平台 | 备份内容 | 恢复 |
| --- | --- | --- |
| OpenWrt / ImmortalWrt | `uci export` 完整 UCI 配置 | 支持写回；不自动重载服务或重启 |
| Cisco | `show running-config` | 当前仅备份与对比 |
| Huawei / H3C | `display current-configuration \| no-more` | 当前仅备份与对比 |
| Linux | 地址、IPv4/IPv6 路由诊断 | 当前仅快照 |

厂商 CLI 的 SSH exec 支持、分页和账户权限因设备而异，Cisco/Huawei/H3C 尚未在实机验证。检测到分页标记、空输出或命令错误时拒绝保存版本。

快照正文采用 AES-256-GCM 加密，包含资产归属校验与内容校验和。网页预览和差异显示隐藏常见密码、密钥、团体字等值；脱敏规则并不能识别所有厂商自定义敏感字段，因此 `config_view` 仍应只授予配置维护人员。快照元数据不包含明文配置。

恢复流程：校验资产归属和权限 → 输入完整设备名确认 → 解密并验证版本 → 备份当前配置并保存回退版本 → 在临时目录校验 UCI 导入 → 写回配置。设备已有未提交 UCI 变更时拒绝写入。成功后由维护人员在维护窗口决定重载服务；发生失败时保留回退版本和审计结果，不宣称已自动回滚。

单个命令最多 45 秒，输出上限 4 MiB；同一 API 进程内按设备串行执行。多 API 副本部署暂需将网络配置写操作固定到单个副本，避免并发恢复。

## 权限模型

授权规则在同一条规则中同时匹配「资产类型、资产范围、操作」。不允许把规则 A 的资产范围与规则 B 的操作拼在一起。用户 ID 使用完整 JSON 数组匹配，禁用用户组和过期规则不生效。列表先限制范围，再计数和分页；空范围返回空列表。

| 操作 | 权限值 |
| --- | --- |
| 列表 / 详情 | `list` / `get` |
| SSH / VNC / RDP | `connect_ssh` / `connect_vnc` / `connect_rdp` |
| 目录 / 上传 / 下载 / 删除文件 | `file_list` / `file_upload` / `file_download` / `file_delete` |
| 单条 SSH 命令 | `command` |
| 查看配置 / 备份 / 恢复 | `config_view` / `config_backup` / `config_restore` |
| 会话审计 | `audit` |
| 资产创建 / 修改 / 删除 | `create` / `update` / `delete` |
| 规则范围内全部动作 | `admin` |

连接档案新增、修改、删除要求该资产的 `update` 权限；绑定和使用凭据另需凭据授权。网络配置备份/恢复也会校验所选 SSH 档案的凭据权限。`admin` 仍受规则资产范围约束，平台管理员单独处理。

新图形连接可以配置只读，后端会丢弃键盘和鼠标控制事件。图形会话的剪贴板、文件流、驱动器映射和打印重定向暂不开放。SSH 交互终端本身允许远程账户可执行的命令；文件 API 的分权不能限制使用 SSH shell 读写文件，需要配合远端账户权限。

### 旧授权迁移注意点

- 旧 `connect` 仅兼容 SSH，不扩大到 VNC/RDP、命令或文件操作。
- 原来凭 `list` 读取详情的账户需增加 `get`；主机分组、机房等辅助列表也需要对应范围授权。
- 创建资产需要明确目标分组或机房；迁移资产需要同一规则覆盖源资产与目标范围。
- SQL 执行和 SQL 日志管理的旧路由缺少可可靠绑定的资产 ID，本次暂收紧为平台管理员专用；数据库资产台账继续使用细粒度授权。
- 普通用户的凭据授权需要同时配置；不要通过授予平台管理员来绕开连接分权。

## 部署

后端启动时 AutoMigrate 新增连接档案、票据、会话审计、配置快照和配置审计表，并为主机增加 `os_type`。上线前按已有流程备份数据库。前端新增锁定版本的 `guacamole-common-js` 依赖。

跨域连接修复在 `cmdb_connection_ticket` 增加 `origin` 字段（启动时自动迁移）；前后端一起更新后重新打开会话，获取新票据。

### 环境变量

| 名称 | 说明 |
| --- | --- |
| `CMDB_GUACD_ADDR` | 如 `guacd:4822`；未配置时 SSH 仍可用，VNC/RDP 显示网关未启用 |
| `CMDB_CONFIG_ENCRYPTION_KEY` | Base64 编码的 32 字节密钥；缺失时禁止备份/恢复，密钥需持久保存并与数据库一起备份 |
| `CMDB_REMOTE_ORIGINS` | 可选，仅用于签发时没有 Origin 的客户端票据，逗号分隔精确来源，不支持通配符。普通浏览器自动绑定申请票据时的来源，无需配置此变量；该变量不能覆盖已绑定的来源 |

首次生成配置加密密钥可用 `openssl rand -base64 32`，通过部署环境或 Secret 注入。不要在每次重启时重新生成，否则已有快照无法解密。多个 API 副本必须使用相同密钥。

### Docker Compose

先重新构建并发布包含本次修改的 API/Web 镜像。在仓库根目录设置上述环境变量后运行：

```bash
docker compose -f docker/docker-compose.yml -f docker/cmdb/docker-compose.override.yml up -d guacd devops-api devops-web
```

guacd 不发布主机端口，只对容器网络开放 4822。该服务没有应用层鉴权，不应直接暴露给浏览器或公网。API 需要能访问 guacd，guacd 需要能访问目标的 VNC/RDP 端口。

### Kubernetes

在与 `devops-api` 相同的命名空间创建 `autoops-cmdb` Secret，键名为 `config-encryption-key`。部署和补丁示例：

```bash
kubectl apply -f k8s/cmdb/guacd.yaml
kubectl patch deployment devops-api --type strategic --patch-file k8s/cmdb/api-patch.example.yaml
```

`NetworkPolicy` 只允许带 `io.kompose.service: devops-api` 标签的同命名空间 Pod 接入 guacd；集群网络插件需支持 NetworkPolicy。自行改过 API 标签时同步修改策略。反向代理需转发 WebSocket Upgrade，推荐浏览器与 API 采用同源 HTTPS/WSS。

## 测试设备记录（2026-10-02）

| 设备 | 只读验证结果 |
| --- | --- |
| `192.168.110.227` | 新后端完成指纹获取、绑定校验、SSH 登录及只读系统查询；Ubuntu 26.04；5900、5901、3389 未监听 |
| `192.168.110.1` | 新后端 SSH 登录成功；ImmortalWrt 25.12.1；小米 AX3600；只读 UCI 备份 15,590 字节，加密存储、校验和及脱敏读取验证通过 |

两台设备均通过新终端完整链路测试：HTTP 票据、WebSocket 连接、窗口调整、输入、二进制终端输出和结束审计。测试只运行 `printf`、`exit` 和系统只读查询。

测试没有改变两台设备的网络配置，没有执行恢复、重启或服务安装。设备密码未写入仓库。VNC/RDP 的真实桌面、物理机带外控制台及 Windows 实机尚需提供可达服务后验收。

实机备份测试使用临时 SQLite 数据库和临时加密密钥，不会向现有资产库自动添加测试设备或快照。需要长期管理时，在界面建立正式资产、凭据和连接档案。

### 回归验证

```bash
cd api
GOMAXPROCS=2 go test -p 1 ./api/cmdb/... ./middleware ./router/cmdb ./pkg/db
```

测试覆盖：Windows/稍后连接的主机登记、资产与动作共同授权、账号禁用与删除、凭据撤销、列表范围和分页、短期票据过期/重放/并发兑换、Origin 校验、Unicode 协议解析、模拟 guacd 握手及真实 WebSocket 的只读控制事件拦截。网络配置测试覆盖快照归属、密文完整性、脱敏、恢复前自动备份和失败审计。

此外已通过全后端包编译检查、CMDB 与权限包的 Go race 检测、前端 `pnpm build`、Compose 配置检查和 Kubernetes YAML 解析。浏览器使用模拟接口完成 14 个明暗主题、窄屏、连接及配置界面截图，未发现页面 JavaScript 错误；另验证 Windows 新增/编辑的实际请求和远程会话重连/断开。模拟接口检查不代表真实 Windows/VNC 桌面验收。

真实设备测试 `TestAssetConnectionLive` 默认跳过。运行前通过进程环境提供 `CMDB_TEST_HOST`、`CMDB_TEST_USER`、`CMDB_TEST_PASSWORD` 和可选 `CMDB_TEST_PORT`；设置 `CMDB_TEST_PLATFORM=openwrt` 才会增加只读 UCI 备份验证。该测试没有恢复配置路径。

同一组环境变量可启用 controller 包中的 `TestAssetTerminalLive`，验证真实 SSH 终端完整链路。未设置测试环境变量时默认跳过，不会主动访问设备。

### 连接故障修复验证

2026-10-02，已有资产 `lenovo`（物理机 5、连接 2）与 `xiaomi路由器`（网络设备 9、连接 1）的 SSH 检测均成功，但多次终端票据未被兑换，也没有连接审计。开发前端直连另一个端口的 API，旧 WebSocket 同源限制在 SSH 连接之前拒绝了握手。

修复后使用数据库中的现有档案、凭据、权限和主机指纹，通过真实浏览器验证：

- 前端 `127.0.0.1:18080`、临时 API `127.0.0.1:18000`，使用真实 JWT 认证和 Redis 会话校验。
- 两台设备均完成票据申请、跨域 WebSocket、SSH 就绪、键盘输入和 `printf` 输出校验、主动断开与重新连接；页面 JavaScript 错误为 0。
- 正常会话生成结束审计。测试没有修改设备配置或资产连接档案。
- 回归测试覆盖票据来源绑定、错误来源拒绝且不消耗票据、非法 Origin 拒绝、过期/重放/并发兑换、凭据撤销，以及 SSH 失败事件。

临时 API 只运行 CMDB 连接路由，测试后关闭。正常使用需启动更新后的后端，并刷新前端重新打开连接。

## 主题规范

三类资产列表采用统一的页面标题、统计摘要、筛选区、两层信息单元格、状态标签和固定操作区。浅色、深色及窄屏规则已写入 `.agents/skills/theme-design/SKILL.md`，样式实现位于 `web/src/style/theme.css`。

## 参考

- [Apache Guacamole 协议与握手](https://guacamole.apache.org/doc/gug/guacamole-protocol.html)
- [Apache Guacamole 连接配置](https://guacamole.apache.org/doc/gug/configuring-guacamole.html)
- [OpenWrt UCI 配置工具](https://openwrt.org/docs/guide-user/base-system/uci)
