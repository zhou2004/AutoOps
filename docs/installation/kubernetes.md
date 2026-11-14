# Kubernetes 安装

此教程提供一个 namespace 内的完整入门部署：MySQL、Redis、API、前端、数据库 worker 与 Web worker。它是单副本配置，适合验证和小规模部署；生产环境可替换为已有数据库及 Redis，并配置 HTTPS Ingress、备份和资源配额。

## 1. 准备

- 可用的 Kubernetes 集群及 `kubectl`（支持 `apply -k`）
- 默认 StorageClass；模板申请 MySQL 20Gi、Redis 2Gi、上传文件 5Gi
- 节点能够拉取 Docker Hub 镜像
- 本机安装 Git、OpenSSL

```bash
git clone --branch v1.1.0 https://github.com/zhou2004/AutoOps.git
cd AutoOps
kubectl get nodes
kubectl get storageclass
bash k8s/quickstart/setup.sh
```

配置生成在 `k8s/quickstart/generated/`，该目录不会提交 Git。查看 `app.env` 中的 `AUTOOPS_INITIAL_ADMIN_PASSWORD`，按需填写 SMTP。配置中数据库、Redis 和加密密钥均为本次安装生成。

## 2. 部署

```bash
kubectl apply -k k8s/quickstart
kubectl -n autoops get pods,pvc
kubectl -n autoops rollout status deployment/mysql --timeout=5m
kubectl -n autoops rollout status deployment/devops-api --timeout=10m
kubectl -n autoops rollout status deployment/devops-web --timeout=5m
```

初次启动期间 API/worker 可能会等待数据库或重试，迁移完成后应恢复正常。模板统一使用 `v1.1.0` 镜像，版本入口位于 `k8s/quickstart/kustomization.yaml`。

## 3. 打开页面

```bash
kubectl -n autoops port-forward service/devops-web 8080:80
```

在运行此命令的电脑上打开 **http://localhost:8080**，使用 `admin` 和生成的初始密码登录。长期对外访问请为 `devops-web:80` 配置 Ingress 和 TLS。

## 4. 配置与升级

```bash
# 修改 generated/app.env 后应用，Kustomize 会生成新的 Secret 名称并触发滚动更新
kubectl apply -k k8s/quickstart
kubectl -n autoops logs deployment/devops-api --tail=100
kubectl -n autoops logs deployment/web-worker --tail=100
```

升级前备份数据库、PVC 数据和 `generated/app.env`。修改 kustomization 的镜像版本后再次应用。已有 `k8s/api` 部署保持兼容；迁移到 quickstart 时须显式复用原数据、密钥和数据库连接，不能直接用空 PVC 替换。

## 常见问题

| 现象 | 检查方法 |
|---|---|
| PVC Pending | `kubectl -n autoops describe pvc`，确认默认 StorageClass 和容量 |
| ImagePullBackOff | 检查节点网络、仓库地址、镜像版本或 imagePullSecret |
| API CrashLoopBackOff | 查看 API 日志，确认 MySQL 初始化完成、密码及配置一致 |
| 验证码无法加载 | 检查 Redis Pod 与 Secret 密码；前端需通过 `devops-api` Service 访问 API |
| 邮件发送失败 | 检查 SMTP 授权码、465/TLS 或 587/STARTTLS，以及集群出口网络 |

删除 `autoops` namespace 会同时影响其中的数据卷，请先备份。
