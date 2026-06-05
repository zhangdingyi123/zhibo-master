#!/usr/bin/env bash
# 无 docker compose 时的手动部署（MySQL/Redis 需已存在或由 compose 启动）
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

ENV_FILE="${ROOT}/.env"
if [[ ! -f "$ENV_FILE" ]]; then
  echo "错误: 缺少 $ENV_FILE" >&2
  exit 1
fi

# 优先用 compose 创建的网络，否则新建
NET="$(docker network ls --format '{{.Name}}' | grep -E 'zhibo.*default' | head -1 || true)"
if [[ -z "$NET" ]]; then
  NET="zhibo_default"
  docker network create "$NET" 2>/dev/null || true
fi
echo "==> Docker 网络: $NET"

# 启动 MySQL / Redis（compose 可用时）
if docker compose version >/dev/null 2>&1; then
  docker compose -f docker-compose.prod.yml up -d mysql redis
elif command -v docker-compose >/dev/null 2>&1; then
  docker-compose -f docker-compose.prod.yml up -d mysql redis
else
  echo "==> 跳过 compose，假定 mysql/redis 容器已在运行"
fi

echo "==> 构建 backend / frontend 镜像..."
docker build -t zhibo-backend ./backend
docker build -t zhibo-frontend ./frontend

echo "==> 启动 backend / frontend..."
docker rm -f zhibo-backend zhibo-frontend 2>/dev/null || true

docker run -d --name zhibo-backend \
  --network "$NET" \
  --env-file "$ENV_FILE" \
  -e PORT=8081 \
  --restart unless-stopped \
  zhibo-backend

docker run -d --name zhibo-frontend \
  --network "$NET" \
  --restart unless-stopped \
  zhibo-frontend

# nginx：不存在则创建，存在则确保在同一网络并重启
if docker ps -a --format '{{.Names}}' | grep -qx zhibo-nginx; then
  docker network connect "$NET" zhibo-nginx 2>/dev/null || true
  docker restart zhibo-nginx
else
  docker run -d --name zhibo-nginx \
    --network "$NET" \
    -p 80:80 \
    -v "${ROOT}/deploy/nginx.conf:/etc/nginx/conf.d/default.conf:ro" \
    --restart unless-stopped \
    nginx:1.27-alpine
fi

echo "==> 等待 backend 启动..."
sleep 3

echo "==> 容器状态"
docker ps --format 'table {{.Names}}\t{{.Status}}\t{{.Ports}}' | grep -E 'NAMES|zhibo' || docker ps

echo ""
echo "==> 健康检查"
if curl -sf http://127.0.0.1/api/v1/health; then
  echo ""
  echo "✓ 部署成功"
else
  echo ""
  echo "✗ API 未响应，查看 backend 日志:" >&2
  echo "  docker logs --tail 80 zhibo-backend" >&2
  exit 1
fi
