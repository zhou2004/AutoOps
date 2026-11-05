# GitHub Actions 构建与发布 AutoOps 镜像

工作流：`.github/workflows/docker-publish.yml`。后端发布到公共仓库 `feelzhou/autoops-api`，前端发布到公共仓库 `feelzhou/autoops-web`，分别使用 `docker/api/Dockerfile` 和 `docker/web/Dockerfile` 构建。数据库 Worker 复用 API 镜像。

## 首次配置

1. 在 Docker Hub 确认 `feelzhou/autoops-api`、`feelzhou/autoops-web` 两个仓库存在且可见性均为 **Public**。
2. 在 Docker Hub 创建有上述两个仓库 Read & Write 权限的访问 Token；已经配置且覆盖两个仓库的 `DOCKERHUB_TOKEN` 可以继续使用。
3. 打开 [GitHub Actions Secrets](https://github.com/zhou2004/AutoOps/settings/secrets/actions)，新增 Repository Secret：`DOCKERHUB_TOKEN`。值填写上述 Token；不要将 Token 写入代码、配置文件或工作流。
4. 在仓库 Actions 页面确认 GitHub Actions 已启用。Docker Hub 用户名固定为 `feelzhou`，不需要额外配置用户名 Secret。
5. 推送到 `main`，或打开 [Build and publish Docker images](https://github.com/zhou2004/AutoOps/actions/workflows/docker-publish.yml)，选择 `main` 后点击 **Run workflow**。缺少 Token 时构建任务会给出明确错误；配置 Secret 后可以重新运行失败任务。

## 触发与标签

每次 push 到 `main` 都触发，不做文件路径过滤。推送 `v*` Git 标签以及手动运行也会触发。一次 push 包含多个 Git 提交时，构建该次 push 的最终提交。

| 场景                       | `feelzhou/autoops-api` 标签 | `feelzhou/autoops-web` 标签 |
| -------------------------- | --------------------------- | --------------------------- |
| 每次构建                   | `sha-<前12位提交SHA>`       | `sha-<前12位提交SHA>`       |
| 当前 main 构建成功         | `latest`                    | `latest`                    |
| 推送 Git 标签 `v1.0.0`     | `v1.0.0`                    | `v1.0.0`                    |
| 手动运行，tag 填 `test-01` | `test-01`                   | `test-01`                   |

手动 tag 填写标签本身即可，不填写仓库名。允许 1–100 位字母、数字、下划线、点和连字符，首位不能为点或连字符。

工作流先执行数据库后端测试、前端国际化与操作确认测试、标签生成测试；通过后并行构建两个 Dockerfile 并推送提交标签。两类镜像均成功后才更新两个 `latest` 标签。更新前重新检查 main：旧提交晚于新提交构建完成时，不覆盖新提交的 latest。构建结果和镜像摘要在 Actions Summary 中查看。

目前发布 `linux/amd64` 镜像。基础镜像使用公开的 Node、Nginx、Go 和 Python；API 使用多阶段构建，保留普通 Ansible、kubeasz 与数据库自动化的独立 Python 环境。数据库控制器使用 Python 3.12，兼容固定的 Ansible Core 2.17。

## 使用已发布镜像

```bash
docker pull feelzhou/autoops-api:latest
docker pull feelzhou/autoops-web:latest
```

`docker/docker-compose.yml` 已使用这两个镜像，并为 API、Web 保留本地 build 配置。配置好原有 MySQL、Redis、API 运行参数后，从仓库根目录执行：

```bash
docker compose -f docker/docker-compose.yml pull
docker compose -f docker/docker-compose.yml up -d --no-build
```

Kubernetes 的 API、Web、数据库 Worker 清单也使用对应的新镜像地址；工作流不会自动更新运行中的服务。

按提交部署可以确保 API、Web 与 Worker 使用同一次构建：

```bash
export AUTOOPS_IMAGE_TAG=sha-0123456789ab
docker compose -f docker/docker-compose.yml pull
docker compose -f docker/docker-compose.yml up -d --no-build
```

按版本部署则设置 `AUTOOPS_IMAGE_TAG=v1.0.0`。需要本地构建时，从仓库根目录执行：

```bash
docker build -f docker/api/Dockerfile -t feelzhou/autoops-api:local .
docker build -f docker/web/Dockerfile -t feelzhou/autoops-web:local .
```

构建上下文必须为仓库根目录 `.`，不是 `docker/api` 或 `docker/web`。根目录 `.dockerignore` 排除 API 运行配置、密钥、日志、实际上传文件、本地开发环境配置与依赖缓存；镜像运行时仍通过 Compose 挂载 `docker/api/config.yaml` 和需要的密钥。前端使用 `.env.production`，API 地址由 Nginx `/api/` 代理到 `devops-api:8000`。

该流程只构建和发布镜像，不自动替换现有服务器上的容器。GitHub Actions 使用 Docker 官方 login/build-push/buildx Actions；认证采用 Docker Hub Token，参见 [Docker 官方文档](https://docs.docker.com/build/ci/github-actions/)。
