#!/bin/bash

# DevOps 服务启动脚本 v3.0
# 用法: ./devops-start.sh <version> <ip> <web_port> [api_port] [mysql_port] [redis_port]
# 示例: ./devops-start.sh v1.0 192.168.1.100 8080

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 默认值
API_PORT=${4:-8000}
MYSQL_PORT=${5:-3307}
REDIS_PORT=${6:-6379}
PROMETHEUS_PORT=9090
PUSHGATEWAY_PORT=9091

# 参数验证
if [ $# -lt 3 ]; then
    echo -e "${RED}错误: 参数不足${NC}"
    echo "用法: $0 <version> <ip> <web_port> [api_port] [mysql_port] [redis_port]"
    echo ""
    echo "参数说明:"
    echo "  version      - 镜像版本 (例如: v1.0, v2.0)"
    echo "  ip           - 服务器IP地址或域名 (例如: 192.168.1.100, devops.example.com)"
    echo "  web_port     - 前端访问端口 (例如: 8080, 8088)"
    echo "  api_port     - API后端端口 (可选, 默认: 8000)"
    echo "  mysql_port   - MySQL端口 (可选, 默认: 3307)"
    echo "  redis_port   - Redis端口 (可选, 默认: 6379)"
    echo ""
    echo "示例:"
    echo "  $0 v1.0 192.168.1.100 8080"
    echo "  $0 v1.0 192.168.1.100 8080 8000 3307 6379"
    exit 1
fi

VERSION=$1
SERVER_IP=$2
WEB_PORT=$3

# 验证IP地址格式
if ! [[ "$SERVER_IP" =~ ^[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}$ ]] && ! [[ "$SERVER_IP" =~ ^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$ ]]; then
    if [ "$SERVER_IP" != "localhost" ]; then
        echo -e "${RED}错误: IP地址或域名格式不正确${NC}"
        exit 1
    fi
fi

# 验证端口号
for port in $WEB_PORT $API_PORT $MYSQL_PORT $REDIS_PORT; do
    if ! [[ "$port" =~ ^[0-9]+$ ]] || [ "$port" -lt 1 ] || [ "$port" -gt 65535 ]; then
        echo -e "${RED}错误: 端口号无效: $port${NC}"
        exit 1
    fi
done

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}DevOps 服务启动脚本 v3.0${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo "配置信息:"
echo "  镜像版本:     $VERSION"
echo "  服务器IP:     $SERVER_IP"
echo "  前端端口:     $WEB_PORT"
echo "  API端口:      $API_PORT"
echo "  MySQL端口:    $MYSQL_PORT"
echo "  Redis端口:    $REDIS_PORT"
echo ""

# 获取脚本所在目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# 检测 docker-compose 命令
detect_docker_compose() {
    if command -v docker-compose &> /dev/null; then
        echo "docker-compose"
    elif docker compose version &> /dev/null 2>&1; then
        echo "docker compose"
    else
        echo ""
    fi
}

DOCKER_COMPOSE_CMD=$(detect_docker_compose)
if [ -z "$DOCKER_COMPOSE_CMD" ]; then
    echo -e "${RED}错误: 找不到 docker-compose 或 docker compose 命令${NC}"
    exit 1
fi

# 检查必要的文件
if [ ! -f "docker-compose.yml" ]; then
    echo -e "${RED}错误: 找不到 docker-compose.yml${NC}"
    exit 1
fi

if [ ! -f ".env" ]; then
    echo -e "${RED}错误: 找不到 .env 文件${NC}"
    exit 1
fi

if [ ! -f "api/config.yaml" ]; then
    echo -e "${RED}错误: 找不到 api/config.yaml${NC}"
    exit 1
fi

echo -e "${YELLOW}步骤 1: 更新 .env 文件...${NC}"
# 更新 .env 文件中的端口和IP配置
sed -i.bak "s|^WEB_PORT=.*|WEB_PORT=$WEB_PORT|" .env
sed -i.bak "s|^API_PORT=.*|API_PORT=$API_PORT|" .env
sed -i.bak "s|^MYSQL_PORT=.*|MYSQL_PORT=$MYSQL_PORT|" .env
sed -i.bak "s|^REDIS_PORT=.*|REDIS_PORT=$REDIS_PORT|" .env
sed -i.bak "s|^IMAGE_HOST=.*|IMAGE_HOST=http://$SERVER_IP:$WEB_PORT|" .env

echo -e "${GREEN}✓ .env 文件已更新${NC}"

echo -e "${YELLOW}步骤 2: 更新 api/config.yaml 文件...${NC}"
# 更新 config.yaml 中的IP地址配置
# 更新 prometheus URL
sed -i.bak "s|url: \"http://prometheus:9090\"|url: \"http://$SERVER_IP:$PROMETHEUS_PORT\"|" api/config.yaml
# 更新 pushgateway URL
sed -i.bak "s|url: \"http://pushgateway:9091\"|url: \"http://$SERVER_IP:$PUSHGATEWAY_PORT\"|" api/config.yaml
# 更新 heartbeat_server_url
sed -i.bak "s|heartbeat_server_url: \"http://devops-api:8000|heartbeat_server_url: \"http://$SERVER_IP:$API_PORT|" api/config.yaml
# 更新 installer_base_url
sed -i.bak "s|installer_base_url: \"http://devops-api:8000|installer_base_url: \"http://$SERVER_IP:$API_PORT|" api/config.yaml
# 更新 pushgateway_url
sed -i.bak "s|pushgateway_url: \"http://pushgateway:9091\"|pushgateway_url: \"http://$SERVER_IP:$PUSHGATEWAY_PORT\"|" api/config.yaml

echo -e "${GREEN}✓ api/config.yaml 文件已更新${NC}"

echo -e "${YELLOW}步骤 3: 更新 docker-compose.yml 中的镜像版本...${NC}"
# 更新镜像版本
sed -i.bak "s|deviops-api:v[0-9.]*|deviops-api:$VERSION|g" docker-compose.yml
sed -i.bak "s|deviops-web:v[0-9.]*|deviops-web:$VERSION|g" docker-compose.yml

echo -e "${GREEN}✓ docker-compose.yml 镜像版本已更新${NC}"

echo -e "${YELLOW}步骤 4: 停止现有服务...${NC}"
$DOCKER_COMPOSE_CMD down 2>/dev/null || true
echo -e "${GREEN}✓ 现有服务已停止${NC}"

echo -e "${YELLOW}步骤 5: 启动新服务...${NC}"
$DOCKER_COMPOSE_CMD up -d

# 等待服务启动
echo -e "${YELLOW}等待服务启动...${NC}"
sleep 10

# 检查服务状态
echo -e "${YELLOW}步骤 6: 检查服务状态...${NC}"
SERVICES=("devops-mysql" "devops-redis" "devops-pushgateway" "devops-prometheus" "devops-api" "devops-web")
ALL_HEALTHY=true

for service in "${SERVICES[@]}"; do
    if docker ps --filter "name=$service" --filter "status=running" | grep -q "$service"; then
        echo -e "${GREEN}✓ $service 运行中${NC}"
    else
        echo -e "${RED}✗ $service 未运行${NC}"
        ALL_HEALTHY=false
    fi
done

echo ""
echo -e "${GREEN}========================================${NC}"
if [ "$ALL_HEALTHY" = true ]; then
    echo -e "${GREEN}✓ 所有服务启动成功!${NC}"
    echo -e "${GREEN}========================================${NC}"
    echo ""
    echo "访问地址:"
    echo -e "  前端:       ${GREEN}http://$SERVER_IP:$WEB_PORT${NC}"
    echo -e "  API:        ${GREEN}http://$SERVER_IP:$API_PORT${NC}"
    echo -e "  Prometheus: ${GREEN}http://$SERVER_IP:$PROMETHEUS_PORT${NC}"
    echo -e "  Pushgateway:${GREEN}http://$SERVER_IP:$PUSHGATEWAY_PORT${NC}"
    echo ""
    echo "数据库连接:"
    echo -e "  MySQL:      ${GREEN}$SERVER_IP:$MYSQL_PORT${NC}"
    echo -e "  Redis:      ${GREEN}$SERVER_IP:$REDIS_PORT${NC}"
else
    echo -e "${RED}✗ 部分服务启动失败,请检查日志${NC}"
    echo -e "${RED}========================================${NC}"
    echo ""
    echo "查看日志:"
    echo "  $DOCKER_COMPOSE_CMD logs -f"
    exit 1
fi
