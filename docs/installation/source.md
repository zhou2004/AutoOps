# 源码安装

适合本地开发、调试和二次开发。以下步骤以 Linux / WSL 为例。

## 1. 准备依赖

- Go 1.25（以 `api/go.mod` 为准）
- Node.js 22、Corepack；pnpm 版本由 `web/package.json` 锁定
- MySQL 8.0、Redis 7

```bash
git clone https://github.com/zhou2004/AutoOps.git
cd AutoOps
corepack enable
```

## 2. 初始化数据库和配置

在 MySQL 中创建一个**空数据库**和专用用户（请替换下面的密码）。不要把初始化 SQL 导入已有业务库。

```sql
CREATE DATABASE autoops CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE USER 'autoops'@'%' IDENTIFIED BY 'YOUR_DATABASE_PASSWORD';
GRANT ALL PRIVILEGES ON autoops.* TO 'autoops'@'%';
```

```bash
mysql -h 127.0.0.1 -u autoops -p autoops < api/sql/bootstrap.sql
cp api/config.example.yaml api/config.yaml
cp web/.env.development.example web/.env.development
```

修改 `api/config.yaml` 中的 `db` 和 `redis` 地址、用户及密码。默认前端通过 `http://127.0.0.1:8000` 访问 API；从另一台电脑访问开发服务器时，将 `web/.env.development` 的 `VITE_API_BASE_URL` 改为浏览器能访问的 API 地址。

本地配置已被 Git 忽略。请保留并备份，不要覆盖成示例文件。

## 3. 启动 API

```bash
cd api
go mod download
# 首次启动时设置管理员密码（10–72 字节），输入不会回显。
read -rs -p 'Initial admin password: ' AUTOOPS_INITIAL_ADMIN_PASSWORD
export AUTOOPS_INITIAL_ADMIN_PASSWORD
export AUTOOPS_JWT_SECRET="$(openssl rand -hex 32)"
go run . -c config.yaml
```

服务默认监听 `8000`。程序自动补齐模块表结构，并仅对未修改的初始管理员应用初始密码。管理员用户名为 `admin`。生产环境需保存固定的 `AUTOOPS_JWT_SECRET`，多个 API 实例必须一致。

## 4. 启动前端

在另一个终端执行：

```bash
cd AutoOps/web
pnpm install --frozen-lockfile
pnpm dev
```

打开 **http://localhost:8080**，使用 `admin` 和第 3 步设置的密码登录。首次登录后补充自己的邮箱，以便使用密码找回。

## 5. 邮箱、任务及生产运行

- [邮箱注册与密码找回](../account-access.md)：SMTP、审批流程和验证码规则。
- 开发模式内置数据库/Web 任务执行器。生产模式请运行独立的 `-mode database-worker` 和 `-mode web-worker` 进程，使用相同数据库和加密密钥。
- SSH、数据库部署、Web 发布等功能需要相应远端权限；Ansible 及其版本隔离参考 [Dockerfile](../../docker/api/Dockerfile)，新用户使用 Docker 安装更省事。

```bash
# 编译后端（从 api 目录执行）
go build -o autoops .
# 构建前端（从 web 目录执行）
pnpm build
```

前端构建结果位于 `web/dist`，生产 Nginx 配置参考 [devops.conf](../../docker/web/devops.conf)。外部访问建议在网关配置 HTTPS。

## 验证与排错

```bash
curl -fsS http://127.0.0.1:8000/api/v1/auth/options
```

应返回 `code: 200`。登录页无法获取验证码时，检查浏览器能否访问 API，以及 Redis 地址和密码是否正确。数据库连接失败时先核对 `api/config.yaml`，不要重复导入初始化 SQL。
