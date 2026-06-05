# 压测报告（5.4）

## 一、部署环境

### 1.1 阿里云 ECS（生产）

| 项 | 值 |
|----|-----|
| 部署方式 | Docker Compose（`docker-compose.prod.yml`） |
| ECS 公网 IP | `47.97.176.185` |
| 用户端域名 | `jj520.xyz` → `/app` |
| 主播端域名 | `mgongchang.xyz` → `/admin` |
| 规格参考 | 2 核 4G，同机 MySQL 8 + Redis 7 + Nginx |
| Redis | 是（缓存 + 分布式锁） |
| 入口 | Nginx `:80` 反代 `/api/*` → `backend:8081` |

**访问说明**：域名已解析至 ECS，但公网 HTTP 访问当前可能被阿里云 **ICP 备案拦截页** 挡住（`Non-compliance ICP Filing`）。压测应在 **ECS 本机** 通过 `127.0.0.1` 执行，不受备案拦截影响；对外演示需先完成域名备案或接入已备案域名。

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

**ECS 上执行**（推荐，走 Nginx 全链路）

```bash
cd /opt/zhibo/backend/scripts

# 先确认健康
curl -s http://127.0.0.1/api/v1/health
curl -s http://127.0.0.1/api/v1/auctions?status=running | head -c 500

CONCURRENCY=120 SESSION_ID=1 BASE_URL=http://127.0.0.1 ./bid_stress.sh
```

**本地执行**

```bash
cd backend/scripts
CONCURRENCY=120 SESSION_ID=1 BASE_URL=http://localhost:8081 ./bid_stress.sh
```

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

## 三、压测结果（在 ECS 执行后填写）

| 指标 | 值 |
|------|-----|
| 日期 | YYYY-MM-DD |
| 环境 | 阿里云 ECS `47.97.176.185` / 本机 `127.0.0.1` |
| 场次 ID | `1` |
| 并发数 | `120` |
| 总请求 | |
| 成功 2xx | |
| 4xx（价低 / 锁忙） | |
| 5xx | |
| RPS | |
| P50 延迟 | |
| P99 延迟 | |

### metrics 快照（压测后）

```json
{
  "bidAttempts": 0,
  "bidSuccess": 0,
  "bidFailures": 0,
  "bidFailureRate": 0,
  "cacheHits": 0,
  "cacheMisses": 0,
  "wsConnections": 0,
  "wsRooms": 0
}
```

### 业务校验

- [ ] MySQL `auction_sessions.current_price` 与最高 `bids.amount` 一致
- [ ] `bids` 无重复 `(session_id, request_id)`
- [ ] `bidFailureRate` 可接受
- [ ] Redis 快照与 DB 抽样一致

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

（填写：是否满足单房间 100+ 并发、瓶颈在锁/DB/Nginx、对外访问是否需先完成备案、后续优化点）

**参考结论模板**：

- 单机 ECS 在 **本机压测** 下可支撑单房间 **120 并发出价** 请求，业务层通过锁串行保证价格一致。
- 并发出价场景下大量 **4xx 为预期**（仅最高价胜出，其余价低或锁忙），应以 **DB 最终价 + metrics** 为准，而非 2xx 比例。
- 公网域名访问需 **ICP 备案** 通过后，外网用户方可正常打开 `jj520.xyz` / `mgongchang.xyz` 进行演示。
