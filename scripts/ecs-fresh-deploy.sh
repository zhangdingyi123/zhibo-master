#!/usr/bin/env bash
# ECS 全量清空并重新部署（删除所有 zhibo 容器 + 数据卷，数据库会重置）
# 用法: bash scripts/ecs-fresh-deploy.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

COMPOSE_FILE="docker-compose.prod.yml"

if docker compose version >/dev/null 2>&1; then
  COMPOSE="docker compose"
elif command -v docker-compose >/dev/null 2>&1; then
  COMPOSE="docker-compose"
else
  echo "错误: 未找到 docker compose" >&2
  exit 1
fi

echo "⚠️  将删除所有 zhibo 容器及 MySQL/Redis 数据卷，数据库会清空！"
echo "    10 秒内 Ctrl+C 取消..."
sleep 10

echo ""
echo "==> 1/6 停止并删除 zhibo 容器"
docker ps -a --format '{{.Names}}' | grep '^zhibo-' | xargs -r docker rm -f 2>/dev/null || true
$COMPOSE -f "$COMPOSE_FILE" down --remove-orphans -v 2>/dev/null || true

echo "==> 2/6 清理残留数据卷"
docker volume ls --format '{{.Name}}' | grep -E '(mysql_data|redis_data|prometheus_data|grafana_data)$' | xargs -r docker volume rm 2>/dev/null || true

echo "==> 3/6 检查 .env"
if [[ ! -f .env ]]; then
  cp .env.example .env
  echo "    已从 .env.example 复制，请按需修改后重新运行"
fi

# Docker 内必须用服务名 mysql / redis，不能用 localhost
if grep -q '@tcp(localhost:3306)' .env 2>/dev/null; then
  sed -i 's|@tcp(localhost:3306)|@tcp(mysql:3306)|g' .env
  sed -i 's|@tcp(127.0.0.1:3306)|@tcp(mysql:3306)|g' .env
  echo "    已自动修正 MYSQL_DSN: localhost → mysql"
fi
if grep -q '^REDIS_ADDR=localhost:6379' .env 2>/dev/null; then
  sed -i 's|^REDIS_ADDR=localhost:6379|REDIS_ADDR=redis:6379|' .env
  echo "    已自动修正 REDIS_ADDR: localhost → redis"
fi

grep -E '^(MYSQL_DSN|REDIS_ADDR)=' .env || true

echo "==> 4/6 构建并启动全栈"
if $COMPOSE -f "$COMPOSE_FILE" up -d --build 2>&1; then
  echo "    compose up 成功"
else
  echo "    compose up 失败（可能是 1.29 ContainerConfig bug），改用手动部署..."
  bash scripts/manual-deploy.sh
fi

echo "==> 5/6 等待 MySQL 就绪"
for i in $(seq 1 40); do
  if $COMPOSE -f "$COMPOSE_FILE" exec -T mysql \
    mysql -uzhibo -pzhibo zhibo -e "SELECT 1" 2>/dev/null; then
    break
  fi
  echo "    等待 MySQL... ($i/40)"
  sleep 3
done

echo "==> 6/6 数据库迁移 + 健康检查"
bash scripts/migrate.sh || true

sleep 5
echo ""
$COMPOSE -f "$COMPOSE_FILE" ps 2>/dev/null || docker ps | grep zhibo

echo ""
if curl -sf http://127.0.0.1/api/v1/health; then
  echo ""
  IP="$(curl -sf ifconfig.me 2>/dev/null || echo '47.97.176.185')"
  echo "✓ 全量部署成功"
  echo "  用户端: http://${IP}/app"
  echo "  主播端: http://${IP}/admin"
  echo "  演示账号: 13800000001 / 123456（主播）"
else
  echo ""
  echo "✗ API 未响应，查看日志:" >&2
  echo "  docker logs --tail 50 zhibo-backend" >&2
  exit 1
fi
