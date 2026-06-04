import { useCallback, useEffect, useRef, useState } from 'react'
import { AuctionSocket, type AuctionSocketOptions } from './auctionSocket'
import type {
  ConnectionState,
  RankEntry,
  RoomEvent,
  SessionSnapshot,
} from './types'
import { WsClientError } from './types'

export type UseAuctionSocketOptions = Omit<
  AuctionSocketOptions,
  'onConnectionChange' | 'onSnapshot' | 'onRank' | 'onRoomEvent' | 'onError'
> & {
  /** 为 false 时不自动 connect（默认 true） */
  enabled?: boolean
}

export type UseAuctionSocketResult = {
  connectionState: ConnectionState
  snapshot: SessionSnapshot | null
  rank: RankEntry[]
  lastSeq: number
  /** 已 Mock 登录，可 WS 出价 */
  canBid: boolean
  lastEvent: RoomEvent | null
  lastError: WsClientError | null
  isBidding: boolean
  bid: (amount: number, requestId?: string) => void
  reconnect: () => void
  disconnect: () => void
}

export function useAuctionSocket(
  options: UseAuctionSocketOptions,
): UseAuctionSocketResult {
  const { enabled = true, roomId, openId, userId, token, ...rest } = options

  const [connectionState, setConnectionState] =
    useState<ConnectionState>('idle')
  const [snapshot, setSnapshot] = useState<SessionSnapshot | null>(null)
  const [rank, setRank] = useState<RankEntry[]>([])
  const [lastSeq, setLastSeq] = useState(0)
  const [lastEvent, setLastEvent] = useState<RoomEvent | null>(null)
  const [lastError, setLastError] = useState<WsClientError | null>(null)
  const [isBidding, setIsBidding] = useState(false)

  const socketRef = useRef<AuctionSocket | null>(null)

  const canBid = Boolean(token?.trim() || openId?.trim() || userId?.trim())

  useEffect(() => {
    if (!enabled || !roomId) {
      socketRef.current?.disconnect()
      socketRef.current = null
      setConnectionState('idle')
      return
    }

    const socket = new AuctionSocket({
      roomId,
      openId,
      userId,
      token,
      ...rest,
      onConnectionChange: setConnectionState,
      onSnapshot: setSnapshot,
      onRank: setRank,
      onRoomEvent: (ev) => {
        setLastEvent(ev)
        setLastSeq(socket.currentLastSeq)
        if (ev.type === 'bid.new') {
          setIsBidding(false)
        }
      },
      onError: (err) => {
        setIsBidding(false)
        setLastError(err)
      },
    })

    socketRef.current = socket
    socket.connect()

    return () => {
      socket.disconnect()
      socketRef.current = null
    }
  }, [enabled, roomId, openId, userId, token])

  const bid = useCallback(
    (amount: number, requestId?: string) => {
      const id =
        requestId ??
        `bid-${crypto.randomUUID?.() ?? `${Date.now()}-${Math.random()}`}`
      try {
        setIsBidding(true)
        socketRef.current?.bid(amount, id)
        setLastError(null)
      } catch (e) {
        setIsBidding(false)
        setLastError(
          e instanceof WsClientError
            ? e
            : new WsClientError(0, e instanceof Error ? e.message : '出价失败'),
        )
      }
    },
    [],
  )

  const reconnect = useCallback(() => {
    const s = socketRef.current
    if (!s) return
    s.disconnect()
    s.connect()
  }, [])

  const disconnect = useCallback(() => {
    socketRef.current?.disconnect()
  }, [])

  return {
    connectionState,
    snapshot,
    rank,
    lastSeq,
    canBid,
    lastEvent,
    lastError,
    isBidding,
    bid,
    reconnect,
    disconnect,
  }
}
