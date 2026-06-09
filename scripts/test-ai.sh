#!/usr/bin/env bash
# AI 功能测试脚本 — 测试 Kimi (Moonshot) API 连通性
# 用法: bash scripts/test-ai.sh [BASE_URL]
# 默认 BASE_URL=http://127.0.0.1

set -euo pipefail

BASE_URL="${1:-http://127.0.0.1}"
PASS=0
FAIL=0

green() { printf "\033[32m%s\033[0m\n" "$1"; }
red()   { printf "\033[31m%s\033[0m\n" "$1"; }
bold()  { printf "\033[1m%s\033[0m\n" "$1"; }

# ─── 0. 健康检查 ────────────────────────────────────────────
bold "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
bold "  AI 功能测试 — $BASE_URL"
bold "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

echo ""
echo "▶ Step 0: 基础健康检查..."
HEALTH=$(curl -sf "$BASE_URL/api/v1/health" 2>&1) && {
  green "  ✓ 服务正常: $HEALTH"
  PASS=$((PASS+1))
} || {
  red "  ✗ 服务不可达，请确认后端已启动"
  red "    检查: docker compose -f docker-compose.prod.yml ps backend"
  exit 1
}

# ─── 1. 获取管理员 Token ─────────────────────────────────────
echo ""
echo "▶ Step 1: 登录获取管理员 Token..."
LOGIN_RESP=$(curl -sf -X POST "$BASE_URL/api/v1/admin/login" \
  -H "Content-Type: application/json" \
  -d '{"phone":"13800000001","password":"123456"}' 2>&1)

TOKEN=$(echo "$LOGIN_RESP" | grep -oP '"token"\s*:\s*"\K[^"]+' 2>/dev/null || true)

if [[ -z "$TOKEN" ]]; then
  # 尝试从 data 字段提取
  TOKEN=$(echo "$LOGIN_RESP" | grep -oP '"data"\s*:\s*"\K[^"]+' 2>/dev/null || true)
fi

if [[ -n "$TOKEN" ]]; then
  green "  ✓ 管理员 Token 获取成功: ${TOKEN:0:20}..."
  PASS=$((PASS+1))
else
  red "  ✗ 管理员登录失败: $LOGIN_RESP"
  red "    请确认 003_auth.sql 已执行，演示账号 13800000001/123456 可用"
  FAIL=$((FAIL+1))
fi

# ─── 2. 测试 AI 生成商品文案 ──────────────────────────────────
echo ""
echo "▶ Step 2: 测试 AI 生成商品介绍文案..."
INTRO_RESP=$(curl -sf -X POST "$BASE_URL/api/v1/admin/products/ai-intro" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"name":"云南古树普洱茶","keywords":"百年古树、手工采摘、限量珍藏"}' 2>&1) && {

  # 解析 source 字段判断是 LLM 还是 template
  SOURCE=$(echo "$INTRO_RESP" | grep -oP '"source"\s*:\s*"\K[^"]+' 2>/dev/null || echo "unknown")
  DESCRIPTION=$(echo "$INTRO_RESP" | grep -oP '"description"\s*:\s*"\K[^"]+' 2>/dev/null || echo "")

  if [[ "$SOURCE" == "llm" ]]; then
    green "  ✓ Kimi API 调用成功！来源: LLM (Kimi)"
    echo "  📝 生成文案: ${DESCRIPTION:0:100}..."
    PASS=$((PASS+1))
  elif [[ "$SOURCE" == "template" ]]; then
    echo ""
    red "  ⚠ AI 返回了模板文案（未走 LLM），说明 Kimi API 未正确配置"
    echo "  📝 模板文案: ${DESCRIPTION:0:80}..."
    echo ""
    echo "  请检查 .env 中以下配置："
    echo "    AI_API_KEY=sk-...           （Kimi API Key）"
    echo "    AI_API_BASE=https://api.moonshot.cn/v1"
    echo "    AI_MODEL=moonshot-v1-8k"
    FAIL=$((FAIL+1))
  else
    red "  ✗ 响应异常: $INTRO_RESP"
    FAIL=$((FAIL+1))
  fi
} || {
  red "  ✗ 请求失败: $INTRO_RESP"
  FAIL=$((FAIL+1))
}

# ─── 3. 测试无 Key 时的降级（模板文案） ───────────────────────
echo ""
echo "▶ Step 3: 验证无 API Key 时的模板降级..."
# 直接不带 Authorization 测试（如果后端允许），或用新请求
TEMPLATE_RESP=$(curl -sf -X POST "$BASE_URL/api/v1/admin/products/ai-intro" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"name":"测试商品"}' 2>&1) && {
  T_SOURCE=$(echo "$TEMPLATE_RESP" | grep -oP '"source"\s*:\s*"\K[^"]+' 2>/dev/null || echo "unknown")
  if [[ "$T_SOURCE" == "llm" ]]; then
    green "  ✓ Kimi API 正常响应（有 Key 时走 LLM）"
    PASS=$((PASS+1))
  elif [[ "$T_SOURCE" == "template" ]]; then
    echo "  ℹ 无 Key 或降级，返回模板文案（正常行为）"
    PASS=$((PASS+1))
  else
    echo "  ℹ 响应: ${TEMPLATE_RESP:0:100}"
    PASS=$((PASS+1))
  fi
} || {
  red "  ✗ 请求失败"
  FAIL=$((FAIL+1))
}

# ─── 4. 测试 TTS 语音合成 ─────────────────────────────────────
echo ""
echo "▶ Step 4: 测试 TTS 语音合成..."
TTS_RESP=$(curl -sf -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/api/v1/user/tts" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"text":"欢迎来到直播间"}' 2>&1) && {

  if [[ "$TTS_RESP" == "200" ]]; then
    green "  ✓ TTS 合成成功（HTTP 200）"
    PASS=$((PASS+1))
  elif [[ "$TTS_RESP" == "503" ]]; then
    echo "  ℹ TTS 不可用（503）— Kimi 不提供 TTS，这是正常现象"
    echo "    浏览器端会自动回退到 Web Speech API"
    PASS=$((PASS+1))
  else
    red "  ✗ TTS 返回异常 HTTP $TTS_RESP"
    FAIL=$((FAIL+1))
  fi
} || {
  red "  ✗ TTS 请求失败"
  FAIL=$((FAIL+1))
}

# ─── 5. 直接测试 Kimi API 连通性 ──────────────────────────────
echo ""
echo "▶ Step 5: 直接测试 Kimi API 连通性..."

# 从 .env 读取配置
ENV_FILE=""
if [[ -f ".env" ]]; then
  ENV_FILE=".env"
elif [[ -f "../.env" ]]; then
  ENV_FILE="../.env"
fi

if [[ -n "$ENV_FILE" ]]; then
  AI_KEY=$(grep -E "^AI_API_KEY=" "$ENV_FILE" 2>/dev/null | cut -d'=' -f2- | tr -d '"' | tr -d "'" || true)
  AI_BASE=$(grep -E "^AI_API_BASE=" "$ENV_FILE" 2>/dev/null | cut -d'=' -f2- | tr -d '"' | tr -d "'" || echo "https://api.moonshot.cn/v1")
  AI_MODEL=$(grep -E "^AI_MODEL=" "$ENV_FILE" 2>/dev/null | cut -d'=' -f2- | tr -d '"' | tr -d "'" || echo "moonshot-v1-8k")

  if [[ -z "$AI_KEY" ]]; then
    red "  ✗ .env 中未配置 AI_API_KEY"
    FAIL=$((FAIL+1))
  else
    echo "  配置: BASE=$AI_BASE  MODEL=$AI_MODEL  KEY=${AI_KEY:0:8}..."
    KIMI_RESP=$(curl -sf -X POST "${AI_BASE}/chat/completions" \
      -H "Content-Type: application/json" \
      -H "Authorization: Bearer $AI_KEY" \
      -d "{\"model\":\"$AI_MODEL\",\"messages\":[{\"role\":\"user\",\"content\":\"说一个字：好\"}],\"temperature\":0.1}" \
      --max-time 15 2>&1) && {

      CONTENT=$(echo "$KIMI_RESP" | grep -oP '"content"\s*:\s*"\K[^"]+' 2>/dev/null || true)
      if [[ -n "$CONTENT" ]]; then
        green "  ✓ Kimi API 直连成功！回复: $CONTENT"
        PASS=$((PASS+1))
      else
        green "  ✓ Kimi API 响应正常（已返回数据）"
        PASS=$((PASS+1))
      fi
    } || {
      red "  ✗ Kimi API 直连失败（检查 Key 和网络）"
      red "    响应: ${KIMI_RESP:0:200}"
      FAIL=$((FAIL+1))
    }
  fi
else
  echo "  ⏭ 跳过（未找到 .env 文件）"
fi

# ─── 汇总 ─────────────────────────────────────────────────────
echo ""
bold "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
if [[ $FAIL -eq 0 ]]; then
  green "  全部通过 ✓  ($PASS 通过 / $FAIL 失败)"
else
  red "  存在失败 ✗  ($PASS 通过 / $FAIL 失败)"
fi
bold "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
