# Docker Compose 安装

适合首次体验和单机部署。包含 MySQL、Redis、API、前端以及独立的数据库/Web 任务执行器。

## 1. 下载并生成本地配置

依赖：Docker Engine、Docker Compose v2、Git、OpenSSL。

```bash
git clone --branch v1.1.0 https://github.com/zhou2004/AutoOps.git
cd AutoOps
bash docker/setup.sh
```

脚本生成 `docker/.env` 和 `docker/api/config.yaml`，为数据库、Redis、管理员和加密密钥生成随机值；重复运行会保留已有文件。

打开 `docker/.env` 查看 `AUTOOPS_INITIAL_ADMIN_PASSWORD`。需要邮件注册与找回密码时，填写其中的 `AUTOOPS_SMTP_*` 字段，见 [邮箱配置](../account-access.md)。默认访问端口为 `8080`，可修改 `WEB_PORT`。

## 2. 启动

```bash
cd docker
docker compose config --quiet
docker compose pull
docker compose up -d
docker compose ps
```

首次启动会导入干净的初始化结构并完成迁移，通常需要等待 MySQL 就绪。然后打开 **http://服务器IP:8080**，使用 `admin` 和生成的初始密码登录。

默认使用公共 Docker Hub 镜像：`feelzhou/autoops-api:v1.1.0`、`feelzhou/autoops-web:v1.1.0`。同版镜像也发布到 GHCR；可将 `AUTOOPS_IMAGE_REGISTRY` 改为 `ghcr.io/zhou2004`。若 GHCR 包要求登录，可继续使用 Docker Hub，或用具有 `read:packages` 权限的凭据执行 `docker login ghcr.io`。

## 3. 常用命令

```bash
docker compose logs --tail=100 devops-api
docker compose logs --tail=100 mysql
# 更新邮箱配置后重建 API 容器
docker compose up -d devops-api
# 停止服务，保留数据卷
docker compose down
# 从源码构建，替代下载镜像
docker compose build devops-api devops-web
docker compose up -d
```

数据库、Redis、上传文件和任务目录使用命名卷。MySQL 和 Redis 不默认暴露宿主机端口；浏览器通过前端 Nginx 访问 API。

## 数据备份与升级

备份数据库、上传卷、`docker/.env` 和加密密钥。升级时修改 `.env` 中的 `AUTOOPS_IMAGE_TAG`，执行 `docker compose pull && docker compose up -d`，再检查 API 日志及登录页。

**已有部署请保留原卷和配置。** 本版本的 Compose 面向全新安装，旧版曾使用 `./mysql/data`、`./redis/data` 等宿主机目录。升级前将新 Compose 的数据卷映射改回原目录，并沿用原数据库名、用户、密码和 Web/AI 加密密钥；不要直接用空的新命名卷替换旧数据。`AUTOOPS_INITIAL_ADMIN_PASSWORD` 只修改未初始化的默认账号，不会重置已修改密码的用户。

`docker compose down -v` 会删除命名卷，不能用于日常停止或升级。
