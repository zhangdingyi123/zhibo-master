import { useEffect, useRef, useState } from 'react'
import { EventBidNew, type BidNewPayload, type RoomEvent } from '../../ws/types'

type DanmakuItem = {
  id: string
  text: string
  tone: 'bid' | 'system' | 'hot'
}

const MOCK_LINES = [
  '主播这件太值了！',
  '冲冲冲',
  '还有谁',
  '手慢无',
  '加价幅度友好',
]

type Props = {
  lastEvent: RoomEvent | null
  participantCount: number
}

export function LiveDanmaku({ lastEvent, participantCount }: Props) {
  const [items, setItems] = useState<DanmakuItem[]>([])
  const mockIdx = useRef(0)

  useEffect(() => {
    if (!lastEvent?.type) return
    if (lastEvent.type !== EventBidNew) return
    const p = lastEvent.payload as BidNewPayload | undefined
    const amount = p?.bid?.amountCents
    if (amount == null) return
    const yuan = (amount / 100).toFixed(amount % 100 === 0 ? 0 : 2)
    const id = `dm-${lastEvent.seq ?? Date.now()}`
    setItems((prev) => [
      ...prev.slice(-12),
      { id, text: `有人出价 ¥${yuan}`, tone: 'bid' },
    ])
  }, [lastEvent])

  useEffect(() => {
    const t = window.setInterval(() => {
      const line = MOCK_LINES[mockIdx.current % MOCK_LINES.length]!
      mockIdx.current += 1
      const id = `dm-mock-${Date.now()}`
      setItems((prev) => [
        ...prev.slice(-12),
        { id, text: line, tone: participantCount > 20 ? 'hot' : 'system' },
      ])
    }, 10000)
    return () => clearInterval(t)
  }, [participantCount])

  useEffect(() => {
    if (items.length === 0) return
    const t = window.setTimeout(() => {
      setItems((prev) => prev.slice(1))
    }, 7000)
    return () => clearTimeout(t)
  }, [items])

  return (
    <div className="live-danmaku" aria-live="polite" aria-relevant="additions">
      {items.map((item) => (
        <span
          key={item.id}
          className={`live-danmaku__item live-danmaku__item--${item.tone}`}
        >
          {item.text}
        </span>
      ))}
    </div>
  )
}
