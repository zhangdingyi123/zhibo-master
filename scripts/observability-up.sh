#!/usr/bin/env bash
# 一键启动监控可视化（Prometheus + Grafana + Nginx /monitor/）
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

PROD_FILE="docker-compose.prod.yml"

if docker compose version >/dev/null 2>&1; then
  COMPOSE="docker compose"
elif command -v docker-compose >/dev/null 2>&1; then
  COMPOSE="docker-compose"
else
  echo "错误: 未找到 docker compose，请安装 docker-compose-plugin 或 docker-compose" >&2
  exit 1
fi

if [[ ! -f "$PROD_FILE" ]]; then
  echo "错误: 缺少 $PROD_FILE" >&2
  exit 1
fi

if [[ ! -f deploy/grafana/dashboards/zhibo.json ]]; then
  echo "错误: 缺少 deploy/grafana/dashboards/zhibo.json" >&2
  exit 1
fi

IP="$(curl -sf ifconfig.me 2>/dev/null || true)"
IP="${IP:-47.97.176.185}"

echo "==> 启动 Prometheus / Grafana（--no-deps，不重建 mysql）"
# 旧容器名冲突时先清理监控容器
docker rm -f zhibo-prometheus zhibo-grafana 2>/dev/null || true
$COMPOSE -f "$PROD_FILE" up -d --no-deps prometheus grafana

echo "==> 重载 Nginx（启用 /monitor/ 反代）"
$COMPOSE -f "$PROD_FILE" up -d --no-deps nginx

echo "==> 等待服务就绪..."
for i in 1 2 3 4 5 6 7 8 9 10; do
  if curl -sf http://127.0.0.1/monitor/login >/dev/null 2>&1; then
    break
  fi
  sleep 2
done

echo ""
echo "==> 容器状态"
$COMPOSE -f "$PROD_FILE" ps prometheus grafana nginx

echo ""
echo "==> 大盘文件是否挂载进 Grafana"
docker exec zhibo-grafana ls -la /var/lib/grafana/dashboards/zhibo.json 2>/dev/null \
  || echo "警告: 大盘 JSON 未挂载，请 git pull 后重新运行本脚本"

echo ""
echo "==> Prometheus 抓取目标"
if curl -sf http://127.0.0.1:9090/api/v1/targets >/dev/null 2>&1; then
  curl -s http://127.0.0.1:9090/api/v1/targets \
    | python3 -c "import sys,json; d=json.load(sys.stdin); [print(t.get('labels',{}).get('job','?'), t.get('health','?')) for t in d.get('data',{}).get('activeTargets',[])]" \
    2>/dev/null || true
else
  echo "Prometheus 尚未就绪，稍后再查: curl http://127.0.0.1:9090/api/v1/targets"
fi

echo ""
echo "==> 业务指标"
curl -sf http://127.0.0.1/api/v1/metrics | python3 -m json.tool 2>/dev/null | head -20 \
  || echo "提示: /api/v1/metrics 未通，确认 backend 正常"

echo ""
echo "=========================================="
echo "  监控可视化（推荐，走 80 端口，免开安全组）"
echo "  http://${IP}/monitor/"
echo "  大盘直达: http://${IP}/monitor/d/zhibo-auction"
echo "  账号: admin / zhibo"
echo ""
echo "  原始 JSON 指标: http://${IP}/api/v1/metrics"
echo "  Prometheus（可选）: http://${IP}:9090"
echo "=========================================="
