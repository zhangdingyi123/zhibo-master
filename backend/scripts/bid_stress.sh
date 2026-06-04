#!/usr/bin/env bash
# 阶段 5.4：单场次并发出价压测（需服务已启动且存在 running 场次）
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8081}"
SESSION_ID="${SESSION_ID:-1}"
CONCURRENCY="${CONCURRENCY:-120}"
USER_PREFIX="${USER_PREFIX:-buyer_}"

if ! command -v hey >/dev/null 2>&1; then
  echo "请先安装 hey: go install github.com/rakyll/hey@latest"
  exit 1
fi

BODY='{"amount":10000,"requestId":"stress-{{.RequestNumber}}"}'
echo "POST $BASE_URL/api/v1/auctions/$SESSION_ID/bids concurrency=$CONCURRENCY"
echo "metrics: $BASE_URL/api/v1/metrics"

hey -n "$CONCURRENCY" -c "$CONCURRENCY" -m POST \
  -H "Content-Type: application/json" \
  -H "X-Mock-Open-Id: ${USER_PREFIX}001" \
  -d "$BODY" \
  "$BASE_URL/api/v1/auctions/$SESSION_ID/bids"

echo ""
echo "--- metrics snapshot ---"
curl -s "$BASE_URL/api/v1/metrics" | python3 -m json.tool 2>/dev/null || curl -s "$BASE_URL/api/v1/metrics"
