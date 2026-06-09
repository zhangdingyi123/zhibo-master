import { useCallback, useEffect, useRef, useState } from 'react'
import {
  EventAuctionCancelled,
  EventAuctionExtended,
  EventAuctionSettled,
  EventBidNew,
  EventCountdownTick,
  EventRankUpdate,
  EventSessionSwitch,
  type BidNewPayload,
  type RankEntry,
  type RoomEvent,
  type SettledPayload,
} from '../ws/types'

export type ToastKind = 'info' | 'success' | 'warn' | 'error'

export type AuctionToast = {
  id: string
  kind: ToastKind
  title: string
  message?: string
}

type Options = {
  currentUserId: number | null
  rank: RankEntry[]
  lastEvent: RoomEvent | null
  soundEnabled?: boolean
  /** 连播房间：成交不跳页，提示更轻 */
  multiSku?: boolean
  /** 成交后回调（含订单 id） */
  onSettled?: (payload: SettledPayload) => void
}

function playTone(kind: ToastKind, enabled: boolean) {
  if (!enabled) return
  try {
    const ctx = new AudioContext()
    const osc = ctx.createOscillator()
    const gain = ctx.createGain()
    osc.connect(gain)
    gain.connect(ctx.destination)
    osc.frequency.value = kind === 'warn' ? 440 : kind === 'success' ? 660 : 520
    gain.gain.value = 0.06
    osc.start()
    osc.stop(ctx.currentTime + 0.12)
    osc.onended = () => void ctx.close()
  } catch {
    /* 静默忽略 */
  }
}

export function useAuctionNotifications({
  currentUserId,
  rank,
  lastEvent,
  soundEnabled = true,
  multiSku = false,
  onSettled,
}: Options) {
  const [toasts, setToasts] = useState<AuctionToast[]>([])
  const [outbidFlash, setOutbidFlash] = useState(false)
  const prevRankRef = useRef<number | null>(null)
  const prevLeaderRef = useRef<number | null>(null)
  const handledSeqRef = useRef(0)

  const push = useCallback((toast: Omit<AuctionToast, 'id'>, sound?: ToastKind) => {
    const id = `toast-${Date.now()}-${Math.random().toString(36).slice(2, 7)}`
    setToasts((prev) => [...prev.slice(-4), { ...toast, id }])
    if (sound) playTone(sound, soundEnabled)
  }, [soundEnabled])

  const dismiss = useCallback((id: string) => {
    setToasts((prev) => prev.filter((t) => t.id !== id))
  }, [])

  useEffect(() => {
    if (!currentUserId) {
      prevRankRef.current = null
      return
    }
    const me = rank.find((r) => r.userId === currentUserId)
    const newRank = me?.rank ?? null
    const leaderId = rank[0]?.userId ?? null

    if (
      prevRankRef.current != null &&
      newRank != null &&
      newRank > prevRankRef.current
    ) {
      push(
        {
          kind: 'warn',
          title: '已被超越',
          message: `您的排名降至第 ${newRank} 名`,
        },
        'warn',
      )
      setOutbidFlash(true)
      window.setTimeout(() => setOutbidFlash(false), 600)
    }

    if (
      prevLeaderRef.current === currentUserId &&
      leaderId != null &&
      leaderId !== currentUserId
    ) {
      push(
        {
          kind: 'warn',
          title: '领先位被抢',
          message: '快出一手夺回领先！',
        },
        'warn',
      )
      setOutbidFlash(true)
      window.setTimeout(() => setOutbidFlash(false), 600)
    }

    prevRankRef.current = newRank
    prevLeaderRef.current = leaderId
  }, [rank, currentUserId, push])

  useEffect(() => {
    if (!lastEvent?.seq || lastEvent.seq <= handledSeqRef.current) return
    if (lastEvent.type === EventCountdownTick) return

    handledSeqRef.current = lastEvent.seq

    const payload = lastEvent.payload as Record<string, unknown> | undefined

    switch (lastEvent.type) {
      case EventBidNew: {
        const p = payload as BidNewPayload | undefined
        const bidUser = p?.bid?.userId
        if (
          currentUserId &&
          bidUser &&
          bidUser !== currentUserId &&
          prevLeaderRef.current === currentUserId
        ) {
          push(
            {
              kind: 'warn',
              title: '有人出价了',
              message: '您的领先优势受到挑战',
            },
            'warn',
          )
        }
        break
      }
      case EventRankUpdate:
        break
      case EventAuctionExtended:
        push(
          {
            kind: 'info',
            title: '竞拍延时',
            message: '结束前有人出价，倒计时已延长',
          },
          'info',
        )
        break
      case EventAuctionSettled: {
        const p = payload as SettledPayload | undefined
        const winnerId = p?.session?.winnerId ?? p?.snapshot?.winnerId
        const isWinner = currentUserId != null && winnerId === currentUserId
        if (multiSku) {
          // 连播：非中标者不弹 Toast，中标者由底部支付条承接
          if (isWinner) {
            playTone('success', soundEnabled)
          }
        } else {
          push(
            {
              kind: isWinner ? 'success' : 'info',
              title: '竞拍结束',
              message: isWinner ? '恭喜您中标！' : '本场竞拍已成交',
            },
            isWinner ? 'success' : 'info',
          )
        }
        if (p) onSettled?.(p)
        break
      }
      case EventSessionSwitch:
        // 连播切品引导由 NextUpBanner 承接，避免 Toast 叠层
        break
      case EventAuctionCancelled:
        push(
          {
            kind: 'error',
            title: '竞拍已取消',
            message:
              (payload?.reason as string | undefined) || '主播已取消本场竞拍',
          },
          'error',
        )
        break
      default:
        break
    }
  }, [lastEvent, currentUserId, multiSku, push, onSettled])

  return { toasts, dismiss, outbidFlash }
}
