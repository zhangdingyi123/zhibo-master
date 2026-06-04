import {
  ClientBid,
  ClientPing,
  ClientSubscribe,
  EventAuctionCancelled,
  EventAuctionExtended,
  EventAuctionSettled,
  EventBidNew,
  EventCountdownTick,
  EventRankUpdate,
  ServerConnected,
  ServerError,
  ServerEvent,
  ServerPong,
  ServerSync,
  WsClientError,
  type BidNewPayload,
  type ConnectedPayload,
  type ConnectionState,
  type Envelope,
  type RankEntry,
  type RankUpdatePayload,
  type RoomEvent,
  type SessionSnapshot,
  type SyncPayload,
  type WsErrorPayload,
} from './types'
import { getOrCreateClientId, getStoredLastSeq, setStoredLastSeq } from './clientId'

const PING_INTERVAL_MS = 25_000
const MAX_BACKOFF_MS = 30_000

export type AuctionSocketAuth = {
  openId?: string | null
  userId?: string | null
  token?: string | null
}

export type AuctionSocketCallbacks = {
  onConnectionChange?: (state: ConnectionState) => void
  onSnapshot?: (snapshot: SessionSnapshot) => void
  onRank?: (items: RankEntry[]) => void
  onRoomEvent?: (event: RoomEvent) => void
  onError?: (err: WsClientError) => void
}

export type AuctionSocketOptions = AuctionSocketAuth &
  AuctionSocketCallbacks & {
    roomId: string
    /** 默认走当前页面 host（Vite 代理 /api） */
    wsBase?: string
    /** 是否持久化 lastSeq 到 localStorage */
    persistLastSeq?: boolean
    /** 是否自动重连 */
    autoReconnect?: boolean
  }

function buildWsUrl(
  roomId: string,
  clientId: string,
  auth: AuctionSocketAuth,
  base?: string,
): string {
  const params = new URLSearchParams({ roomId, clientId })
  if (auth.token) params.set('token', auth.token)
  if (auth.openId) params.set('openId', auth.openId)
  if (auth.userId) params.set('userId', auth.userId)

  if (base) {
    const sep = base.includes('?') ? '&' : '?'
    return `${base}${sep}${params.toString()}`
  }

  const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  return `${proto}//${window.location.host}/api/v1/ws?${params.toString()}`
}

function parsePayload<T>(raw: unknown): T | undefined {
  if (raw == null) return undefined
  return raw as T
}

/**
 * 竞拍房间 WebSocket 客户端（7.3）
 * - 连接 / 重连 / 订阅房间
 * - 心跳 ping + lastSeq
 * - 未登录可围观；bid() 需登录
 */
export class AuctionSocket {
  private readonly roomId: string
  private readonly clientId: string
  private readonly auth: AuctionSocketAuth
  private readonly callbacks: AuctionSocketCallbacks
  private readonly wsBase?: string
  private readonly persistLastSeq: boolean
  private readonly autoReconnect: boolean

  private ws: WebSocket | null = null
  private state: ConnectionState = 'idle'
  private lastSeq = 0
  private snapshot: SessionSnapshot | null = null
  private rank: RankEntry[] = []
  private pingTimer: ReturnType<typeof setInterval> | null = null
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null
  private reconnectAttempt = 0
  private intentionalClose = false

  constructor(options: AuctionSocketOptions) {
    this.roomId = options.roomId
    this.clientId = getOrCreateClientId()
    this.auth = {
      openId: options.openId,
      userId: options.userId,
      token: options.token,
    }
    this.callbacks = {
      onConnectionChange: options.onConnectionChange,
      onSnapshot: options.onSnapshot,
      onRank: options.onRank,
      onRoomEvent: options.onRoomEvent,
      onError: options.onError,
    }
    this.wsBase = options.wsBase
    this.persistLastSeq = options.persistLastSeq ?? true
    this.autoReconnect = options.autoReconnect ?? true

    if (this.persistLastSeq) {
      this.lastSeq = getStoredLastSeq(this.roomId)
    }
  }

  get connectionState(): ConnectionState {
    return this.state
  }

  get currentSnapshot(): SessionSnapshot | null {
    return this.snapshot
  }

  get currentRank(): RankEntry[] {
    return this.rank
  }

  get currentLastSeq(): number {
    return this.lastSeq
  }

  /** 是否已登录，可 WS 出价 */
  get canBid(): boolean {
    return Boolean(
      this.auth.token?.trim() ||
        this.auth.openId?.trim() ||
        this.auth.userId?.trim(),
    )
  }

  connect(): void {
    this.intentionalClose = false
    this.clearReconnectTimer()
    this.openSocket()
  }

  disconnect(): void {
    this.intentionalClose = true
    this.clearReconnectTimer()
    this.stopPing()
    if (this.ws) {
      this.ws.close()
      this.ws = null
    }
    this.setState('closed')
  }

  /** 发送 subscribe（连接后服务端可能已自动订阅，重连时显式发送） */
  resubscribe(): void {
    this.send({
      type: ClientSubscribe,
      clientId: this.clientId,
      roomId: this.roomId,
      lastSeq: this.lastSeq,
      payload: {
        roomId: this.roomId,
        clientId: this.clientId,
        lastSeq: this.lastSeq,
      },
    })
  }

  /**
   * WebSocket 出价（需登录）
   * @throws WsClientError 未登录或连接未就绪
   */
  bid(amount: number, requestId: string): void {
    if (!this.canBid) {
      throw new WsClientError(401, '未登录，无法通过 WebSocket 出价')
    }
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
      throw new WsClientError(400, 'WebSocket 未连接')
    }
    if (!requestId.trim()) {
      throw new WsClientError(400, 'requestId 不能为空')
    }
    this.send({
      type: ClientBid,
      clientId: this.clientId,
      roomId: this.roomId,
      payload: { amount, requestId },
    })
  }

  private openSocket(): void {
    if (this.ws?.readyState === WebSocket.OPEN) return

    this.setState(this.reconnectAttempt > 0 ? 'reconnecting' : 'connecting')

    const url = buildWsUrl(this.roomId, this.clientId, this.auth, this.wsBase)
    const ws = new WebSocket(url)
    this.ws = ws

    ws.onopen = () => {
      this.reconnectAttempt = 0
      this.setState('connected')
      this.startPing()
      // Query 已带 roomId 时服务端会自动 subscribe；重连时补发以确保 lastSeq
      if (this.lastSeq > 0) {
        this.resubscribe()
      }
    }

    ws.onmessage = (ev) => {
      this.handleMessage(ev.data)
    }

    ws.onerror = () => {
      this.emitError(0, 'WebSocket 连接异常')
    }

    ws.onclose = () => {
      this.stopPing()
      this.ws = null
      if (this.intentionalClose) {
        this.setState('closed')
        return
      }
      this.setState('reconnecting')
      if (this.autoReconnect) {
        this.scheduleReconnect()
      } else {
        this.setState('closed')
      }
    }
  }

  private scheduleReconnect(): void {
    this.clearReconnectTimer()
    const delay = Math.min(1000 * 2 ** this.reconnectAttempt, MAX_BACKOFF_MS)
    this.reconnectAttempt++
    this.reconnectTimer = setTimeout(() => {
      this.openSocket()
    }, delay)
  }

  private clearReconnectTimer(): void {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer)
      this.reconnectTimer = null
    }
  }

  private startPing(): void {
    this.stopPing()
    this.pingTimer = setInterval(() => this.sendPing(), PING_INTERVAL_MS)
  }

  private stopPing(): void {
    if (this.pingTimer) {
      clearInterval(this.pingTimer)
      this.pingTimer = null
    }
  }

  private sendPing(): void {
    this.send({
      type: ClientPing,
      clientId: this.clientId,
      roomId: this.roomId,
      lastSeq: this.lastSeq,
      payload: {
        clientId: this.clientId,
        lastSeq: this.lastSeq,
      },
    })
  }

  private send(env: Envelope): void {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) return
    this.ws.send(JSON.stringify(env))
  }

  private handleMessage(raw: string): void {
    let env: Envelope
    try {
      env = JSON.parse(raw) as Envelope
    } catch {
      return
    }

    switch (env.type) {
      case ServerConnected:
        this.handleConnected(parsePayload<ConnectedPayload>(env.payload))
        break
      case ServerSync:
        this.handleSync(parsePayload<SyncPayload>(env.payload))
        break
      case ServerEvent:
        this.handleServerEvent(env)
        break
      case ServerPong:
        break
      case ServerError: {
        const p = parsePayload<WsErrorPayload>(env.payload)
        if (p) this.emitError(p.code, p.message)
        break
      }
      default:
        break
    }
  }

  private handleConnected(p?: ConnectedPayload): void {
    if (p?.currentSeq != null) {
      this.updateLastSeq(p.currentSeq)
    }
  }

  private handleSync(p?: SyncPayload): void {
    if (!p) return
    if (p.snapshot) {
      this.applySnapshot(p.snapshot)
    }
    for (const ev of p.events ?? []) {
      this.applyRoomEvent(ev)
    }
  }

  private handleServerEvent(env: Envelope): void {
    const roomEv = parsePayload<RoomEvent>(env.payload)
    if (!roomEv) return
    if (env.seq != null && env.seq > 0) {
      this.updateLastSeq(env.seq)
    } else if (roomEv.seq != null && roomEv.seq > 0) {
      this.updateLastSeq(roomEv.seq)
    }
    this.applyRoomEvent(roomEv)
  }

  private applyRoomEvent(ev: RoomEvent): void {
    this.callbacks.onRoomEvent?.(ev)

    switch (ev.type) {
      case EventBidNew: {
        const p = parsePayload<BidNewPayload>(ev.payload)
        if (p?.snapshot) this.applySnapshot(p.snapshot)
        break
      }
      case EventRankUpdate: {
        const p = parsePayload<RankUpdatePayload>(ev.payload)
        if (p?.items) {
          this.rank = p.items
          this.callbacks.onRank?.(p.items)
        }
        break
      }
      case EventCountdownTick:
        this.applySnapshot(parsePayload<SessionSnapshot>(ev.payload))
        break
      case EventAuctionExtended:
      case EventAuctionSettled:
      case EventAuctionCancelled: {
        const p = ev.payload as { snapshot?: SessionSnapshot } | undefined
        if (p?.snapshot) this.applySnapshot(p.snapshot)
        break
      }
      default:
        break
    }
  }

  private applySnapshot(snap: SessionSnapshot | undefined): void {
    if (!snap) return
    this.snapshot = snap
    this.callbacks.onSnapshot?.(snap)
  }

  private updateLastSeq(seq: number): void {
    if (seq <= this.lastSeq) return
    this.lastSeq = seq
    if (this.persistLastSeq) {
      setStoredLastSeq(this.roomId, seq)
    }
  }

  private setState(state: ConnectionState): void {
    if (this.state === state) return
    this.state = state
    this.callbacks.onConnectionChange?.(state)
  }

  private emitError(code: number, message: string): void {
    this.callbacks.onError?.(new WsClientError(code, message))
  }
}
