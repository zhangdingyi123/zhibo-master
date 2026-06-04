# 缓存与一致性（阶段 5）

> 完整说明（MySQL 表结构、出价事务、读路径、代码索引）见 [mysql-redis.md](./mysql-redis.md)。

## 5.1 Redis 热数据

| Key | 类型 | 内容 |
|-----|------|------|
| `zhibo:room:{roomId}:snapshot` | String(JSON) | 场次快照：当前价、人数、状态、倒计时等 |
| `zhibo:session:{sessionId}:snapshot` | String(JSON) | 与房间快照双写，支持按场次 ID 查询 |
| `zhibo:room:{roomId}:rank` | ZSET | 用户最高出价排序（score 编码价+序） |
| `zhibo:room:{roomId}:rank_top` | String(JSON) | TopN 完整展示字段（昵称、头像） |
| `zhibo:room:{roomId}:countdown` | String | 权威结束时间 Unix 毫秒 |
| `zhibo:room:{roomId}:participants` | SET | 参与用户 ID |
| `zhibo:room:{roomId}:null` | String | 空值标记，防穿透（TTL 60s） |

## 5.2 读写策略

```
出价 POST/WS
  → Redis 分布式锁（场次）
  → MySQL 事务（行锁 + 乐观锁 + 幂等 requestId）
  → 提交成功后写穿 Redis（快照 + ZSET 排名，失效 rank_top）
  → WebSocket 广播（排名 DB 回填后写 rank_top）

读快照 GET /snapshot、WS 订阅、倒计时 tick
  → 优先 Redis
  → Miss：singleflight 回源 MySQL，回填缓存
  → remainingMs / serverTimeMs 在读出时按当前时刻重算（避免展示过期）
```

**一致性模型**：以 MySQL 为唯一真相源；Redis 为读优化与推送辅助。出价路径为 **Write-Through**（先 DB 后缓存）。取消/终态后 **Invalidate + Refresh**，避免脏读。

## 5.3 击穿 / 穿透兜底

| 场景 | 策略 |
|------|------|
| 热点房间缓存过期 | `singleflight` 合并回源，仅一次 DB 查询 |
| 不存在 roomId | 写 `null` 标记 60s，快速失败 |
| 排行榜展示 | 出价后失效 `rank_top`；推送时 DB 查 TopN 并回填缓存 |

## 5.6 可观测

`GET /api/v1/metrics` 返回：

- `bidAttempts` / `bidSuccess` / `bidFailures` / `bidFailureRate`
- `cacheHits` / `cacheMisses`
- `wsConnections` / `wsRooms`
