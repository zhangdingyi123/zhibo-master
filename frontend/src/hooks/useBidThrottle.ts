import { useCallback, useRef, useState } from 'react'

const DEFAULT_MS = 300

/** 出价按钮防抖，与后端 WS 限流 300ms 对齐 */
export function useBidThrottle(cooldownMs = DEFAULT_MS) {
  const lastAt = useRef(0)
  const [cooling, setCooling] = useState(false)

  const run = useCallback(
    (fn: () => void) => {
      const now = Date.now()
      if (now - lastAt.current < cooldownMs) {
        return false
      }
      lastAt.current = now
      setCooling(true)
      fn()
      window.setTimeout(() => setCooling(false), cooldownMs)
      return true
    },
    [cooldownMs],
  )

  return { run, cooling }
}
