#!/usr/bin/env bash
# ECS 更新：git pull → 迁移 → 重部署 → 验证
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

git_pull() {
  local attempt
  for attempt in 1 2 3; do
    if git -c http.version=HTTP/1.1 pull --ff-only; then
      return 0
    fi
    echo "    git pull 失败（第 ${attempt}/3 次），3 秒后重试..."
    sleep 3
  done
  echo "错误: git pull 多次失败。可手动执行：" >&2
  echo "  git -c http.version=HTTP/1.1 pull --ff-only" >&2
  echo "  或改用 SSH: git remote set-url origin git@github.com:zhangdingyi123/zhibo-master.git" >&2
  return 1
}

echo "==> 1/5 拉取最新代码 (git pull)"
if [[ -d .git ]]; then
  git_pull
else
  echo "（非 git 目录，跳过 pull）"
fi

echo ""
echo "==> 2/5 数据库迁移"
bash scripts/migrate.sh

echo ""
echo "==> 3/5 重部署容器"
SKIP_GIT_PULL=1 bash scripts/redeploy.sh

echo ""
echo "==> 4/5 API 健康检查"
curl -sf http://127.0.0.1/api/v1/health | head -c 200
echo ""

echo ""
echo "==> 5/5 AI 功能冒烟（文案生成 / TTS / Kimi 连通性）"
bash scripts/test-ai.sh http://127.0.0.1 || true

IP="${ECS_PUBLIC_IP:-47.97.176.185}"
echo ""
echo "✓ 全部完成"
echo "  用户端: http://${IP}/app"
echo "  主播端: http://${IP}/admin"
echo "  订单列表需登录买家账号后访问 /api/v1/orders"
