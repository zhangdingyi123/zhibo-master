#!/usr/bin/env bash
# 核心业务栈：mysql redis backend frontend nginx（docker run，绕过 compose ContainerConfig）
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

# 默认密码（与 docker-compose.prod.yml 一致）；生产以 .env 为准
MYSQL_ROOT_PASSWORD="${MYSQL_ROOT_PASSWORD:-root}"
MYSQL_DATABASE="${MYSQL_DATABASE:-zhibo}"
MYSQL_USER="${MYSQL_USER:-zhibo}"
MYSQL_PASSWORD="${MYSQL_PASSWORD:-zhibo}"
if [[ -f .env ]]; then
  set -a
  # shellcheck disable=SC1091
  source .env
  set +a
fi

detect_net() {
  for c in zhibo-backend zhibo-redis zhibo-mysql; do
    if docker inspect "$c" >/dev/null 2>&1; then
      docker inspect "$c" --format '{{range $k,$v := .NetworkSettings.Networks}}{{$k}}{{"\n"}}{{end}}' | head -1
      return
    fi
  done
  echo "zhibo_default"
}

NET="$(detect_net)"
echo "==> Docker 网络: $NET"
docker network inspect "$NET" >/dev/null 2>&1 || docker network create "$NET"

mysql_vol="$(docker volume ls --format '{{.Name}}' | grep -E 'mysql_data$' | head -1)"
mysql_vol="${mysql_vol:-zhibo_mysql_data}"
redis_vol="$(docker volume ls --format '{{.Name}}' | grep -E 'redis_data$' | head -1)"
redis_vol="${redis_vol:-zhibo_redis_data}"
docker volume create "$mysql_vol" >/dev/null
docker volume create "$redis_vol" >/dev/null
echo "==> 数据卷: mysql=$mysql_vol redis=$redis_vol"

if ! docker inspect zhibo-mysql >/dev/null 2>&1; then
  echo "==> 创建 MySQL 容器"
  docker run -d --name zhibo-mysql \
    --restart unless-stopped \
    --network "$NET" \
    -e "MYSQL_ROOT_PASSWORD=${MYSQL_ROOT_PASSWORD}" \
    -e "MYSQL_DATABASE=${MYSQL_DATABASE}" \
    -e "MYSQL_USER=${MYSQL_USER}" \
    -e "MYSQL_PASSWORD=${MYSQL_PASSWORD}" \
    -v "${mysql_vol}:/var/lib/mysql" \
    -v "$ROOT/backend/migrations:/docker-entrypoint-initdb.d:ro" \
    --health-cmd="mysqladmin ping -h localhost" \
    --health-interval=5s \
    --health-timeout=5s \
    --health-retries=20 \
    mysql:8.0
else
  echo "==> 启动 MySQL"
  docker start zhibo-mysql
fi

if ! docker inspect zhibo-redis >/dev/null 2>&1; then
  echo "==> 创建 Redis 容器"
  docker run -d --name zhibo-redis \
    --restart unless-stopped \
    --network "$NET" \
    -v "${redis_vol}:/data" \
    --health-cmd="redis-cli ping" \
    --health-interval=5s \
    --health-timeout=3s \
    --health-retries=20 \
    redis:7-alpine
else
  echo "==> 启动 Redis"
  docker start zhibo-redis
fi

echo "==> 等待 MySQL / Redis..."
for _ in $(seq 1 40); do
  mysql_ok=$(docker inspect zhibo-mysql --format '{{.State.Health.Status}}' 2>/dev/null || echo "unknown")
  redis_ok=$(docker inspect zhibo-redis --format '{{.State.Health.Status}}' 2>/dev/null || echo "unknown")
  if [[ "$mysql_ok" == "healthy" && "$redis_ok" == "healthy" ]]; then
    break
  fi
  sleep 2
done

echo "==> 启动 Backend / Frontend"
if ! docker inspect zhibo-backend >/dev/null 2>&1; then
  echo "错误: zhibo-backend 容器不存在，需重新 build 镜像" >&2
  exit 1
fi
docker start zhibo-backend zhibo-frontend

# 确保 backend 在正确网络（若曾离线重建 mysql）
docker network connect "$NET" zhibo-backend 2>/dev/null || true
docker network connect "$NET" zhibo-frontend 2>/dev/null || true

echo "==> 启动 Nginx（80）"
docker rm -f zhibo-nginx 2>/dev/null || true
docker run -d --name zhibo-nginx \
  --restart unless-stopped \
  --network "$NET" \
  -p 80:80 \
  -v "$ROOT/deploy/nginx.conf:/etc/nginx/conf.d/default.conf:ro" \
  nginx:1.27-alpine

echo "==> 等待 API..."
docker restart zhibo-backend 2>/dev/null || true
for _ in $(seq 1 20); do
  if curl -sf http://127.0.0.1/api/v1/health >/dev/null 2>&1; then
    break
  fi
  sleep 2
done

echo ""
docker ps --format 'table {{.Names}}\t{{.Status}}\t{{.Ports}}' | grep -E 'NAMES|zhibo-'

echo ""
if curl -sf http://127.0.0.1/api/v1/health; then
  echo ""
  IP="$(curl -sf ifconfig.me 2>/dev/null || echo '47.97.176.185')"
  echo "✓ 业务栈已就绪"
  echo "  用户端: http://${IP}/app"
  echo "  主播端: http://${IP}/admin"
else
  echo "✗ API 未响应" >&2
  echo "  docker logs --tail 40 zhibo-backend" >&2
  echo "  docker logs --tail 20 zhibo-mysql" >&2
  exit 1
fi
