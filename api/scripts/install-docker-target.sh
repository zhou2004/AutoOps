#!/bin/bash
#
# 在目标机器上安装 Docker（供 AutoOps Docker 管理 / 跨机迁移复制使用）
#
# 用法（在能访问目标机器的机器上执行）：
#   ssh root@192.168.0.52 'bash -s' < install-docker-target.sh
# 或直接登录目标机器后执行：
#   bash install-docker-target.sh
#
set -euo pipefail

log() { echo "[$(date '+%F %T')] $*"; }

if command -v docker >/dev/null 2>&1; then
  log "Docker 已安装: $(docker --version 2>/dev/null || true)"
  exit 0
fi

if [ "$(id -u)" -ne 0 ]; then
  echo "请以 root 运行" >&2
  exit 1
fi

detect_os() {
  if [ -f /etc/os-release ]; then
    . /etc/os-release
    echo "$ID"
  else
    echo "unknown"
  fi
}

OS_ID=$(detect_os)
log "检测到系统: ${OS_ID}"

install_deps() {
  case "$OS_ID" in
    centos|rhel|rocky|almalinux|fedora|anolis|openEuler)
      yum install -y yum-utils curl >/dev/null 2>&1 || dnf install -y dnf-plugins-core curl >/dev/null 2>&1 || true
      ;;
    ubuntu|debian)
      export DEBIAN_FRONTEND=noninteractive
      apt-get update >/dev/null 2>&1
      apt-get install -y ca-certificates curl gnupg lsb-release >/dev/null 2>&1 || true
      ;;
    *)
      log "未知发行版，尝试继续"
      ;;
  esac
}

install_docker() {
  case "$OS_ID" in
    centos|rhel|rocky|almalinux|fedora|anolis|openEuler)
      yum-config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo >/dev/null 2>&1 \
        || dnf config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo >/dev/null 2>&1 || true
      (yum install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin || \
        dnf install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin) >/dev/null 2>&1
      ;;
    ubuntu|debian)
      install -m 0755 -d /etc/apt/keyrings
      curl -fsSL https://download.docker.com/linux/${OS_ID}/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg >/dev/null 2>&1 || true
      echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/${OS_ID} $(lsb_release -cs) stable" \
        > /etc/apt/sources.list.d/docker.list
      apt-get update >/dev/null 2>&1
      apt-get install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin >/dev/null 2>&1
      ;;
    *)
      log "请手动安装 Docker"
      exit 1
      ;;
  esac
}

install_deps
log "开始安装 Docker..."
install_docker

log "启动并设置开机自启..."
systemctl enable --now docker >/dev/null 2>&1 || service docker start >/dev/null 2>&1 || true

log "验证：$(docker --version 2>/dev/null || true)"
log "验证：$(docker compose version 2>/dev/null || docker-compose --version 2>/dev/null || echo 'compose 插件未检测到')"
log "Docker 安装完成"
