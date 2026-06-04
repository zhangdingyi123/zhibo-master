# 高并发扩展方案（5.5 加分）

## 当前架构（单进程）

- Gin HTTP + Gorilla WebSocket 同进程
- 房间级 `Hub` 内存路由，按 `roomId` 隔离广播
- Redis：分布式锁 + 热数据缓存
- MySQL：场次、出价持久化

单房间 **100+ 并发出价** 由 Redis 锁 + DB 行锁串行化，压测脚本见 `backend/scripts/bid_stress.sh`。

## 1000+ WebSocket 连接扩展

### 1. 连接层水平扩展

```
                    ┌─────────────┐
  Client ──WS──────►│  WS Gateway │  (N 实例，无状态握手)
                    └──────┬──────┘
                           │ Redis Pub/Sub  channel: zhibo:room:{roomId}
                    ┌──────┴──────┐
                    ▼             ▼
              WS Pod-1       WS Pod-2
```

- 每实例维护本机连接表；跨实例广播走 **Redis Pub/Sub** 或 **NATS/Kafka**
- 客户端 `subscribe.roomId` 不变；网关根据 `roomId` 订阅频道

### 2. 房间级路由

- **Sticky Session**（按 roomId 一致性哈希）可减少跨节点广播，但非必须
- 权威倒计时、快照仍以 **Redis + DB** 为准，任意节点可读缓存推送 tick

### 3. 出价 API

- 保持无状态 REST，任意 API 节点 + Redis 锁
- 可选：按 `sessionId` 分片到固定 API 分区，降低锁竞争（多房间并行）

### 4. 容量预估（参考）

| 指标 | 单 Pod 参考 | 扩展方式 |
|------|-------------|----------|
| WS 连接 | ~5k–10k | 增加 WS Pod + Pub/Sub |
| 单房间 QPS 出价 | ~200–500（锁串行） | 业务上合并出价批次（非课题范围） |
| 读快照 | 10k+ QPS | Redis 集群 |

## 演示建议

- 100+ 出价：`hey` 压测 + `metrics` 观察失败率
- 1000+ 连接：可用 `k6` / `websocket` 脚本多连接订阅，配合 2 个 WS 实例 + Redis 广播（实现为 P2 加分）
