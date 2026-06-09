import { useEffect, useState } from 'react'
import { formatRemainingMs, remainingMsTo } from '../../utils/time'

interface Props {
  scheduledStartAt: string
  compact?: boolean
}

/** 预约开拍倒计时卡片 */
export function ScheduledStartBanner({ scheduledStartAt, compact }: Props) {
  const [remainingMs, setRemainingMs] = useState<number | null>(() =>
    remainingMsTo(scheduledStartAt),
  )

  useEffect(() => {
    const tick = () => setRemainingMs(remainingMsTo(scheduledStartAt))
    tick()
    const id = window.setInterval(tick, 1000)
    return () => clearInterval(id)
  }, [scheduledStartAt])

  if (remainingMs == null || remainingMs <= 0) return null

  const urgent = remainingMs <= 60_000

  if (compact) {
    return (
      <span
        className={`scheduled-start scheduled-start--compact${urgent ? ' scheduled-start--urgent' : ''}`}
      >
        预约 {formatRemainingMs(remainingMs)} 后开拍
      </span>
    )
  }

  return (
    <div
      className={`scheduled-start${urgent ? ' scheduled-start--urgent' : ''}`}
      role="status"
    >
      <span className="scheduled-start__badge">预约开拍</span>
      <div className="scheduled-start__body">
        <span className="scheduled-start__label">距开拍</span>
        <strong className="scheduled-start__time">{formatRemainingMs(remainingMs)}</strong>
        <span className="scheduled-start__sub">
          {new Date(scheduledStartAt).toLocaleString('zh-CN', {
            month: 'numeric',
            day: 'numeric',
            hour: '2-digit',
            minute: '2-digit',
          })}{' '}
          开始
        </span>
      </div>
    </div>
  )
}
