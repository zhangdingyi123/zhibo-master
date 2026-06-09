import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { listMyOrders } from '../../api/user'
import type { OrderListItem } from '../../api/types'
import { ORDER_STATUS_LABEL } from '../../admin/labels'
import { isLoggedIn } from '../../auth/session'
import { formatCents } from '../../utils/money'

export function MyOrdersPage() {
  const loggedIn = isLoggedIn()
  const [rows, setRows] = useState<OrderListItem[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!loggedIn) {
      setLoading(false)
      return
    }
    let cancelled = false
    listMyOrders({ page: 1, pageSize: 50 })
      .then((res) => {
        if (!cancelled) setRows(res.items)
      })
      .catch((e) => {
        if (!cancelled) setError(e instanceof Error ? e.message : '加载失败')
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [loggedIn])

  if (!loggedIn) {
    return (
      <div className="user-page">
        <header className="user-page__head">
          <h1>我的订单</h1>
        </header>
        <p className="user-hint">请先登录后查看订单</p>
        <Link to="/app/login" className="btn-primary">
          去登录
        </Link>
      </div>
    )
  }

  return (
    <div className="user-page">
      <header className="user-page__head">
        <h1>我的订单</h1>
        <p className="page-desc">中标后订单将自动生成，请在 30 分钟内完成支付</p>
      </header>

      {loading && <p className="user-hint">加载中…</p>}
      {error && <p className="user-error">{error}</p>}

      <ul className="order-list">
        {rows.map(({ order, product }) => (
          <li key={order.id}>
            <Link to={`/app/orders/${order.id}`} className="order-row order-row--rich">
              {product?.coverUrl ? (
                <img src={product.coverUrl} alt="" className="order-row__thumb" />
              ) : (
                <div className="order-row__thumb order-row__thumb--placeholder" aria-hidden />
              )}
              <div className="order-row__body">
                <h2 className="order-row__title">
                  {product?.name ?? `场次 #${order.sessionId}`}
                </h2>
                <span className="order-no">{order.orderNo}</span>
                <span className={`badge badge--${order.status}`}>
                  {ORDER_STATUS_LABEL[order.status]}
                </span>
              </div>
              <div className="order-row__aside">
                <strong>{formatCents(order.amount)}</strong>
                {order.status === 'pending_pay' && (
                  <span className="order-row__pay-hint">去支付</span>
                )}
                {order.status === 'paid' && (
                  <span className="order-row__pay-hint">填地址</span>
                )}
                {order.status === 'shipped' && (
                  <span className="order-row__pay-hint">确认收货</span>
                )}
                {(order.status === 'cancelled' || order.status === 'refunded') && (
                  <span className="order-row__pay-hint order-row__pay-hint--muted">
                    查看详情
                  </span>
                )}
              </div>
            </Link>
          </li>
        ))}
      </ul>

      {!loading && rows.length === 0 && (
        <p className="user-hint">暂无订单，中标后将自动生成</p>
      )}
    </div>
  )
}
