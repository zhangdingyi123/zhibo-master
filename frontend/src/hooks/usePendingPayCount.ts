import { useCallback, useEffect, useState } from 'react'
import { listMyOrders } from '../api/user'
import { isLoggedIn } from '../auth/session'

/** 待支付订单数量，用于 Tab 角标 */
export function usePendingPayCount() {
  const [count, setCount] = useState(0)

  const refresh = useCallback(() => {
    if (!isLoggedIn()) {
      setCount(0)
      return
    }
    listMyOrders({ status: 'pending_pay', page: 1, pageSize: 1 })
      .then((res) => setCount(res.total))
      .catch(() => setCount(0))
  }, [])

  useEffect(() => {
    refresh()
    const id = window.setInterval(refresh, 30_000)
    return () => clearInterval(id)
  }, [refresh])

  return { count, refresh }
}
