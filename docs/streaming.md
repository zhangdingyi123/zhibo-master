# 最小推拉流服务 — 完整教程

基于 [SRS](https://ossrs.io/)：主播 **RTMP 推流**，观众 **HLS 拉流**。流名与竞拍房间 `roomId` 一一对应（如 `room_sess_1`）。

## 架构

```
OBS / ffmpeg ──RTMP:1935──► SRS ──HLS──► 浏览器 (hls.js)
                              ▲
管理端复制 pushUrl ◄── GET /api/v1/streams/:roomId
         │
    竞拍 WebSocket（出价/排名/倒计时，与视频流独立）
```

---

## 第一步：启动全套服务

```bash
# 项目根目录
docker compose up -d          # MySQL + Redis + SRS
cd backend && go run ./cmd/server   # 终端 1，:8081
cd frontend && npm install && npm run dev   # 终端 2，:5173
```

| 端口 | 服务 | 用途 |
|------|------|------|
| 3306 | MySQL | 竞拍数据 |
| 6379 | Redis | 缓存 / 锁 |
| 1935 | SRS | RTMP 推流 |
| 8080 | SRS | HLS 拉流（Vite 代理 `/live` → 8080） |
| 8081 | Go API | REST + WebSocket |
| 5173 | Vite | 前端开发 |

**健康检查**

```bash
curl http://localhost:8081/api/v1/health          # {"status":"ok"}
curl http://localhost:8080/api/v1/summaries       # SRS 摘要
curl http://localhost:8081/api/v1/streams/room_sess_1   # 推流地址
```

---

## 第二步：确认房间号 (roomId)

种子数据里已有场次，房间号固定：

| 场次 ID | roomId | 商品 | 状态（种子） |
|---------|--------|------|--------------|
| 1 | `room_sess_1` | Vintage 机械腕表 | pending |
| 2 | `room_sess_2` | 手工皮具钱包 | pending |

**方式 A — 管理后台**

1. 打开 `http://localhost:5173/admin`，登录主播 `13800000001` / `123456`
2. 商品列表 → 点进商品 → **竞拍进度** 区块可见 `房间号` 与 **推流地址**

**方式 B — API**

```bash
curl -s http://localhost:8081/api/v1/auctions/1 | jq '.roomId'
# "room_sess_1"
```

> 推流码必须与 `roomId` 完全一致，否则用户端拉不到画面。

---

## 第三步：主播推流

### 方式 A — 一键验证脚本（推荐首次）

```bash
chmod +x scripts/stream_demo.sh
./scripts/stream_demo.sh room_sess_1
```

脚本会：检查 SRS → 推 12 秒测试画面 → 确认 m3u8 可访问 → 打印用户端链接。

### 方式 B — OBS（真实直播）

1. OBS → **设置 → 推流**
2. 服务：**自定义**
3. 服务器：`rtmp://localhost:1935/live`
4. 推流码：`room_sess_1`
5. 点 **开始推流**

### 方式 C — ffmpeg 持续推流

```bash
ffmpeg -re -f lavfi -i testsrc=size=1280x720:rate=30 \
  -f lavfi -i sine=frequency=1000 \
  -c:v libx264 -preset ultrafast -tune zerolatency -pix_fmt yuv420p \
  -c:a aac -b:a 128k \
  -f flv rtmp://localhost:1935/live/room_sess_1
```

推流地址 API：

`GET /api/v1/streams/room_sess_1`

```json
{
  "roomId": "room_sess_1",
  "pushUrl": "rtmp://localhost:1935/live/room_sess_1",
  "hlsUrl": "/live/room_sess_1.m3u8",
  "flvUrl": "/live/room_sess_1.flv"
}
```

---

## 第四步：验证拉流

1. 浏览器打开 **`http://localhost:5173/app/live/room_sess_1`**
2. 有推流时：角标 **LIVE**，显示真实画面
3. 无推流时：封面占位 + **等待推流**（每 5 秒自动重试）

**命令行验证 HLS**

```bash
curl -s http://localhost:8080/live/room_sess_1.m3u8 | head
# 应看到 #EXTM3U
```

**开发环境代理验证**（经 Vite）

```bash
curl -s http://localhost:5173/live/room_sess_1.m3u8 | head
```

---

## 第五步：推拉流 + 竞拍联调（完整演示）

按顺序走通「直播 + 竞拍」全流程：

| 步骤 | 角色 | 操作 | 预期 |
|------|------|------|------|
| 1 | 运维 | `docker compose up -d` + 启后端/前端 | 各端口正常 |
| 2 | 主播 | 管理端登录，确认 `room_sess_1` 推流地址 | 可复制 pushUrl |
| 3 | 主播 | OBS / ffmpeg 开始推流 | HLS 可访问 |
| 4 | 主播 | 管理端发布竞拍或等场次 `running` | 用户端可出价 |
| 5 | 买家 | `http://localhost:5173/app`，登录 `13800000002` | 进入直播间 |
| 6 | 买家 | 打开 `/app/live/room_sess_1` | 画面 LIVE + 倒计时 |
| 7 | 买家 | 出价 | WS 实时更新价格/排名 |
| 8 | 双浏览器 | 两个买家同房间 | 排名、倒计时一致 |

> **说明**：视频流（SRS）与竞拍信令（WebSocket）是两条独立通道。推流只影响画面；出价、排名、延时仍走 WS，互不依赖。

**双窗口演示建议**

- 窗口 A（主播）：管理端 + OBS 推流
- 窗口 B（买家 1）：用户端直播间 + 出价
- 窗口 C（买家 2）：无痕模式登录 `13800000002` 以外账号，验证排名同步

---

## 第六步：生产部署

本地验证通过后，上云需额外处理：

### 6.1 docker-compose.prod 已含 SRS

```bash
docker compose -f docker-compose.prod.yml up -d --build
```

`deploy/nginx.conf` / `nginx-https.conf` 已配置 `/live/` → `srs:8080`。

### 6.2 环境变量

`.env` 生产示例：

```env
STREAM_RTMP_HOST=你的公网IP:1935
STREAM_HLS_BASE=/live
```

| 变量 | 本地 | 生产 |
|------|------|------|
| `STREAM_RTMP_HOST` | `localhost:1935` | 公网 IP 或 `live.example.com:1935` |
| `STREAM_HLS_BASE` | `/live` | `/live`（经 HTTPS 域名访问） |

### 6.3 安全组 / 防火墙

| 端口 | 是否对外开放 | 说明 |
|------|:------------:|------|
| 80 / 443 | 是 | 网站 + HLS 拉流 |
| 1935 | 是 | 主播 RTMP 推流（OBS 从外网推入） |
| 8080 | 否 | 仅容器内，由 nginx 反代 |
| 3306 / 6379 | 否 | 数据库不暴露 |

### 6.4 主播 OBS（生产）

- 服务器：`rtmp://你的公网IP:1935/live` 或专用子域名
- 推流码：`room_sess_1`（每场竞拍对应各自 roomId）
- 观众访问：`https://你的域名/app/live/room_sess_1`（HLS 走 `https://域名/live/...`）

更完整的 ECS 部署见 [deploy-aliyun.md](./deploy-aliyun.md)。

---

## 故障排查

| 现象 | 可能原因 | 处理 |
|------|----------|------|
| 一直「等待推流」 | 未推流或 roomId 不一致 | 核对推流码 = 房间号；`docker logs zhibo-srs` |
| `curl m3u8` 404 | SRS 未收到 RTMP | 确认 1935 端口、OBS 已开始推流 |
| 管理端无推流地址 | 后端未启动 | `curl localhost:8081/api/v1/streams/room_sess_1` |
| 前端有流但无竞拍 | 仅视频通，WS 未连 | 看右上角连接状态；检查 8081 / WS 代理 |
| `npm run dev` 拉不到流 | Vite 代理未生效 | 确认 `vite.config.ts` 有 `/live` → `:8080` |
| 生产 HTTPS 混合内容 | 页面 https 拉 http 流 | HLS 必须走 `https://域名/live/...`（nginx 已配） |
| ffmpeg 报 Connection refused | SRS 未启动 | `docker compose up -d srs` |

**常用诊断命令**

```bash
docker ps | grep srs
docker logs --tail 50 zhibo-srs
curl http://localhost:8080/api/v1/streams/
curl -I http://localhost:8080/live/room_sess_1.m3u8
```

---

## 环境变量速查

| 变量 | 默认 | 说明 |
|------|------|------|
| `STREAM_RTMP_HOST` | `localhost:1935` | 返回给主播的 RTMP 主机 |
| `STREAM_HLS_BASE` | `/live` | HLS 路径前缀（生产经 nginx 反代） |

---

## 相关文件

| 文件 | 说明 |
|------|------|
| `deploy/srs.conf` | SRS 配置 |
| `docker-compose.yml` | 本地 SRS 服务 |
| `scripts/stream_demo.sh` | 一键推流验证 |
| `backend/internal/api/handler/stream_handler.go` | 推流地址 API |
| `frontend/src/components/auction/LiveVideo.tsx` | HLS 播放器 |
| `frontend/vite.config.ts` | `/live` 开发代理 |
| `deploy/nginx.conf` | 生产 HLS 反代 |
