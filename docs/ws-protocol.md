# WebSocket 协议（阶段 4）

连接地址（开发）：

```
ws://localhost:8081/api/v1/ws?roomId=room_sess_1&clientId=uuid-001&lastSeq=0&openId=buyer_001
```

经 Vite 代理：

```
ws://localhost:5173/api/v1/ws?roomId=room_sess_1&clientId=uuid-001&openId=buyer_001
```

## 鉴权（4.2）

浏览器 WebSocket 无法自定义 Header，使用 Query（与 REST Mock 一致）：

| 参数 | 说明 |
|------|------|
| `openId` | 如 `buyer_001`、`anchor_001` |
| `userId` | 数字用户 ID |

也可在升级请求中带 `X-Mock-Open-Id` / `X-User-Id`。未登录可围观，但不能通过 WS 出价。

## 消息信封

```json
{
  "type": "subscribe",
  "clientId": "uuid-001",
  "roomId": "room_sess_1",
  "lastSeq": 0,
  "ts": 1716700000000,
  "payload": { }
}
```

## 客户端 → 服务端

| type | 说明 |
|------|------|
| `subscribe` | 加入房间；`payload.roomId`、`clientId`、`lastSeq` |
| `ping` | 心跳；`payload.clientId`、`lastSeq` |
| `bid` | 出价（需登录）；`payload.amount`、`requestId` |

`subscribe` 示例：

```json
{
  "type": "subscribe",
  "clientId": "uuid-001",
  "payload": {
    "roomId": "room_sess_1",
    "clientId": "uuid-001",
    "lastSeq": 12
  }
}
```

## 服务端 → 客户端

| type | 说明 |
|------|------|
| `connected` | 订阅成功，含 `currentSeq` |
| `sync` | 快照 + 增量事件（重连补偿） |
| `event` | 房间事件（含 `seq`） |
| `pong` | 心跳响应 |
| `error` | 错误 |

## 房间事件（4.4）

`event` 的 `payload` 内层：

| event.type | 触发时机 |
|------------|----------|
| `bid.new` | 新出价 |
| `rank.update` | 排行榜 Top10 更新 |
| `countdown.tick` | 权威倒计时（200ms，不计 seq） |
| `auction.extended` | 结束前延时 |
| `auction.settled` | 成交 |
| `auction.cancelled` | 主播取消 |

## 心跳与重连（4.3 / 4.6）

1. 客户端生成稳定 `clientId`（localStorage）。
2. 每 25s 发 `ping`，携带 `lastSeq`。
3. 断线重连：`lastSeq` 传入 `subscribe`；服务端返回 `sync`：
   - `snapshot`：与 `GET /api/v1/rooms/:roomId/snapshot` 一致
   - `events`：`seq > lastSeq` 的增量列表

## 出价限流（4.7）

同一用户 WS 出价最小间隔 **300ms**，过快返回 `error` code `429`。

## 权威倒计时（4.5）

`countdown.tick` 由服务端每 200ms 推送 `SessionSnapshot`（含 `remainingMs`、`serverTimeMs`）。客户端仅展示，不以本地计时为准。

## 前端封装（7.3）

| 模块 | 说明 |
|------|------|
| `frontend/src/ws/auctionSocket.ts` | `AuctionSocket` 类：连接、重连、ping、subscribe、`bid()` |
| `frontend/src/ws/useAuctionSocket.ts` | React Hook：`useAuctionSocket({ roomId, openId })` |
| `frontend/src/auth/mockAuth.ts` | Mock 登录态（localStorage） |

未传 `openId` / `userId` 时为围观；`bid()` 在客户端直接拒绝（401），与服务端一致。
