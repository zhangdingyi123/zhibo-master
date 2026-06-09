import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { cancelOrder, getOrder, refundOrder, shipOrder } from '../../api/admin'
import type { Order } from '../../api/types'
import { formatCents } from '../../utils/money'
import { AftersaleForm } from '../components/AftersaleForm'
import { StatusBadge } from '../components/StatusBadge'
import { ORDER_STATUS_LABEL } from '../labels'

function hasAddress(order: Order) {
  return Boolean(
    order.receiverName?.trim() &&
      order.receiverPhone?.trim() &&
      order.receiverAddress?.trim(),
  )
}

export function OrderDetailPage() {
  const { id } = useParams()
  const [order, setOrder] = useState<Order | null>(null)
  const [trackingNo, setTrackingNo] = useState('')
  const [shipping, setShipping] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const oid = Number(id)
    if (!oid) return
    getOrder(oid)
      .then((o) => {
        setOrder(o)
        setTrackingNo(o.trackingNo ?? '')
      })
      .catch((e) => setError(e instanceof Error ? e.message : '加载失败'))
  }, [id])

  async function handleShip() {
    if (!order) return
    setShipping(true)
    setError(null)
    try {
      const updated = await shipOrder(order.id, trackingNo)
      setOrder(updated)
    } catch (e) {
      setError(e instanceof Error ? e.message : '发货失败')
    } finally {
      setShipping(false)
    }
  }

  if (error && !order) return <p className="form-error">{error}</p>
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

      {hasAddress(order) && (
        <section className="admin-card">
          <h3>收货信息</h3>
          <dl className="detail-dl">
            <dt>收货人</dt>
            <dd>{order.receiverName}</dd>
            <dt>手机号</dt>
            <dd>{order.receiverPhone}</dd>
            <dt>地址</dt>
            <dd>{order.receiverAddress}</dd>
          </dl>
        </section>
      )}

      {order.status === 'paid' && (
        <section className="admin-card">
          <h3>发货</h3>
          {!hasAddress(order) ? (
            <p className="muted">买家尚未填写收货地址，暂不可发货。</p>
          ) : (
            <>
              <label className="form-field">
                <span>物流单号（可选）</span>
                <input
                  value={trackingNo}
                  onChange={(e) => setTrackingNo(e.target.value)}
                  placeholder="SF1234567890"
                />
              </label>
              <button
                type="button"
                className="btn-primary"
                disabled={shipping}
                onClick={() => void handleShip()}
              >
                {shipping ? '提交中…' : '确认发货'}
              </button>
            </>
          )}
        </section>
      )}

      {(order.status === 'shipped' || order.status === 'completed') && (
        <section className="admin-card">
          <h3>物流</h3>
          <dl className="detail-dl">
            {order.trackingNo && (
              <>
                <dt>物流单号</dt>
                <dd>{order.trackingNo}</dd>
              </>
            )}
            {order.shippedAt && (
              <>
                <dt>发货时间</dt>
                <dd>{new Date(order.shippedAt).toLocaleString('zh-CN')}</dd>
              </>
            )}
            {order.completedAt && (
              <>
                <dt>完成时间</dt>
                <dd>{new Date(order.completedAt).toLocaleString('zh-CN')}</dd>
              </>
            )}
          </dl>
        </section>
      )}

      {order.status === 'pending_pay' && (
        <section className="admin-card admin-card--warn">
          <h3>售后 · 取消待支付</h3>
          <AftersaleForm
            actionLabel="取消订单"
            busyLabel="处理中…"
            hint="适用于误拍、协商放弃等场景。取消后买家将收到站内通知。"
            onSubmit={async (reason) => {
              const updated = await cancelOrder(order.id, reason)
              setOrder(updated)
            }}
          />
        </section>
      )}

      {(order.status === 'paid' || order.status === 'shipped') && (
        <section className="admin-card admin-card--warn">
          <h3>售后 · 模拟退款</h3>
          <AftersaleForm
            actionLabel="确认退款"
            busyLabel="退款中…"
            hint={
              order.status === 'paid'
                ? '未发货订单可原路退款（演示环境模拟）。'
                : '已发货订单售后退款（演示环境模拟，无需退货入库）。'
            }
            onSubmit={async (reason) => {
              const updated = await refundOrder(order.id, reason)
              setOrder(updated)
            }}
          />
        </section>
      )}

      {(order.status === 'cancelled' || order.status === 'refunded') && order.cancelReason && (
        <section className="admin-card">
          <h3>售后记录</h3>
          <dl className="detail-dl">
            <dt>原因</dt>
            <dd>{order.cancelReason}</dd>
            {order.cancelledBy && (
              <>
                <dt>操作方</dt>
                <dd>{order.cancelledBy === 'seller' ? '主播' : order.cancelledBy === 'buyer' ? '买家' : '系统'}</dd>
              </>
            )}
            {order.cancelledAt && (
              <>
                <dt>取消时间</dt>
                <dd>{new Date(order.cancelledAt).toLocaleString('zh-CN')}</dd>
              </>
            )}
            {order.refundedAt && (
              <>
                <dt>退款时间</dt>
                <dd>{new Date(order.refundedAt).toLocaleString('zh-CN')}</dd>
              </>
            )}
          </dl>
        </section>
      )}

      {error && <p className="form-error">{error}</p>}
    </div>
  )
}
