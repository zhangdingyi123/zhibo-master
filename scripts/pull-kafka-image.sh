#!/usr/bin/env bash
# 国内 ECS 从加速镜像拉取 Redpanda 并打 tag，供 docker-compose.prod.yml 使用
set -euo pipefail

TARGET="${KAFKA_IMAGE:-docker.redpanda.com/redpandadata/redpanda:v24.2.4}"
MIRRORS=(
  "docker.m.daocloud.io/redpandadata/redpanda:v24.2.4"
  "docker.1ms.run/redpandadata/redpanda:v24.2.4"
  "redpandadata/redpanda:v24.2.4"
)

echo "==> 目标镜像: $TARGET"

for src in "${MIRRORS[@]}"; do
  echo "==> 尝试拉取: $src"
  if docker pull "$src"; then
    docker tag "$src" "$TARGET"
    echo "✓ 已标记: $TARGET"
    docker images | grep redpanda || true
    exit 0
  fi
  echo "    失败，换下一个镜像源..."
done

echo "错误: 所有镜像源均拉取失败。" >&2
echo "可选方案：" >&2
echo "  1. 配置 Docker 加速：见 docs/deploy-aliyun.md § Kafka 镜像" >&2
echo "  2. 跳过 Kafka：SKIP_KAFKA=1 bash scripts/redeploy.sh" >&2
exit 1
