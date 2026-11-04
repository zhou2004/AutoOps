# MinIO 备份仓库部署与使用

## 本次已部署环境

| 项目 | 值 |
| --- | --- |
| CMDB 主机 | `192.168.80.53`，主机 ID `531` |
| S3 Endpoint | `http://192.168.80.53:9000` |
| 控制台 | <http://192.168.80.53:9001> |
| Bucket / Region | `autoops-database-backups` / `us-east-1` |
| 平台仓库 | `minio-53-database-backups`，业务分组 `18`，仓库 ID `3` |
| systemd 服务 | `autoops-minio.service`，已开机自启 |
| 程序 / 数据 | `/usr/local/bin/autoops-minio` / `/var/lib/autoops-minio/data` |
| 服务配置 | `/etc/systemd/system/autoops-minio.service` |
| 私有环境文件 | `/etc/autoops-minio/environment`，root 所有、权限 `0600` |
| 版本 | `RELEASE.2025-04-22T22-12-26Z`，官方标签源码构建 |
| 资源限制 | 独立用户 `autoopsminio`，内存上限 1 GiB、CPU 配额 150% |

两组凭据均保存在 **配置中心 → 密钥管理**，文档和仓库中不保存明文：

- `AutoOps MinIO .53 root`（密钥记录 `7`）：用于登录 MinIO 控制台和管理桶、用户、策略。密钥 ID 是登录用户名，Secret 是密码。
- `AutoOps MinIO .53 backup`（密钥记录 `8`）：平台仓库使用此记录，只允许对专用桶列举、上传、读取及删除对象，不能管理 MinIO 用户。

没有使用 root 账号执行日常数据库备份，没有创建公开读写桶。凭据轮换时需同时更新配置中心引用和 MinIO 对应用户；不要重新生成 root 密码后遗漏服务环境文件。

## 为什么使用这个版本

按本次需求使用 2025 年 4 月的版本，早于 2025 年 5 月的控制台改动。版本依据：[官方发行标签](https://github.com/minio/minio/releases/tag/RELEASE.2025-04-22T22-12-26Z)、[官方控制台变更](https://github.com/minio/minio/pull/21278)。

2026-11-02 实施时，上游旧版二进制下载返回 410，公开镜像仓库也无法匿名拉取该包，因此没有使用第三方镜像。安装程序从官方标签源码构建；提交为 `0d7408fc9969caf07de6a8c3a84f9fbb10a6739e`，构建工具为 Go `1.25.8 linux/amd64`。源码和程序 SHA-256 记录在 [minio-version.json](../api/automation/database-platform/minio-version.json)。它是固定源码构建版，不是声称重新获得了官方二进制发行包。

此旧版用于当前内网联调，不是当前安全补丁版本；不开放公网，也不启用 OIDC/LDAP/STS 集成。MinIO 与数据库节点共用 `.53` 的故障域，且当前数据落在该主机根文件系统，**这不是生产异机/异地容灾副本**。生产使用独立存储主机/磁盘与 TLS，核对受支持版本和安全更新后再采用。

## 如何在页面使用

1. 打开“数据库管理 → 备份恢复 → 备份仓库”，位置输入 `192.168.80.53` 或 `autoops-database-backups` 搜索，检查 `minio-53-database-backups` 为正常。
2. 切换需要的数据库类型，选择操作集群，点击“立即备份”，保存位置选择此仓库。同业务分组的 MySQL、PostgreSQL、Redis 可共用此桶，具体备份方式仍受引擎版本和预检约束。
3. 确认任务成功后在“恢复演练”查看新备份，副本位置应显示“独立仓库已核验”。这表示上传后下载对象并计算 SHA-256 通过，不只是调用了上传接口。
4. 在该行“操作 → 恢复验证”，选择隔离主机及空闲端口。恢复从 S3 取回并校验备份，实际启动恢复实例、核验数据，再停止实例并保留目录。
5. 验证成功后记录进入“备份集”，详情里查看恢复报告。正式业务回灌或入口切换按 [备份恢复使用指南](database-backup-guide.md) 单独安排。
6. 需要删除某个备份时使用“清理备份”。本地包及桶中的对应对象都将被删除；未过期备份也可清理。仓库本身的“删除”仅删除平台登记，不清空桶。

桶内对象路径采用 `集群ID/备份任务ID/member_成员ID.tar`。不要手动改名或覆盖这些对象，也不要给这个桶配置与平台不一致的自动过期生命周期，否则历史恢复清单会失效。

## 可重复部署

自动化入口：[minio.yml](../api/automation/database-platform/minio.yml)。它只管理独立服务、目录和端口，不格式化磁盘、不修改现有 MySQL、Redis、PostgreSQL 或 Kubernetes 服务。

先在控制机获取固定源码并验证已记录的源码散列（以下工作目录由操作者自行选择）：

```bash
mkdir -p /tmp/autoops-minio-build
cd /tmp/autoops-minio-build
curl -fL --retry 2 'https://codeload.github.com/minio/minio/tar.gz/refs/tags/RELEASE.2025-04-22T22-12-26Z' -o source.tar.gz
printf '%s\n' '1a63209927b9d1fb239b2a776b7b01d1902eff19379a108f4bc7005edcc0968c  source.tar.gz' | sha256sum -c -
mkdir source
tar -xzf source.tar.gz -C source --strip-components=1
cd source
CGO_ENABLED=0 GOTOOLCHAIN=local go build -p 4 -trimpath \
  -ldflags '-s -w -X github.com/minio/minio/cmd.Version=2025-04-22T22:12:26Z -X github.com/minio/minio/cmd.ReleaseTag=RELEASE.2025-04-22T22-12-26Z -X github.com/minio/minio/cmd.CopyrightYear=2025 -X github.com/minio/minio/cmd.CommitID=0d7408fc9969caf07de6a8c3a84f9fbb10a6739e -X github.com/minio/minio/cmd.ShortCommitID=0d7408fc9969c' \
  -o ../minio .
../minio --version
sha256sum ../minio
```

Go 版本或构建环境变化会影响程序散列。使用本次构建产物时应匹配记录的散列；重新构建时先记录工具链、来源及新散列，不能随意把下载文件的散列当成可信来源。

准备受信任的 CMDB SSH inventory（本次执行复用平台 `InventoryHosts` 的主机指纹和凭据解析），并通过 **Ansible Vault 或权限 0600 的临时变量文件** 提供：

| 变量 | 用途 |
| --- | --- |
| `minio_bind_ip` | 明确指定当前主机已配置的地址，不选择新 VIP |
| `minio_binary` | 控制机已验证的程序绝对路径 |
| `minio_binary_sha256` | 该受信任产物的 SHA-256 |
| `minio_root_user` / `minio_root_password` | 从配置中心安全取得 root 凭据，不能写到命令行或 Git |
| `minio_api_port` / `minio_console_port` | 默认 9000 / 9001，首次部署检查端口空闲 |

执行命令仅引用变量文件：

```bash
ansible-playbook -i inventory.json api/automation/database-platform/minio.yml \
  --extra-vars @minio-vault.yml --ask-vault-pass
```

控制台登录 root 后创建私有桶 `autoops-database-backups`；在 Identity 中创建专用备份用户并绑定仅限此桶的读写策略。平台仓库引用这个备份用户在配置中心的凭据，不引用 root。此次这些步骤已完成，无需重复创建。

运行检查：

```bash
systemctl status autoops-minio --no-pager
journalctl -u autoops-minio -n 60 --no-pager
curl -fsS http://192.168.80.53:9000/minio/health/live
```

连接检查执行主机需要 `python3-boto3`，部署脚本已安装。不要在终端打印环境文件。服务升级/变更按独立变更安排，不跟随 `latest` 自动更新。

## 本次验证记录（2026-11-03）

- Ansible 部署及重复执行通过；专用服务开机自启，9000/9001 可达。
- 专用备份账号的上传、下载、列举、SHA-256 校验、删除通过；管理员接口访问被拒绝。
- 平台仓库连接检查正常；浏览器登录旧版控制台，确认 Buckets、Policies、Identity 等管理导航可用。
- Redis Sentinel 隔离集群完成 S3 备份，备份 `54` 经任务 `246` 在 `.51:19395` 隔离恢复，实际测试键值匹配；恢复实例已停止，报告与目录保留。
- 新建备份 `55` 未修改创建时间，通过任务 `248` 清理；S3 前缀列举确认对应对象已不存在，其他备份保留。
- 过程中修复旧版 Ansible 不支持 `apt.lock_timeout` 的兼容问题。中断任务 `240`、`243` 已通过正式核验流程确认为失败并释放执行锁，保留审计记录，未伪造验证成功。

上述证明 S3 存储与实际恢复链路可用，不代表对三种引擎所有备份方式、生产灾难场景或 MinIO 单机高可用作出验收。

## 应用侧回归

后端数据库模块、仓库权限/引用、备份视图分页和迁移测试通过；前端构建及 7 项操作确认/幂等回归通过。浏览器验证了仅显示已验证备份、64px 行高、集群筛选、恢复演练、生产帮助、位置搜索、仓库取消/确认删除、清理对话框和深浅主题/窄屏展示。

项目全量类型检查的默认配置存在 `ignoreDeprecations` 与本机 TypeScript 版本不匹配；临时使用 `--ignoreDeprecations 5.0` 检查后，仍有项目既有的 71 项类型诊断，本次修改的数据库管理文件无类型诊断。此限制与前端打包通过分别记录。

本次已更新本地 API 与数据库 Worker 进程。开发环境重新启动时，在 `api/` 运行 `go run . -c config.yaml`；debug 模式默认同时运行数据库 Worker，无需再单独启动。需要分进程运行时，API 设置 `DATABASE_WORKER_MODE=external`，另行运行 `go run . -c config.yaml -mode database-worker`。MinIO 由 `.53` 的 systemd 持续运行，与这两个开发进程分离。
