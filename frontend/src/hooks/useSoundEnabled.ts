import { useCallback, useState } from 'react'

const STORAGE_KEY = 'zhibo-auction-sound'

export function useSoundEnabled() {
  const [enabled, setEnabled] = useState(() => {
    if (typeof window === 'undefined') return true
    return localStorage.getItem(STORAGE_KEY) !== '0'
  })

  const toggle = useCallback(() => {
    setEnabled((prev) => {
      const next = !prev
      localStorage.setItem(STORAGE_KEY, next ? '1' : '0')
      return next
    })
  }, [])

  return { soundEnabled: enabled, toggleSound: toggle }
}
