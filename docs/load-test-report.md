# 压测报告模板（5.4）

## 环境

| 项 | 值 |
|----|-----|
| 日期 | YYYY-MM-DD |
| 服务 | `http://localhost:8081` |
| Redis | 是 / 否 |
| 场次 ID | `SESSION_ID` |
| 场次状态 | running |

## 并发出价（目标：100+）

```bash
cd backend/scripts
CONCURRENCY=120 SESSION_ID=1 BASE_URL=http://localhost:8081 ./bid_stress.sh
```

### hey 结果摘要

| 指标 | 值 |
|------|-----|
| 总请求 | |
| 成功 2xx | |
| 4xx（价低/限流/锁忙） | |
| 5xx | |
| RPS | |
| P99 延迟 | |

### 业务校验

- [ ] MySQL `auction_sessions.current_price` 与最高出价一致
- [ ] `bids` 无重复 `request_id`
- [ ] `GET /api/v1/metrics` 中 `bidFailureRate` 可接受（<5% 为演示参考）
- [ ] Redis 快照与 DB 一致（抽样对比）

## 快照读压（可选）

```bash
hey -n 5000 -c 100 "http://localhost:8081/api/v1/rooms/room_sess_1/snapshot"
```

对比开启 Redis 前后 DB QPS（`SHOW GLOBAL STATUS LIKE 'Questions'`）。

## 结论

（填写：是否满足单房间 100+ 并发、瓶颈在锁/DB/网络、后续优化点）
