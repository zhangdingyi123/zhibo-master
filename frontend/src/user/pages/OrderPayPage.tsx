import { useCallback, useEffect, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import {
  cancelOrder,
  confirmReceive,
  getOrder,
  mockPayOrder,
  submitShippingAddress,
} from '../../api/user'
import { AFTERSALE_REASON_PRESETS } from '../../admin/labels'
import type { Order, OrderListItem } from '../../api/types'
import { ORDER_STATUS_LABEL } from '../../admin/labels'
import { isLoggedIn } from '../../auth/session'
import { CopyButton } from '../../components/CopyButton'
import { formatCents } from '../../utils/money'

function formatExpireRemaining(payExpireAt?: string) {
  if (!payExpireAt) return null
  const ms = new Date(payExpireAt).getTime() - Date.now()
  if (ms <= 0) return '已超时'
  const min = Math.floor(ms / 60000)
  const sec = Math.floor((ms % 60000) / 1000)
  return `${min}:${sec.toString().padStart(2, '0')}`
}

function hasAddress(order: Order) {
  return Boolean(
    order.receiverName?.trim() &&
      order.receiverPhone?.trim() &&
      order.receiverAddress?.trim(),
  )
}

const FULFILLMENT_STEPS = [
  { key: 'pay', label: '支付' },
  { key: 'address', label: '填地址' },
  { key: 'ship', label: '发货' },
  { key: 'done', label: '完成' },
] as const

function fulfillmentStep(order: Order): number {
  if (order.status === 'completed') return 4
  if (order.status === 'shipped') return 3
  if (order.status === 'paid') return hasAddress(order) ? 2 : 1
  if (order.status === 'pending_pay') return 0
  return 0
}

export function OrderPayPage() {
  const { orderId } = useParams<{ orderId: string }>()
  const navigate = useNavigate()
  const [detail, setDetail] = useState<OrderListItem | null>(null)
  const [expireLabel, setExpireLabel] = useState<string | null>(null)
  const [paying, setPaying] = useState(false)
  const [savingAddress, setSavingAddress] = useState(false)
  const [confirming, setConfirming] = useState(false)
  const [cancelling, setCancelling] = useState(false)
  const [cancelReason, setCancelReason] = useState<string>(AFTERSALE_REASON_PRESETS[1])
  const [error, setError] = useState<string | null>(null)
  const [receiverName, setReceiverName] = useState('')
  const [receiverPhone, setReceiverPhone] = useState('')
  const [receiverAddress, setReceiverAddress] = useState('')

  const order: Order | null = detail?.order ?? null
  const product = detail?.product ?? null

  const loadOrder = useCallback(async (id: number) => {
    const d = await getOrder(id)
    setDetail(d)
    setReceiverName(d.order.receiverName ?? '')
    setReceiverPhone(d.order.receiverPhone ?? '')
    setReceiverAddress(d.order.receiverAddress ?? '')
    return d
  }, [])

  useEffect(() => {
    const id = Number(orderId)
    if (!Number.isFinite(id) || !isLoggedIn()) return
    let cancelled = false
    loadOrder(id)
      .catch((e) => {
        if (!cancelled) setError(e instanceof Error ? e.message : '加载失败')
      })
    return () => {
      cancelled = true
    }
  }, [orderId, loadOrder])

  useEffect(() => {
    if (!order?.payExpireAt || order.status !== 'pending_pay') {
      setExpireLabel(null)
      return
    }
    const tick = () => setExpireLabel(formatExpireRemaining(order.payExpireAt))
    tick()
    const id = window.setInterval(tick, 1000)
    return () => clearInterval(id)
  }, [order?.payExpireAt, order?.status])

  const handlePay = useCallback(async () => {
    if (!order) return
    setPaying(true)
    setError(null)
    try {
      const updated = await mockPayOrder(order.id)
      setDetail(updated)
      setReceiverName(updated.order.receiverName ?? '')
      setReceiverPhone(updated.order.receiverPhone ?? '')
      setReceiverAddress(updated.order.receiverAddress ?? '')
    } catch (e) {
      setError(e instanceof Error ? e.message : '支付失败')
    } finally {
      setPaying(false)
    }
  }, [order])

  const handleSaveAddress = useCallback(async () => {
    if (!order) return
    setSavingAddress(true)
    setError(null)
    try {
      const updated = await submitShippingAddress(order.id, {
        receiverName,
        receiverPhone,
        receiverAddress,
      })
      setDetail(updated)
    } catch (e) {
      setError(e instanceof Error ? e.message : '保存失败')
    } finally {
      setSavingAddress(false)
    }
  }, [order, receiverName, receiverPhone, receiverAddress])

  const handleCancel = useCallback(async () => {
    if (!order) return
    const reason = cancelReason.trim()
    if (!reason) {
      setError('请填写取消原因')
      return
    }
    setCancelling(true)
    setError(null)
    try {
      const updated = await cancelOrder(order.id, reason)
      setDetail(updated)
    } catch (e) {
      setError(e instanceof Error ? e.message : '取消失败')
    } finally {
      setCancelling(false)
    }
  }, [order, cancelReason])

  const handleConfirmReceive = useCallback(async () => {
    if (!order) return
    setConfirming(true)
    setError(null)
    try {
      const updated = await confirmReceive(order.id)
      setDetail(updated)
    } catch (e) {
      setError(e instanceof Error ? e.message : '确认失败')
    } finally {
      setConfirming(false)
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

  const step = order ? fulfillmentStep(order) : 0

  return (
    <div className="user-page">
      <Link to="/app/orders" className="back-link">
        ← 我的订单
      </Link>

      {order && (
        <>
          <h1>订单详情</h1>
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

          <div className="pay-amount">
            <span>成交金额</span>
            <strong>{formatCents(order.amount)}</strong>
          </div>

          <p className={`badge badge--${order.status}`}>
            {ORDER_STATUS_LABEL[order.status]}
          </p>

          {order.status !== 'closed' &&
            order.status !== 'cancelled' &&
            order.status !== 'refunded' && (
            <ol className="fulfillment-steps" aria-label="交易进度">
              {FULFILLMENT_STEPS.map((s, i) => (
                <li
                  key={s.key}
                  className={`fulfillment-steps__item${i < step ? ' fulfillment-steps__item--done' : ''}${i === step ? ' fulfillment-steps__item--current' : ''}`}
                >
                  <span className="fulfillment-steps__dot" />
                  <span>{s.label}</span>
                </li>
              ))}
            </ol>
          )}

          {order.status === 'pending_pay' && (
            <>
              <div className="pay-demo-banner" role="note">
                <strong>演示环境</strong>
                <p>当前为模拟支付，不会产生真实扣款。</p>
              </div>
              {expireLabel && (
                <p className="user-hint">
                  支付剩余时间：<strong>{expireLabel}</strong>
                </p>
              )}
              {expireLabel !== '已超时' && (
                <button
                  type="button"
                  className="btn-primary btn-block"
                  disabled={paying}
                  onClick={handlePay}
                >
                  {paying ? '支付中…' : '确认模拟支付'}
                </button>
              )}
              {expireLabel !== '已超时' && (
                <section className="fulfillment-card fulfillment-card--subtle">
                  <h2>不想买了？</h2>
                  <p className="user-hint">待支付订单可主动取消，无需联系客服。</p>
                  <label className="form-field">
                    <span>取消原因</span>
                    <select
                      value={cancelReason}
                      onChange={(e) => setCancelReason(e.target.value)}
                    >
                      {AFTERSALE_REASON_PRESETS.map((p) => (
                        <option key={p} value={p}>
                          {p}
                        </option>
                      ))}
                    </select>
                  </label>
                  <button
                    type="button"
                    className="btn-ghost btn-block"
                    disabled={cancelling}
                    onClick={handleCancel}
                  >
                    {cancelling ? '取消中…' : '取消订单'}
                  </button>
                </section>
              )}
            </>
          )}

          {order.status === 'closed' && (
            <p className="user-hint">订单已超时关闭，请联系主播重新竞拍。</p>
          )}

          {order.status === 'cancelled' && (
            <section className="fulfillment-card fulfillment-card--subtle">
              <h2>订单已取消</h2>
              {order.cancelReason && (
                <p className="user-hint">原因：{order.cancelReason}</p>
              )}
              {order.cancelledBy && (
                <p className="muted">
                  操作方：
                  {order.cancelledBy === 'buyer'
                    ? '您本人'
                    : order.cancelledBy === 'seller'
                      ? '主播'
                      : '系统'}
                </p>
              )}
              {order.cancelledAt && (
                <p className="muted">
                  {new Date(order.cancelledAt).toLocaleString('zh-CN')}
                </p>
              )}
              <p className="user-hint">如有疑问请联系主播协商。</p>
            </section>
          )}

          {order.status === 'refunded' && (
            <section className="fulfillment-card fulfillment-card--subtle">
              <h2>订单已退款</h2>
              {order.cancelReason && (
                <p className="user-hint">原因：{order.cancelReason}</p>
              )}
              {order.refundedAt && (
                <p className="muted">
                  退款时间：{new Date(order.refundedAt).toLocaleString('zh-CN')}
                </p>
              )}
              <p className="user-hint">
                演示环境已模拟原路退款，款项将退回模拟账户。
              </p>
            </section>
          )}

          {order.status === 'paid' && (
            <section className="fulfillment-card">
              <h2>填写收货地址</h2>
              <p className="user-hint">支付成功，请填写地址以便主播发货。</p>
              <label className="form-field">
                <span>收货人</span>
                <input
                  value={receiverName}
                  onChange={(e) => setReceiverName(e.target.value)}
                  placeholder="姓名"
                />
              </label>
              <label className="form-field">
                <span>手机号</span>
                <input
                  value={receiverPhone}
                  onChange={(e) => setReceiverPhone(e.target.value)}
                  placeholder="11 位手机号"
                />
              </label>
              <label className="form-field">
                <span>详细地址</span>
                <textarea
                  value={receiverAddress}
                  onChange={(e) => setReceiverAddress(e.target.value)}
                  placeholder="省市区 + 街道门牌"
                  rows={3}
                />
              </label>
              <button
                type="button"
                className="btn-primary btn-block"
                disabled={savingAddress}
                onClick={handleSaveAddress}
              >
                {savingAddress ? '保存中…' : hasAddress(order) ? '更新地址' : '保存地址'}
              </button>
              {hasAddress(order) && (
                <p className="form-info">地址已保存，等待主播发货</p>
              )}
              <p className="user-hint muted">
                误拍或需退款？请联系主播在管理端发起退款。
              </p>
            </section>
          )}

          {order.status === 'shipped' && (
            <section className="fulfillment-card">
              <h2>物流信息</h2>
              <dl className="detail-dl">
                <dt>收货人</dt>
                <dd>{order.receiverName}</dd>
                <dt>手机号</dt>
                <dd>{order.receiverPhone}</dd>
                <dt>地址</dt>
                <dd>{order.receiverAddress}</dd>
                {order.trackingNo && (
                  <>
                    <dt>物流单号</dt>
                    <dd className="detail-dd--copy">
                      <code className="tracking-no">{order.trackingNo}</code>
                      <CopyButton value={order.trackingNo} />
                    </dd>
                  </>
                )}
                {order.shippedAt && (
                  <>
                    <dt>发货时间</dt>
                    <dd>{new Date(order.shippedAt).toLocaleString('zh-CN')}</dd>
                  </>
                )}
              </dl>
              <button
                type="button"
                className="btn-primary btn-block"
                disabled={confirming}
                onClick={handleConfirmReceive}
              >
                {confirming ? '提交中…' : '确认收货'}
              </button>
              <p className="user-hint muted">
                误拍或需退款？请联系主播在管理端发起退款。
              </p>
            </section>
          )}

          {order.status === 'completed' && (
            <div className="pay-success">
              <p>交易已完成</p>
              {order.completedAt && (
                <span className="muted">
                  {new Date(order.completedAt).toLocaleString('zh-CN')}
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

          {(order.status === 'cancelled' || order.status === 'refunded') && (
            <section className="fulfillment-card fulfillment-card--subtle">
              <h2>{order.status === 'refunded' ? '订单已退款' : '订单已取消'}</h2>
              {order.cancelReason && <p className="user-hint">{order.cancelReason}</p>}
              {order.status === 'refunded' && (
                <p className="form-info">演示环境已模拟原路退款，资金将退回支付账户。</p>
              )}
              {order.refundedAt && (
                <p className="muted">
                  退款时间：{new Date(order.refundedAt).toLocaleString('zh-CN')}
                </p>
              )}
              {order.cancelledAt && order.status === 'cancelled' && (
                <p className="muted">
                  取消时间：{new Date(order.cancelledAt).toLocaleString('zh-CN')}
                </p>
              )}
            </section>
          )}
        </>
      )}

      {error && <p className="user-error">{error}</p>}
    </div>
  )
}
