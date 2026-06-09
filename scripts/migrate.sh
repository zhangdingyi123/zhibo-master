#!/usr/bin/env bash
# 在 ECS 上执行未自动运行的 SQL 迁移（已有数据卷时 compose 不会重跑 initdb）
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.prod.yml}"

if docker compose version >/dev/null 2>&1; then
  COMPOSE="docker compose"
elif command -v docker-compose >/dev/null 2>&1; then
  COMPOSE="docker-compose"
else
  echo "错误: 未找到 docker compose" >&2
  exit 1
fi

mysql_exec() {
  $COMPOSE -f "$COMPOSE_FILE" exec -T mysql sh -c \
    'mysql -u"$MYSQL_USER" -p"$MYSQL_PASSWORD" "$MYSQL_DATABASE" '"$*"
}

run_file_if_needed() {
  local file="$1"
  local check_sql="$2"
  local name
  name="$(basename "$file")"

  if [[ ! -f "$file" ]]; then
    echo "跳过 $name（文件不存在）"
    return 0
  fi

  if mysql_exec -N -e "$check_sql" 2>/dev/null | grep -q 1; then
    echo "✓ $name 已应用，跳过"
    return 0
  fi

  echo "==> 应用 $name ..."
  $COMPOSE -f "$COMPOSE_FILE" exec -T mysql sh -c \
    'mysql -u"$MYSQL_USER" -p"$MYSQL_PASSWORD" "$MYSQL_DATABASE"' < "$file"
  echo "✓ $name 完成"
}

echo "==> 检查 MySQL 连接..."
if ! mysql_exec -e "SELECT 1" >/dev/null 2>&1; then
  echo "错误: 无法连接 MySQL，请先启动: $COMPOSE -f $COMPOSE_FILE up -d mysql" >&2
  exit 1
fi

run_file_if_needed \
  "backend/migrations/005_order_pay_expire.sql" \
  "SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='orders' AND COLUMN_NAME='pay_expire_at'"

run_file_if_needed \
  "backend/migrations/006_order_fulfillment.sql" \
  "SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='orders' AND COLUMN_NAME='receiver_name'"

run_file_if_needed \
  "backend/migrations/007_order_aftersale.sql" \
  "SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='orders' AND COLUMN_NAME='cancel_reason'"

echo ""
echo "==> 迁移校验"
mysql_exec -e "
  SELECT COLUMN_NAME, COLUMN_TYPE, COLUMN_COMMENT
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'orders'
    AND COLUMN_NAME IN ('pay_expire_at', 'receiver_name', 'shipped_at', 'completed_at', 'cancel_reason', 'refunded_at');
  SELECT id, order_no, status, pay_expire_at, receiver_name, shipped_at, completed_at, created_at
  FROM orders
  ORDER BY id DESC
  LIMIT 5;
"

echo ""
echo "✓ 迁移检查完成"
