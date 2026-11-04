ARG NODE_IMAGE=node:22-alpine
ARG NGINX_IMAGE=nginx:stable-alpine

FROM ${NODE_IMAGE} AS builder
WORKDIR /app/web
ENV COREPACK_ENABLE_DOWNLOAD_PROMPT=0
RUN corepack enable
COPY web/package.json web/pnpm-lock.yaml web/.npmrc ./
# packageManager in package.json pins pnpm, including its integrity hash.
RUN pnpm install --frozen-lockfile
COPY web/ ./
RUN pnpm build

FROM ${NGINX_IMAGE} AS runtime
RUN rm -rf /usr/share/nginx/html/* /etc/nginx/conf.d/default.conf
COPY docker/web/devops.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /app/web/dist /usr/share/nginx/html
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
