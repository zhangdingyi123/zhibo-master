# 高并发扩展方案（5.5）

## 当前架构

- Gin HTTP + Gorilla WebSocket 同进程
- **多实例**：`zhibo-backend` + `zhibo-backend-2`，Nginx `upstream` 轮询
- **跨实例广播**：Redis Pub/Sub `zhibo:room:{roomId}:broadcast`
- **跨实例重连补偿**：Redis `INCR` 序号 + `LIST` 事件缓冲（`zhibo:room:{roomId}:events`）
- Redis：分布式锁 + 热数据缓存
- MySQL：场次、出价持久化

单房间 **100+ 并发出价** 由 Redis 锁 + DB 行锁串行化，压测脚本见 `backend/scripts/bid_stress.sh`。

## 广播路径（已实现）

```
REST 出价 / Notifier.Publish
        ↓
  Redis INCR seq + LPUSH events + PUBLISH
        ↓
  ┌─────┴─────┐
  ▼           ▼
Pod-1 Hub    Pod-2 Hub   （PSUBSCRIBE zhibo:room:*:broadcast）
  ↓           ↓
本机 WS 连接  本机 WS 连接
```

- **带 seq 事件**（出价、排名、成交等）：走 Pub/Sub，所有实例同步
- **倒计时 tick**：仍由**各实例对本机有观众的房间**本地推送（避免多实例重复 fan-out）

Redis 不可用时自动回退：**内存 Hub 广播**（单实例模式）。

## 部署双实例

```bash
docker compose -f docker-compose.prod.yml up -d --build backend backend-2 nginx
```

Nginx 已配置：

```nginx
upstream zhibo_backend {
    server zhibo-backend:8081;
    server zhibo-backend-2:8081;
}
```

客户端无需改动；多浏览器连到不同 Pod 时，出价后价格仍应同步。

## 验证多实例广播

1. 开两个浏览器进同一直播间（刷新多次或无痕，提高命中不同 Pod 概率）
2. 一方出价，另一方价格应更新
3. 查看日志：`ws: redis pub/sub broadcast enabled (multi-instance)`

可选：分别 `docker logs zhibo-backend` / `zhibo-backend-2` 观察连接分布。

## 1000+ WebSocket 连接（后续）

| 指标 | 单 Pod 参考 | 扩展方式 |
|------|-------------|----------|
| WS 连接 | ~5k–10k | 增加 backend 副本 + 已有 Pub/Sub |
| 单房间 QPS 出价 | ~200–500（锁串行） | 业务上合并出价批次（非课题范围） |
| 读快照 | 10k+ QPS | Redis 集群 |

### 可选增强

- **Sticky Session**（`ip_hash`）：减少跨节点连接迁移，非必须
- **Kafka/NATS**：房间数极大时可替换 Pub/Sub，当前 Redis 足够

## 代码索引

| 模块 | 路径 |
|------|------|
| Pub/Sub + 事件 LIST | `backend/internal/infra/redis/ws_broadcast.go` |
| Hub 多实例 | `backend/internal/ws/hub.go` |
| Redis Key | `backend/internal/domain/redis_keys.go` |
| 双实例编排 | `docker-compose.prod.yml` → `backend-2` |
