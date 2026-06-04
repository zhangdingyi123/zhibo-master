import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { getAuction, listMyOrders } from '../../api/user'
import type { Order } from '../../api/types'
import type { ProductBrief } from '../../api/user'
import { ORDER_STATUS_LABEL } from '../../admin/labels'
import { isLoggedIn } from '../../auth/session'
import { formatCents } from '../../utils/money'

type OrderRow = {
  order: Order
  product: ProductBrief | null
}

export function MyOrdersPage() {
  const loggedIn = isLoggedIn()
  const [rows, setRows] = useState<OrderRow[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!loggedIn) {
      setLoading(false)
      return
    }
    let cancelled = false
    listMyOrders({ page: 1, pageSize: 50 })
      .then(async (res) => {
        const enriched = await Promise.all(
          res.items.map(async (order) => {
            try {
              const detail = await getAuction(order.sessionId)
              return {
                order,
                product: {
                  id: detail.product.id,
                  name: detail.product.name,
                  description: detail.product.description,
                  coverUrl: detail.product.coverUrl,
                },
              }
            } catch {
              return { order, product: null }
            }
          }),
        )
        if (!cancelled) setRows(enriched)
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
        <p className="page-desc">中标后订单将自动生成，请及时完成支付</p>
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
