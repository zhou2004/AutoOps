# ============ 构建阶段：使用 Node.js 22 构建前端 ============
FROM swr.cn-north-4.myhuaweicloud.com/ddn-k8s/docker.io/node:22.14.0-alpine AS builder

# 设置工作目录
WORKDIR /app/web

# 安装 pnpm（锁定 v10 以兼容 package.json 中的 pnpm 配置字段）
RUN corepack enable && corepack prepare pnpm@10 --activate

# 复制依赖声明文件（利用 Docker 缓存层）
COPY web/package.json web/pnpm-lock.yaml web/.npmrc ./

# 安装依赖
RUN pnpm install --frozen-lockfile

# 复制前端源码
COPY web/ ./

# 构建前端
RUN pnpm build

# ============ 运行阶段：nginx 提供服务 ============
FROM crpi-aj3vgoxp9kzh2jx4.cn-hangzhou.personal.cr.aliyuncs.com/zhangfan_k8s/nginx:latest

# 删除默认的 nginx 配置和静态文件
RUN rm -rf /usr/share/nginx/html/* && \
    rm -f /etc/nginx/conf.d/default.conf

# 复制自定义 nginx 配置
COPY docker/web/devops.conf /etc/nginx/conf.d/default.conf

# 从构建阶段复制编译好的静态文件
COPY --from=builder /app/web/dist /usr/share/nginx/html

# 暴露80端口
EXPOSE 80

# 启动 nginx
CMD ["nginx", "-g", "daemon off;"]
