#!/usr/bin/env bash
# 推拉流本地验证：检查 SRS → 推测试流 → 探测 HLS
set -euo pipefail

ROOM_ID="${1:-room_sess_1}"
RTMP_HOST="${STREAM_RTMP_HOST:-localhost:1935}"
HLS_BASE="${STREAM_HLS_BASE:-http://localhost:8080/live}"
PUSH_URL="rtmp://${RTMP_HOST}/live/${ROOM_ID}"
HLS_URL="${HLS_BASE%/}/${ROOM_ID}.m3u8"
PUSH_SECS="${PUSH_SECS:-12}"

echo "==> 房间: ${ROOM_ID}"
echo "    推流: ${PUSH_URL}"
echo "    拉流: ${HLS_URL}"

if ! command -v ffmpeg >/dev/null 2>&1; then
  echo "错误: 未找到 ffmpeg，请先安装 (macOS: brew install ffmpeg)" >&2
  exit 1
fi

if ! curl -sf "http://${RTMP_HOST%%:*}:8080/api/v1/summaries" >/dev/null 2>&1; then
  echo "错误: SRS 未响应，请先执行: docker compose up -d srs" >&2
  exit 1
fi
echo "✓ SRS 已启动"

ffmpeg -hide_banner -loglevel error \
  -re -f lavfi -i "testsrc=size=1280x720:rate=30" \
  -f lavfi -i "sine=frequency=1000" \
  -t "${PUSH_SECS}" \
  -c:v libx264 -preset ultrafast -tune zerolatency -pix_fmt yuv420p \
  -c:a aac -b:a 128k \
  -f flv "${PUSH_URL}" &
FFMPEG_PID=$!

echo "==> 推流中 (${PUSH_SECS}s)…"
for i in $(seq 1 20); do
  sleep 1
  if curl -sf "${HLS_URL}" | grep -q '#EXTM3U'; then
    echo "✓ HLS 可拉取"
    echo ""
    echo "下一步:"
    echo "  1. 浏览器打开 http://localhost:5173/app/live/${ROOM_ID}"
    echo "  2. 角标应显示 LIVE（推流结束后会回到「等待推流」）"
    echo "  3. 持续推流请用 OBS 或去掉 -t 参数手动 ffmpeg"
    wait "${FFMPEG_PID}" 2>/dev/null || true
    exit 0
  fi
done

kill "${FFMPEG_PID}" 2>/dev/null || true
echo "错误: ${PUSH_SECS}s 内未拿到 m3u8，检查 SRS 日志: docker logs zhibo-srs" >&2
exit 1
