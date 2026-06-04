import { useCallback, useEffect, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { getAuction, getOrder, mockPayOrder } from '../../api/user'
import type { Order } from '../../api/types'
import type { ProductBrief } from '../../api/user'
import { ORDER_STATUS_LABEL } from '../../admin/labels'
import { isLoggedIn } from '../../auth/session'
import { formatCents } from '../../utils/money'

export function OrderPayPage() {
  const { orderId } = useParams<{ orderId: string }>()
  const navigate = useNavigate()
  const [order, setOrder] = useState<Order | null>(null)
  const [product, setProduct] = useState<ProductBrief | null>(null)
  const [paying, setPaying] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const id = Number(orderId)
    if (!Number.isFinite(id) || !isLoggedIn()) return
    let cancelled = false
    getOrder(id)
      .then((o) => {
        if (cancelled) return
        setOrder(o)
        return getAuction(o.sessionId)
          .then((d) => {
            if (!cancelled) {
              setProduct({
                id: d.product.id,
                name: d.product.name,
                description: d.product.description,
                coverUrl: d.product.coverUrl,
              })
            }
          })
          .catch(() => {
            if (!cancelled) setProduct(null)
          })
      })
      .catch((e) => {
        if (!cancelled) setError(e instanceof Error ? e.message : '加载失败')
      })
    return () => {
      cancelled = true
    }
  }, [orderId])

  const handlePay = useCallback(async () => {
    if (!order) return
    setPaying(true)
    setError(null)
    try {
      const updated = await mockPayOrder(order.id)
      setOrder(updated)
    } catch (e) {
      setError(e instanceof Error ? e.message : '支付失败')
    } finally {
      setPaying(false)
    }
  }, [order])

  if (!isLoggedIn()) {
    return (
      <div className="user-page">
        <p className="user-hint">请先登录买家账号</p>
        <Link to="/app/login">去登录</Link>
      </div>
    )
  }

  if (!order && !error) {
    return (
      <div className="user-page">
        <p className="user-hint">加载中…</p>
      </div>
    )
  }

  return (
    <div className="user-page">
      <Link to="/app/orders" className="back-link">
        ← 我的订单
      </Link>

      {order && (
        <>
          <h1>订单支付</h1>
          <p className="order-no">{order.orderNo}</p>

          {product && (
            <div className="pay-product">
              <img src={product.coverUrl} alt="" className="pay-product__img" />
              <div>
                <h2 className="pay-product__name">{product.name}</h2>
                <p className="pay-product__desc muted">{product.description}</p>
              </div>
            </div>
          )}

          <div className="pay-demo-banner" role="note">
            <strong>演示环境</strong>
            <p>当前为模拟支付，不会产生真实扣款。正式环境将接入微信/支付宝等渠道。</p>
          </div>

          <div className="pay-amount">
            <span>应付金额</span>
            <strong>{formatCents(order.amount)}</strong>
          </div>

          <p className={`badge badge--${order.status}`}>
            {ORDER_STATUS_LABEL[order.status]}
          </p>

          {order.status === 'pending_pay' && (
            <button
              type="button"
              className="btn-primary btn-block"
              disabled={paying}
              onClick={handlePay}
            >
              {paying ? '支付中…' : '确认模拟支付'}
            </button>
          )}

          {order.status === 'paid' && (
            <div className="pay-success">
              <p>支付成功</p>
              {order.paidAt && (
                <span className="muted">
                  {new Date(order.paidAt).toLocaleString('zh-CN')}
                </span>
              )}
              <button
                type="button"
                className="btn-ghost btn-block"
                onClick={() => navigate(`/app/result/${order.sessionId}`)}
              >
                查看竞拍结果
              </button>
            </div>
          )}
        </>
      )}

      {error && <p className="user-error">{error}</p>}
    </div>
  )
}
