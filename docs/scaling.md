# 高并发扩展方案（5.5）

## 当前架构

- Gin HTTP + Gorilla WebSocket 同进程
- **多实例**：`zhibo-backend` + `zhibo-backend-2`，Nginx `upstream` 轮询
- **跨实例广播**：**Kafka**（Redpanda 单节点）Topic `zhibo.room.broadcast`
- **重连补偿**：Redis `INCR` 序号 + `LIST` 事件缓冲（与广播解耦）
- Redis：分布式锁 + 热数据缓存
- MySQL：场次、出价持久化

未配置 `KAFKA_BROKERS` 时，跨实例广播降级为 **Redis Pub/Sub**；无 Redis 时为单机内存 Hub。

## 广播路径（Kafka）

```
REST 出价 / Notifier.Publish
        ↓
  Redis INCR seq + LPUSH events（重连用）
        ↓
  Kafka Produce（key=roomId, value=WS Envelope JSON）
        ↓
  ┌─────┴─────┐
  ▼           ▼
Pod-1         Pod-2   （各实例独立 Consumer Group → fan-out）
  ↓           ↓
本机 WS       本机 WS
```

- **带 seq 事件**（出价、排名、成交）：Kafka 广播
- **倒计时 tick**：各实例对本机有观众的房间本地推送

## 部署（ECS · git pull）

在 ECS `/opt/zhibo` 目录：

```bash
# 推荐：拉代码 + 迁移 + 全量重部署
bash scripts/ecs-update.sh
```

或分步：

```bash
cd /opt/zhibo
git -c http.version=HTTP/1.1 pull --ff-only
bash scripts/migrate.sh

# 全栈
docker-compose -f docker-compose.prod.yml up -d --build

# 仅 Kafka + 双 backend + Nginx
docker-compose -f docker-compose.prod.yml up -d --build kafka backend backend-2 nginx
```

> 使用 `docker compose`（V2 插件）时把 `docker-compose` 换成 `docker compose`。  
> `git pull` 若 TLS 失败见结项文档 §26.4。

### 环境变量

| 变量 | 默认 | 说明 |
|------|------|------|
| `KAFKA_BROKERS` | `kafka:9092` | 逗号分隔 |
| `KAFKA_TOPIC` | `zhibo.room.broadcast` | 房间事件 Topic |
| `INSTANCE_ID` | hostname | 每实例唯一，用于 Consumer Group |

## 验证

```bash
docker logs zhibo-backend 2>&1 | grep kafka
# ws: kafka broadcast enabled (multi-instance)
# kafka: ws subscriber started topic=zhibo.room.broadcast group=zhibo-ws-backend-1

docker logs zhibo-backend-2 2>&1 | grep kafka
```

双浏览器同房间出价，跨 Pod 价格应同步。

## 容量参考

| 指标 | 单 Pod | 扩展 |
|------|--------|------|
| WS 连接 | ~5k–10k | 增加 backend 副本 |
| 单房间出价 QPS | ~200–500 | Redis 锁串行 |
| 广播 | Kafka Partition 按 roomId | 水平加 Consumer 实例 |

## 代码索引

| 模块 | 路径 |
|------|------|
| Kafka 广播 | `backend/internal/infra/kafka/broadcast.go` |
| 广播接口 | `backend/internal/ws/broadcast.go` |
| Hub | `backend/internal/ws/hub.go` |
| Redis 事件缓冲 | `backend/internal/infra/redis/ws_broadcast.go` |
| 编排 | `docker-compose.prod.yml` → `kafka`, `backend-2` |
