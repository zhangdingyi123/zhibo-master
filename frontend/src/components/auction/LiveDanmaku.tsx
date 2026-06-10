import { useEffect, useState } from 'react'
import {
  EventBidNew,
  EventCommentNew,
  type BidNewPayload,
  type RoomCommentPayload,
  type RoomEvent,
} from '../../ws/types'

type DanmakuItem = {
  id: string
  text: string
  tone: 'bid' | 'comment' | 'hot'
  author?: string
}

type Props = {
  lastEvent: RoomEvent | null
  seedComments?: { id: number; nickname: string; content: string }[]
}

export function LiveDanmaku({ lastEvent, seedComments = [] }: Props) {
  const [items, setItems] = useState<DanmakuItem[]>([])

  useEffect(() => {
    if (seedComments.length === 0) return
    setItems(
      seedComments.slice(-8).map((c) => ({
        id: `seed-${c.id}`,
        text: c.content,
        tone: 'comment' as const,
        author: c.nickname,
      })),
    )
  }, [seedComments])

  useEffect(() => {
    if (!lastEvent?.type) return

    if (lastEvent.type === EventBidNew) {
      const p = lastEvent.payload as BidNewPayload | undefined
      const amount = p?.bid?.amount
      if (amount == null) return
      const yuan = (amount / 100).toFixed(amount % 100 === 0 ? 0 : 2)
      const id = `dm-bid-${lastEvent.seq ?? Date.now()}`
      setItems((prev) => [
        ...prev.slice(-14),
        { id, text: `有人出价 ¥${yuan}`, tone: 'bid' },
      ])
      return
    }

    if (lastEvent.type === EventCommentNew) {
      const p = lastEvent.payload as RoomCommentPayload | undefined
      const c = p?.comment
      if (!c?.content) return
      const id = `dm-c-${c.id}`
      setItems((prev) => [
        ...prev.slice(-14),
        {
          id,
          text: c.content,
          tone: 'comment',
          author: c.nickname,
        },
      ])
    }
  }, [lastEvent])

  useEffect(() => {
    if (items.length === 0) return
    const t = window.setTimeout(() => {
      setItems((prev) => prev.slice(1))
    }, 8000)
    return () => clearTimeout(t)
  }, [items])

  return (
    <div className="live-danmaku" aria-live="polite" aria-relevant="additions">
      {items.map((item) => (
        <span
          key={item.id}
          className={`live-danmaku__item live-danmaku__item--${item.tone}`}
        >
          {item.author ? (
            <>
              <strong className="live-danmaku__author">{item.author}</strong>
              {item.text}
            </>
          ) : (
            item.text
          )}
        </span>
      ))}
    </div>
  )
}
