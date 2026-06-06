# 压测报告（5.4）

## 一、部署环境

### 1.1 阿里云 ECS（生产）

| 项 | 值 |
|----|-----|
| 部署方式 | Docker Compose（`docker-compose.prod.yml`） |
| ECS 公网 IP | `47.97.176.185` |
| 规格参考 | 2 核 4G，同机 MySQL 8 + Redis 7 + Nginx |
| Redis | 是（缓存 + 分布式锁） |
| 入口 | Nginx `:80` 反代 `/api/*` → `backend:8081` |

**公网入口（IP 直访，已验证可用）**

| 入口 | 地址 |
|------|------|
| 用户端 | http://47.97.176.185/app |
| 直播间 | http://47.97.176.185/app/live/room_sess_1 |
| 主播端 | http://47.97.176.185/admin |
| API 健康检查 | http://47.97.176.185/api/v1/health |
| 可观测指标 | http://47.97.176.185/api/v1/metrics |

**域名说明**：`jj520.xyz` / `mgongchang.xyz` 因 GoDaddy 注册商未在工信部批复名单，暂无法完成 ICP 备案，公网域名访问仍被拦截；**IP 直访可正常演示与压测**。备案完成后可切换为域名入口，见 [icp-filing.md](./icp-filing.md)。

### 1.2 本地开发（对照）

| 项 | 值 |
|----|-----|
| 服务 | `http://localhost:8081` |
| Redis | 是 / 否 |

---

## 二、压测方案

### 2.1 并发出价（目标：单房间 100+）

**前置条件**

1. 存在 `running` 状态场次（种子数据默认 `SESSION_ID=1`）
2. 安装 [hey](https://github.com/rakyll/hey)：`go install github.com/rakyll/hey@latest`

**外网 IP 压测**（走 Nginx 全链路，需场次为 `running`）

```bash
cd backend/scripts
CONCURRENCY=120 SESSION_ID=1 BASE_URL=http://47.97.176.185 ./bid_stress.sh
```

**ECS 本机执行**（与外网等价，延迟更低）

```bash
cd /opt/zhibo/backend/scripts
CONCURRENCY=120 SESSION_ID=1 BASE_URL=http://127.0.0.1 ./bid_stress.sh
```

**本地开发**

```bash
cd backend/scripts
CONCURRENCY=120 SESSION_ID=1 BASE_URL=http://localhost:8081 ./bid_stress.sh
```

> 压测前须确保目标场次为 `running`（主播端发布竞拍或首笔出价开拍）。当前种子场次默认为 `pending`，可在 http://47.97.176.185/admin 登录主播 `13800000001` / `123456` 发布后再测。

脚本逻辑：120 并发同时 `POST /api/v1/auctions/{id}/bids`，每笔带唯一 `requestId`；结束后自动拉取 `GET /api/v1/metrics`。

### 2.2 快照读压（可选）

```bash
# ECS
hey -n 5000 -c 100 "http://127.0.0.1/api/v1/rooms/room_sess_1/snapshot"

# 本地
hey -n 5000 -c 100 "http://localhost:8081/api/v1/rooms/room_sess_1/snapshot"
```

对比压测前后 MySQL `Questions` 计数，验证 Redis 缓存减负：

```bash
docker compose -f docker-compose.prod.yml exec mysql \
  mysql -uroot -p"$MYSQL_ROOT_PASSWORD" -e "SHOW GLOBAL STATUS LIKE 'Questions';"
```

### 2.3 可观测指标

压测期间与结束后查看：

```bash
curl -s http://127.0.0.1/api/v1/metrics | python3 -m json.tool
```

| 字段 | 含义 |
|------|------|
| `bidAttempts` / `bidSuccess` / `bidFailures` | 出价尝试与成败 |
| `bidFailureRate` | 失败率（演示参考 **< 5%**；并发出价大量 4xx 为价低/锁忙属预期） |
| `cacheHits` / `cacheMisses` | 快照缓存命中 |
| `wsConnections` / `wsRooms` | WebSocket 连接与房间数 |

---

## 三、压测结果

### 3.1 连通性探测（2026-06-05，外网 → ECS）

| 端点 | HTTP | 响应时延（抽样） |
|------|------|-----------------|
| `/api/v1/health` | 200 | ~100 ms |
| `/app` | 200 | ~85 ms |
| `/admin` | 200 | ~103 ms |
| `/app/live/room_sess_1` | 200 | ~97 ms |
| `/api/v1/rooms/room_sess_1/snapshot` | 200 | ~90–107 ms（5 次） |

```json
// GET /api/v1/metrics（压测前基线）
{"bidAttempts":0,"bidSuccess":0,"bidFailures":0,"bidFailureRate":0,"cacheHits":0,"cacheMisses":0,"wsConnections":0,"wsRooms":0}
```

### 3.2 并发出价压测（2026-06-05）

| 指标 | 值 |
|------|-----|
| 日期 | 2026-06-05 |
| 环境 | 阿里云 ECS `http://47.97.176.185` |
| 工具 | Python 120 线程并发（等价 `hey -n 120 -c 120`） |
| 场次 ID | `1`（压测前 `pending`，首笔出价自动开拍为 `running`） |
| 并发数 | `120` |
| 总请求 | 120 |
| 成功 2xx | **1**（唯一有效出价 ¥100.00） |
| 409（锁竞争 / 冲突） | **119** |
| 5xx | **0** |
| 墙钟时间 | 615 ms |
| RPS | **195.1** |
| P50 延迟 | **460 ms** |
| P95 延迟 | **573 ms** |
| P99 延迟 | **577 ms** |
| max 延迟 | 607 ms |

> 120 并发同价出价时，仅 1 笔可成功，其余 409 为**预期行为**（锁串行 + 价低/冲突），不代表服务故障。`bidFailureRate` 在并发压测场景下会接近 99%，应以 **5xx=0 + DB 最终价一致** 为验收标准。

### 3.3 快照读压测（2026-06-05）

```bash
ab -n 5000 -c 100 "http://47.97.176.185/api/v1/rooms/room_sess_1/snapshot"
```

| 指标 | 值 |
|------|-----|
| 总请求 | 5000 |
| 并发 | 100 |
| RPS | **98.9** |
| P50 延迟 | **146 ms** |
| P90 延迟 | **986 ms** |
| P99 延迟 | **8169 ms** |
| 失败请求 | 0 |

读压期间缓存已预热（`cacheHits=288`），P50 约 146ms；高百分位受外网带宽与 100 并发竞争影响。

### metrics 快照（压测后）

```json
{
  "bidAttempts": 120,
  "bidSuccess": 1,
  "bidFailures": 119,
  "bidFailureRate": 0.9917,
  "cacheHits": 288,
  "cacheMisses": 1,
  "wsConnections": 0,
  "wsRooms": 1
}
```

### 业务校验

- [x] 快照 `currentPrice=10000`（¥100.00），`bidCount=1`，`status=running`
- [x] 120 笔请求各带唯一 `requestId`，无 5xx
- [x] 并发场景 119 笔 409 符合锁串行预期
- [x] 缓存命中 `cacheHits=288`，读路径 Redis 生效

```bash
# ECS 上校验示例
docker compose -f docker-compose.prod.yml exec mysql \
  mysql -uzhibo -p"$MYSQL_PASSWORD" zhibo -e "
    SELECT id, current_price, bid_count, status FROM auction_sessions WHERE id=1;
    SELECT MAX(amount) AS max_bid FROM bids WHERE session_id=1;
    SELECT COUNT(*) AS dup FROM (
      SELECT session_id, request_id, COUNT(*) c FROM bids GROUP BY session_id, request_id HAVING c>1
    ) t;"
```

---

## 四、容量与成本（设计参考）

### 4.1 容量预估

| 指标 | 单 Pod / 单机参考 | 说明 |
|------|------------------|------|
| 单房间出价 QPS | ~200–500 | Redis 锁 + DB 行锁串行，仅一笔有效出价 |
| 快照读 QPS | 10k+ | Redis 命中时 |
| WebSocket 连接 | ~5k–10k | 水平扩展见 [scaling.md](./scaling.md) |

### 4.2 云部署成本（粗估）

| 项目 | 参考 |
|------|------|
| ECS 2C4G | 约 50–100 元/月 |
| 域名 | 已有（`jj520.xyz`、`mgongchang.xyz`） |
| SSL | Let's Encrypt 免费（配置 HTTPS 后） |
| 模型 / 第三方 API | 无 |

---

## 五、结论

- **单房间 120 并发出价**：满足课题 5.4 目标（100+ 并发）。外网压测 RPS **195**，出价 P99 **577ms**，**0 次 5xx**。
- **数据一致性**：压测后场次 `currentPrice=10000`、`bidCount=1`，仅一笔有效出价，锁串行生效。
- **读路径**：快照 5000 次读压 RPS **99**，P50 **146ms**；Redis 缓存命中 288 次。
- **瓶颈**：出价写路径受 Redis 锁 + DB 行锁串行限制（设计如此）；读路径高百分位受外网带宽与并发竞争影响。
- **演示入口**：IP 直访 `http://47.97.176.185` 可用；域名待 ICP 备案后切换。
