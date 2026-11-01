# 数据库 VIP 配置、漂移验证与排查

更新：2026-11-01。本次已在授权的 `.51/.52/.53` 隔离联调环境部署，数据库和宿主机未重启。测试仅调整本模块独立的 HAProxy / Keepalived 服务。

## 1. 已配置的入口

| 集群 | VIP / 网卡 | 写入口 | 只读入口 | 承载主机 | VRID / 服务 |
| --- | --- | --- | --- | --- | --- |
| MySQL：`autoops-mysql-validation`（ID 4） | `192.168.80.60/24` / `ens33` | `192.168.80.60:23306` | `192.168.80.60:23307` | `.51`、`.52` | 84 / `autoops-db-vip-4` |
| PostgreSQL：`db-lab-postgresql-patroni`（ID 2） | `192.168.80.61/24` / `ens33` | `192.168.80.61:25432` | `192.168.80.61:25433` | `.51`、`.52` | 85 / `autoops-db-vip-2` |

数据库登录凭据沿用配置中心关联账号，本文不保存密码。数据源管理已登记各集群的 VIP 逻辑连接；名称以“`/VIP`”结尾，原双代理数据源继续可用。工作台优先使用 VIP 写入口，避免读副本延迟影响刚写入的数据查询。

Redis Sentinel 和 Redis Cluster 保持原生客户端入口，不套用 SQL VIP。

地址分配前已核对 CMDB 主机地址、实际网卡、路由、现有 Keepalived 配置，并在三台主机分别发送 ARP 探测，两个地址均未发现应答。它们是本实验网段的部署地址，不是其他环境可直接复用的地址。如果 VMware/网关开启 DHCP，请在地址池中排除或保留 `.60` 和 `.61`；当前没有管理 DHCP 地址池的凭据，ARP 探测不能代替 DHCP 预留。

## 2. 在页面配置和使用

1. 进入 **数据库管理 → 集群管理**。确认两个 HAProxy 成员处于健康状态，并位于不同 CMDB 主机。
2. 点击该行 **操作 → 配置虚拟 IP**。填写 `VIP/CIDR`、网卡、VRID，选择两个受管代理。新环境必须使用当地分配的地址，不照抄本教程的 IP。
3. 点击预检。系统核对主机权限、网卡网段、CMDB 地址冲突、ARP 应答、VRID、代理端口和配置归属。预检不通过时按提示处理后重新生成计划。
4. 点击 **确认执行**。不填写操作原因，不手输集群名。Ansible 逐台更新独立代理监听并安装 Keepalived，不覆盖发行版 `keepalived.service`。
5. 在任务中查看步骤与日志。成功后打开 **详情 → 拓扑**，查看 VIP、实际绑定主机和最近采集时间；数据源管理会出现 VIP 逻辑连接。
6. 客户端以 VIP 和上表端口连接，账号从配置中心获取。切换后客户端需正常重连；结果未知的 SQL 写请求不能直接重放。

当前界面不允许直接覆盖已登记 VIP 的地址、VRID 或承载成员；需要迁移地址时，先安排客户端入口迁移和旧配置退役。

## 3. 漂移验证

从集群行选择 **操作 → 验证 VIP 漂移 → 确认执行**。

任务执行顺序：

1. SSH 检查两个承载节点，确认仅一个节点绑定 VIP，两个独立 Keepalived 服务均运行。
2. 通过 VIP 连续查询 SQL 入口，确认路由到可写主库。
3. **先创建独立的 60 秒 systemd 恢复定时器**，再停止当前承载节点的本集群 Keepalived 服务。
4. 观察两台主机的实际地址，确认 VIP 唯一绑定到另一节点，并通过 VIP 连续完成三次 SQL 主库连接核验。
5. 启动原节点服务并核对两端恢复健康，取消该次恢复定时器，记录原/新承载、耗时与验证结果。

采用 `BACKUP + nopreempt`，原节点恢复后通常保持备用，不为了切回高优先级节点再造成一次漂移。下次验证会从当前承载节点发起，因此可以验证两个方向。

SQL 连接核验只执行只读查询，不修改业务表。任务中的 `elapsedMs` 包含故障注入、地址检查和连续连接核验时间，**不是写入 RTO，也不是数据节点故障的完整 HA 验收**。MySQL 数据节点仍由 Orchestrator 管理，PostgreSQL 仍由 Patroni 管理；VIP 不参与数据库选主。

如果 API 或 Worker 在停止服务后退出，恢复定时器在承载主机上独立执行。恢复状态不明时任务保留为“待核验”，从行操作进入“核验中断任务”，不会盲目再做一次故障注入。

## 4. 在主机检查

在 `.51` 和 `.52` 执行；以下以 MySQL 集群 4 为例，PostgreSQL 将 `4` 替换为 `2`：

```bash
ip -4 addr show dev ens33
systemctl status autoops-db-vip-4 --no-pager
journalctl -u autoops-db-vip-4 --since '-10 minutes' --no-pager
systemctl list-timers 'autoops-db-vip-4-rescue-*'
/usr/sbin/keepalived --config-test --use-file=/etc/autoops-database-vip/4/keepalived.conf
```

每个集群的目录与进程独立：

```text
/etc/autoops-database-vip/<集群ID>/keepalived.conf
/etc/autoops-database-vip/<集群ID>/health.py
/etc/systemd/system/autoops-db-vip-<集群ID>.service
/run/autoops-db-vip-<集群ID>.pid
/run/autoops-db-vip-<集群ID>-vrrp.pid
```

MySQL 代理服务分别为 `autoops-db-proxy-4-530`（`.51`）、`autoops-db-proxy-4-528`（`.52`）。PostgreSQL 代理服务名可从集群“节点”页查看。Keepalived 通过该代理服务状态及实际监听端口检查入口；不通过全局 sysctl 开启非本地 IP 绑定。

同一集群的两台主机使用单播 VRRP；网络 ACL 需要允许双方实 IP 间的 **IP 协议 112**，不是 TCP/UDP 112 端口。业务侧还需允许上表 SQL 端口。自动化没有关闭宿主机防火墙或修改 K8s 网络。

## 5. 常见问题

| 现象 | 检查与处理 |
| --- | --- |
| 两端均没有 VIP | 检查独立代理是否运行、端口是否监听、`health.py` 返回码及 Keepalived 日志；健康失败会阻止绑定 |
| 两端同时绑定 VIP | 检查单播 VRRP 连通性和重复 VRID；界面显示冲突，漂移验证不能算成功；先恢复控制网络并确认唯一承载 |
| VIP 已漂移，SQL 仍不可用 | 检查客户端重连、ARP 更新、HAProxy 角色探测和数据库主库状态；地址漂移成功不能替代业务入口验证 |
| 集群过期 / 未知 | 查看最后采集时间、API/Collector 日志、节点凭据和 SSH 指纹；运行中或待核验任务会占用集群锁，不能删锁掩盖现场 |
| 日志删除按钮没有显示 | 只有管理员可删除；运行中、结果待核验的任务不可删除，后端还检查集群锁和 API Key 授权 |
| 更新配置提示 VIP 已存在 | 不覆盖其他集群或外部 Keepalived；先核对地址和服务归属 |

## 6. 采集、Worker 与日志保留

API 启动时自动运行只读状态采集，每 15 秒扫描一次，超过 90 秒仍显示过期。独立 Worker 负责队列中的运维操作，部署时两者都要运行：

```bash
# 在 api 目录，各用一个终端或交由服务管理器运行
 go run . -c config.yaml
 go run . -mode database-worker -c config.yaml
# 可选：专用的只读采集进程，多个进程由租约协调
 go run . -mode database-collector -c config.yaml
```

数据库安装和 VIP 操作要求 Worker 能找到 `ansible-playbook`；开发环境可通过 `DATABASE_ANSIBLE_PLAYBOOK` 指定其路径。API、Worker、Collector、联调程序的 `TZ` 必须一致，元数据库 MySQL DSN 均使用 `loc=Local`。不能让测试程序写 UTC 无时区时间，而正式服务按北京时间解读。

Docker Compose 已提供 `database-worker`；Kubernetes 清单见 `k8s/api/database-worker-deployment.yaml`，它部署的是 AutoOps Worker，受管数据库依旧部署到 CMDB 主机。构建本次代码后同时更新 API 和 Worker 镜像。

**操作审计 → 日志保存周期**：默认 90 天，0 为永久。管理员可以修改周期、确认删除已结束任务日志或事件；过期清理不删除备份文件。内部保留最小任务回执、关联 ID 和原生事件去重标识，以保证幂等与恢复引用。运行中、待核验或仍持锁的任务不会被清理。

## 7. 设计依据

Keepalived 参数使用当前安装的 2.2.7 文档核对：单播 VRRP、`nopreempt`、脚本权限和配置验证见 [上游配置手册](https://github.com/acassen/keepalived/blob/v2.2.7/doc/man/man5/keepalived.conf.5.in)。实际测试结果和边界见 [数据库平台联调记录](2026-11-01-database-platform.md)。
