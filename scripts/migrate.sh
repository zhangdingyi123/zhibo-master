#!/usr/bin/env bash
# 在 ECS 上执行未自动运行的 SQL 迁移（已有数据卷时 compose 不会重跑 initdb）
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.prod.yml}"
MYSQL_CONTAINER="${MYSQL_CONTAINER:-zhibo-mysql}"

# 勿 source 整个 .env：MYSQL_DSN 含 tcp(...) 括号会导致 bash 语法错误
env_var() {
  local key="$1" default="$2"
  if [[ -f .env ]]; then
    local line val
    line="$(grep -E "^${key}=" .env | tail -1 || true)"
    if [[ -n "$line" ]]; then
      val="${line#*=}"
      val="${val%\"}"; val="${val#\"}"
      val="${val%\'}"; val="${val#\'}"
      if [[ -n "$val" ]]; then
        echo "$val"
        return
      fi
    fi
  fi
  echo "$default"
}

MYSQL_USER="$(env_var MYSQL_USER zhibo)"
MYSQL_PASSWORD="$(env_var MYSQL_PASSWORD zhibo)"
MYSQL_DATABASE="$(env_var MYSQL_DATABASE zhibo)"

if docker compose version >/dev/null 2>&1; then
  COMPOSE="docker compose"
elif command -v docker-compose >/dev/null 2>&1; then
  COMPOSE="docker-compose"
else
  COMPOSE=""
fi

mysql_exec() {
  if docker inspect -f '{{.State.Running}}' "$MYSQL_CONTAINER" 2>/dev/null | grep -q true; then
    docker exec -i "$MYSQL_CONTAINER" env MYSQL_PWD="$MYSQL_PASSWORD" \
      mysql -u"$MYSQL_USER" "$MYSQL_DATABASE" "$@"
    return
  fi
  if [[ -n "$COMPOSE" ]]; then
    $COMPOSE -f "$COMPOSE_FILE" exec -T mysql sh -c \
      'mysql -u"$MYSQL_USER" -p"$MYSQL_PASSWORD" "$MYSQL_DATABASE" '"$*"
    return
  fi
  echo "错误: MySQL 容器 $MYSQL_CONTAINER 未运行，且未找到 docker compose" >&2
  return 1
}

mysql_apply_file() {
  local file="$1"
  if docker inspect -f '{{.State.Running}}' "$MYSQL_CONTAINER" 2>/dev/null | grep -q true; then
    docker exec -i "$MYSQL_CONTAINER" env MYSQL_PWD="$MYSQL_PASSWORD" \
      mysql -u"$MYSQL_USER" "$MYSQL_DATABASE" < "$file"
    return
  fi
  $COMPOSE -f "$COMPOSE_FILE" exec -T mysql sh -c \
    'mysql -u"$MYSQL_USER" -p"$MYSQL_PASSWORD" "$MYSQL_DATABASE"' < "$file"
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
  mysql_apply_file "$file"
  echo "✓ $name 完成"
}

echo "==> 检查 MySQL 连接（容器: $MYSQL_CONTAINER）..."
if ! err="$(mysql_exec -e "SELECT 1" 2>&1)"; then
  echo "错误: 无法连接 MySQL" >&2
  echo "  详情: $err" >&2
  if [[ -n "$COMPOSE" ]]; then
    echo "  请先启动: $COMPOSE -f $COMPOSE_FILE up -d mysql" >&2
  else
    echo "  请先启动 MySQL 容器: $MYSQL_CONTAINER" >&2
  fi
  echo "  可手动测试: docker exec $MYSQL_CONTAINER mysql -u$MYSQL_USER -p zhibo -e \"SELECT 1\"" >&2
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
