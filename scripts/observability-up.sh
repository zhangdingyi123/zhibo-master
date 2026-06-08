#!/usr/bin/env bash
# 启动 Prometheus + Grafana（与生产栈同网）
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

PROD_FILE="docker-compose.prod.yml"
OBS_FILE="docker-compose.observability.yml"

if docker compose version >/dev/null 2>&1; then
  COMPOSE="docker compose"
elif command -v docker-compose >/dev/null 2>&1; then
  COMPOSE="docker-compose"
else
  echo "错误: 未找到 docker compose，请安装：" >&2
  echo "  apt install -y docker-compose-plugin   # Compose V2 插件" >&2
  echo "  或: apt install -y docker-compose       # 独立二进制（V1）" >&2
  exit 1
fi

if [[ ! -f "$PROD_FILE" || ! -f "$OBS_FILE" ]]; then
  echo "错误: 缺少 $PROD_FILE 或 $OBS_FILE" >&2
  exit 1
fi

echo "==> 使用: $COMPOSE -f $PROD_FILE -f $OBS_FILE"
$COMPOSE -f "$PROD_FILE" -f "$OBS_FILE" up -d prometheus grafana

echo "==> 等待 Prometheus 就绪..."
sleep 3

echo "==> 容器状态"
$COMPOSE -f "$PROD_FILE" -f "$OBS_FILE" ps prometheus grafana

echo ""
echo "==> 指标检查"
if curl -sf http://127.0.0.1/metrics | head -5; then
  echo "..."
  echo "✓ /metrics 可访问（经 Nginx）"
else
  echo "提示: /metrics 未通，确认已拉取最新代码并重载 nginx：" >&2
  echo "  $COMPOSE -f $PROD_FILE up -d --build nginx backend" >&2
fi

IP="$(curl -sf ifconfig.me 2>/dev/null || true)"
IP="${IP:-<ECS公网IP>}"
echo ""
echo "Grafana:    http://${IP}:3000  (admin / zhibo)"
echo "Prometheus: http://${IP}:9090"
echo "请在安全组放行 TCP 3000、9090（或仅用 SSH 隧道访问）"
