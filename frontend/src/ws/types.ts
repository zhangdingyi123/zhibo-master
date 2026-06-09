/** 与服务端 ws/message.go 对齐 */

export const ClientSubscribe = 'subscribe'
export const ClientPing = 'ping'
export const ClientBid = 'bid'

export const ServerConnected = 'connected'
export const ServerSync = 'sync'
export const ServerEvent = 'event'
export const ServerPong = 'pong'
export const ServerError = 'error'

export const EventBidNew = 'bid.new'
export const EventRankUpdate = 'rank.update'
export const EventCountdownTick = 'countdown.tick'
export const EventAuctionExtended = 'auction.extended'
export const EventAuctionSettled = 'auction.settled'
export const EventAuctionCancelled = 'auction.cancelled'
export const EventSessionSwitch = 'session.switch'

export type ConnectionState =
  | 'idle'
  | 'connecting'
  | 'connected'
  | 'reconnecting'
  | 'closed'

export type SessionStatus =
  | 'pending'
  | 'running'
  | 'settled'
  | 'cancelled'
  | 'failed'

export type AuctionRules = {
  startingPrice: number
  bidIncrement: number
  capPrice?: number
  durationSec: number
  extendThresholdSec?: number
  extendSec?: number
}

export type SessionSnapshot = {
  sessionId: number
  roomId: string
  status: SessionStatus
  currentPrice: number
  bidCount: number
  participantCount: number
  minNextBid: number
  rules: AuctionRules
  endAtMs?: number
  remainingMs: number
  serverTimeMs: number
  winnerId?: number
}

export type Bid = {
  id: number
  sessionId: number
  userId: number
  amount: number
  requestId: string
  seq: number
  isWinning: boolean
  createdAt: string
}

export type RankEntry = {
  userId: number
  nickname: string
  avatar: string
  amount: number
  seq: number
  rank: number
}

export type Envelope<T = unknown> = {
  type: string
  clientId?: string
  roomId?: string
  seq?: number
  lastSeq?: number
  ts?: number
  payload?: T
}

export type RoomEvent<T = unknown> = {
  seq?: number
  type: string
  ts: number
  payload?: T
}

export type ConnectedPayload = {
  roomId: string
  sessionId: number
  currentSeq: number
  userId?: number
}

export type SyncPayload = {
  snapshot: SessionSnapshot
  events: RoomEvent[]
}

export type BidNewPayload = {
  bid: Bid
  snapshot: SessionSnapshot
}

export type RankUpdatePayload = {
  items: RankEntry[]
}

export type SettledPayload = {
  session: {
    id: number
    winnerId?: number
    currentPrice: number
    roomId: string
  }
  snapshot: SessionSnapshot
  order?: {
    id: number
    orderNo: string
    sessionId: number
    amount: number
    status: string
  }
}

export type ExtendedPayload = {
  snapshot: SessionSnapshot
  previousEndAtMs: number
  newEndAtMs: number
}

export type CancelledPayload = {
  session: { id: number }
  snapshot: SessionSnapshot
  reason: string
}

export type SessionSummaryPayload = {
  sessionId: number
  productId: number
  productName: string
  coverUrl: string
  status: SessionStatus
  finalPrice: number
  winnerId?: number
  seqInRoom: number
}

export type SessionSwitchPayload = {
  liveRoomId: number
  roomId: string
  previous?: SessionSummaryPayload
  current?: {
    session: AuctionSession
    product: { id: number; name: string; description?: string; coverUrl: string }
    snapshot: SessionSnapshot
  }
  history: SessionSummaryPayload[]
}

export type AuctionSession = {
  id: number
  productId: number
  roomId: string
  status: SessionStatus
  currentPrice: number
}

export type WsErrorPayload = {
  code: number
  message: string
}

export class WsClientError extends Error {
  code: number
  constructor(code: number, message: string) {
    super(message)
    this.code = code
    this.name = 'WsClientError'
  }
}
