#!/usr/bin/env bash
# 一键启动监控可视化（绕过 docker-compose 1.29 ContainerConfig bug，直接用 docker run）
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

if [[ ! -f deploy/grafana/dashboards/zhibo.json ]]; then
  echo "错误: 缺少 deploy/grafana/dashboards/zhibo.json，请先 git pull" >&2
  exit 1
fi

if ! docker inspect zhibo-backend >/dev/null 2>&1; then
  echo "错误: zhibo-backend 未运行，请先启动业务栈" >&2
  exit 1
fi

# 与 backend 同网，容器间可通过 zhibo-backend / zhibo-grafana 互访
NET="$(docker inspect zhibo-backend --format '{{range $k,$v := .NetworkSettings.Networks}}{{$k}}{{"\n"}}{{end}}' | head -1)"
if [[ -z "$NET" ]]; then
  echo "错误: 无法检测 Docker 网络" >&2
  exit 1
fi

# 从 .env 读取公网 IP / Grafana 子路径（可选）
GRAFANA_ROOT_URL="http://47.97.176.185/monitor/"
if [[ -f .env ]]; then
  # shellcheck disable=SC1091
  set -a
  source .env 2>/dev/null || true
  set +a
  if [[ -n "${GRAFANA_ROOT_URL:-}" ]]; then
    GRAFANA_ROOT_URL="$GRAFANA_ROOT_URL"
  elif [[ -n "${FRONTEND_URL:-}" ]]; then
    GRAFANA_ROOT_URL="${FRONTEND_URL%/}/monitor/"
  fi
fi

IP="$(curl -sf ifconfig.me 2>/dev/null || true)"
IP="${IP:-47.97.176.185}"

echo "==> Docker 网络: $NET"
echo "==> Grafana ROOT_URL: $GRAFANA_ROOT_URL"

docker volume create zhibo_prometheus_data >/dev/null 2>&1 || true
docker volume create zhibo_grafana_data >/dev/null 2>&1 || true

echo "==> 启动 Prometheus"
docker rm -f zhibo-prometheus 2>/dev/null || true
docker run -d --name zhibo-prometheus \
  --restart unless-stopped \
  --network "$NET" \
  -p 9090:9090 \
  -v "$ROOT/deploy/prometheus.yml:/etc/prometheus/prometheus.yml:ro" \
  -v "$ROOT/deploy/prometheus/alerts.yml:/etc/prometheus/alerts.yml:ro" \
  -v zhibo_prometheus_data:/prometheus \
  prom/prometheus:v2.54.1 \
  --config.file=/etc/prometheus/prometheus.yml \
  --storage.tsdb.path=/prometheus \
  --web.enable-lifecycle

echo "==> 启动 Grafana"
docker rm -f zhibo-grafana 2>/dev/null || true
docker run -d --name zhibo-grafana \
  --restart unless-stopped \
  --network "$NET" \
  -p 3000:3000 \
  -e GF_SECURITY_ADMIN_USER=admin \
  -e GF_SECURITY_ADMIN_PASSWORD=zhibo \
  -e GF_USERS_ALLOW_SIGN_UP=false \
  -e "GF_SERVER_ROOT_URL=${GRAFANA_ROOT_URL}" \
  -e GF_SERVER_SERVE_FROM_SUB_PATH=true \
  -v zhibo_grafana_data:/var/lib/grafana \
  -v "$ROOT/deploy/grafana/provisioning:/etc/grafana/provisioning:ro" \
  -v "$ROOT/deploy/grafana/dashboards:/var/lib/grafana/dashboards:ro" \
  grafana/grafana:11.2.0

echo "==> 重载 Nginx（/monitor/ 反代）"
if docker inspect zhibo-nginx >/dev/null 2>&1; then
  # 确保挂载最新 nginx.conf
  docker rm -f zhibo-nginx 2>/dev/null || true
fi
docker run -d --name zhibo-nginx \
  --restart unless-stopped \
  --network "$NET" \
  -p 80:80 \
  -v "$ROOT/deploy/nginx.conf:/etc/nginx/conf.d/default.conf:ro" \
  nginx:1.27-alpine

echo "==> 等待服务就绪..."
for _ in 1 2 3 4 5 6 7 8 9 10; do
  if curl -sf http://127.0.0.1/monitor/login >/dev/null 2>&1; then
    break
  fi
  sleep 2
done

echo ""
echo "==> 容器状态"
docker ps --format 'table {{.Names}}\t{{.Status}}\t{{.Ports}}' | grep -E 'NAMES|zhibo-'

echo ""
echo "==> 大盘文件挂载"
docker exec zhibo-grafana ls -la /var/lib/grafana/dashboards/zhibo.json

echo ""
echo "==> Prometheus 抓取"
sleep 3
curl -s http://127.0.0.1:9090/api/v1/targets \
  | python3 -c "import sys,json; d=json.load(sys.stdin); [print(t.get('labels',{}).get('job','?'), t.get('health','?')) for t in d.get('data',{}).get('activeTargets',[])]" \
  2>/dev/null || echo "（Prometheus 启动中，稍后再查）"

echo ""
echo "=========================================="
echo "  监控可视化（80 端口，免开 3000）"
echo "  http://${IP}/monitor/"
echo "  大盘: http://${IP}/monitor/d/zhibo-auction"
echo "  账号: admin / zhibo"
echo ""
echo "  JSON 指标: http://${IP}/api/v1/metrics"
echo "=========================================="
