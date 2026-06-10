#!/usr/bin/env bash
# 阿里云 ECS 一键重部署（保留 MySQL/Redis 数据卷）
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

# 由 ecs-update.sh 调用时已 pull，可 SKIP_GIT_PULL=1 跳过
if [[ -d .git ]] && [[ "${SKIP_GIT_PULL:-}" != "1" ]]; then
  echo "==> 拉取最新代码 (git pull)..."
  for attempt in 1 2 3; do
    if git -c http.version=HTTP/1.1 pull --ff-only; then
      break
    fi
    if [[ "$attempt" -eq 3 ]]; then
      echo "错误: git pull 失败，请检查网络或改用 SSH remote" >&2
      exit 1
    fi
    echo "    重试 ${attempt}/3..."
    sleep 3
  done
fi

COMPOSE_FILE="docker-compose.prod.yml"
COMPOSE_ARGS=(-f "$COMPOSE_FILE")

if [[ "${SKIP_KAFKA:-}" == "1" ]]; then
  COMPOSE_ARGS+=(-f docker-compose.prod.no-kafka.yml)
fi

if docker compose version >/dev/null 2>&1; then
  COMPOSE="docker compose"
elif command -v docker-compose >/dev/null 2>&1; then
  COMPOSE="docker-compose"
else
  echo "错误: 未找到 docker compose，请先安装：" >&2
  echo "  apt install -y docker-compose-plugin" >&2
  echo "  或参考 docs/deploy-aliyun.md 安装独立二进制" >&2
  exit 1
fi

if [[ ! -f .env ]]; then
  echo "错误: 缺少 .env，请先执行: cp .env.example .env && nano .env" >&2
  exit 1
fi

if [[ "${SKIP_KAFKA:-}" == "1" ]]; then
  echo "==> SKIP_KAFKA=1：不启动 Kafka，跨实例 WS 使用 Redis Pub/Sub"
fi
echo "==> 使用: $COMPOSE ${COMPOSE_ARGS[*]}"
echo "==> 停止旧容器（保留数据卷）..."
$COMPOSE "${COMPOSE_ARGS[@]}" down --remove-orphans

if [[ "${SKIP_KAFKA:-}" != "1" ]] && ! docker image inspect "${KAFKA_IMAGE:-docker.redpanda.com/redpandadata/redpanda:v24.2.4}" >/dev/null 2>&1; then
  if [[ -x scripts/pull-kafka-image.sh ]]; then
    echo "==> 本地无 Kafka 镜像，尝试加速拉取..."
    bash scripts/pull-kafka-image.sh || true
  fi
fi

echo "==> 构建并启动..."
$COMPOSE "${COMPOSE_ARGS[@]}" up -d --build

echo "==> 等待服务就绪..."
sleep 5

echo "==> 容器状态"
$COMPOSE "${COMPOSE_ARGS[@]}" ps

echo ""
echo "==> 健康检查"
if curl -sf http://127.0.0.1/api/v1/health; then
  echo ""
  echo "✓ 部署成功"
  IP="$(curl -sf ifconfig.me 2>/dev/null || true)"
  IP="${IP:-<ECS公网IP>}"
  echo "  用户端: http://${IP}/app"
  echo "  主播端: http://${IP}/admin"
else
  echo ""
  echo "✗ API 未响应，查看日志: $COMPOSE ${COMPOSE_ARGS[*]} logs --tail 80 backend" >&2
  exit 1
fi
