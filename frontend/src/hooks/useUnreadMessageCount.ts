import { useCallback, useEffect, useState } from 'react'
import { getUnreadMessageCount } from '../api/messages'
import { isLoggedIn } from '../auth/session'

export function useUnreadMessageCount() {
  const [count, setCount] = useState(0)

  const refresh = useCallback(() => {
    if (!isLoggedIn()) {
      setCount(0)
      return
    }
    getUnreadMessageCount()
      .then((res) => setCount(res.count))
      .catch(() => setCount(0))
  }, [])

  useEffect(() => {
    refresh()
    const id = window.setInterval(refresh, 30_000)
    return () => clearInterval(id)
  }, [refresh])

  return { count, refresh }
}
