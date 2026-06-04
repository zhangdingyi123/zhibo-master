import { useEffect, useState } from 'react'
import { remainingMsFromEndAt } from '../utils/time'

/** 每秒刷新，用于列表卡片等静态 endAt 倒计时 */
export function useCountdown(endAt: string | undefined, active: boolean) {
  const [remainingMs, setRemainingMs] = useState<number | null>(() =>
    active ? remainingMsFromEndAt(endAt) : null,
  )

  useEffect(() => {
    if (!active || !endAt) {
      setRemainingMs(null)
      return
    }
    const tick = () => setRemainingMs(remainingMsFromEndAt(endAt))
    tick()
    const id = window.setInterval(tick, 1000)
    return () => clearInterval(id)
  }, [endAt, active])

  return remainingMs
}
