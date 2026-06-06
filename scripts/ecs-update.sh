#!/usr/bin/env bash
# ECS 更新：拉代码 → 执行迁移 → 重部署 → 验证
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

echo "==> 1/4 拉取最新代码"
if [[ -d .git ]]; then
  git pull --ff-only
else
  echo "（非 git 目录，跳过 pull）"
fi

echo ""
echo "==> 2/4 数据库迁移"
bash scripts/migrate.sh

echo ""
echo "==> 3/4 重部署容器"
bash scripts/redeploy.sh

echo ""
echo "==> 4/4 订单 API 冒烟（需买家 token 时可手动测）"
curl -sf http://127.0.0.1/api/v1/health | head -c 200
echo ""

IP="${ECS_PUBLIC_IP:-47.97.176.185}"
echo ""
echo "✓ 全部完成"
echo "  用户端: http://${IP}/app"
echo "  主播端: http://${IP}/admin"
echo "  订单列表需登录买家账号后访问 /api/v1/orders"
