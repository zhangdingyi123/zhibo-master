import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { getOrder } from '../../api/admin'
import type { Order } from '../../api/types'
import { formatCents } from '../../utils/money'
import { StatusBadge } from '../components/StatusBadge'
import { ORDER_STATUS_LABEL } from '../labels'

export function OrderDetailPage() {
  const { id } = useParams()
  const [order, setOrder] = useState<Order | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const oid = Number(id)
    if (!oid) return
    getOrder(oid)
      .then(setOrder)
      .catch((e) => setError(e instanceof Error ? e.message : '加载失败'))
  }, [id])

  if (error) return <p className="form-error">{error}</p>
  if (!order) return <p className="muted">加载中…</p>

  return (
    <div className="admin-page">
      <div className="admin-page__head">
        <div>
          <Link to="/admin/orders" className="breadcrumb">
            ← 订单列表
          </Link>
          <h2>订单详情</h2>
          <code className="order-no">{order.orderNo}</code>
        </div>
        <StatusBadge
          label={ORDER_STATUS_LABEL[order.status]}
          variant="order"
          tone={order.status}
        />
      </div>

      <section className="admin-card">
        <dl className="detail-dl">
          <dt>订单 ID</dt>
          <dd>{order.id}</dd>
          <dt>竞拍场次</dt>
          <dd>
            <Link to={`/admin/products`}>场次 #{order.sessionId}</Link>
          </dd>
          <dt>商品 ID</dt>
          <dd>
            <Link to={`/admin/products/${order.productId}`}>
              商品 #{order.productId}
            </Link>
          </dd>
          <dt>买家</dt>
          <dd>用户 #{order.buyerId}</dd>
          <dt>卖家</dt>
          <dd>用户 #{order.sellerId}</dd>
          <dt>成交金额</dt>
          <dd className="price-lg">{formatCents(order.amount)}</dd>
          <dt>创建时间</dt>
          <dd>{new Date(order.createdAt).toLocaleString('zh-CN')}</dd>
          {order.paidAt && (
            <>
              <dt>支付时间</dt>
              <dd>{new Date(order.paidAt).toLocaleString('zh-CN')}</dd>
            </>
          )}
        </dl>
      </section>
    </div>
  )
}
