import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { getAuction, getOrderBySession } from '../../api/user'
import type { UserAuctionDetail } from '../../api/user'
import type { OrderListItem } from '../../api/types'
import { getUser, isLoggedIn } from '../../auth/session'
import { formatCents } from '../../utils/money'

type Props = {
  sessionId: number | null
  onClose: () => void
}

/** 连播房间内回看已拍场次，无需离开直播间 */
export function SessionRecapSheet({ sessionId, onClose }: Props) {
  const [detail, setDetail] = useState<UserAuctionDetail | null>(null)
  const [orderItem, setOrderItem] = useState<OrderListItem | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const userId = getUser()?.id ?? null
  const isWinner = userId != null && detail?.session.winnerId === userId

  useEffect(() => {
    if (!sessionId) {
      setDetail(null)
      setOrderItem(null)
      setError(null)
      return
    }
    let cancelled = false
    setLoading(true)
    setError(null)

    getAuction(sessionId)
      .then((d) => {
        if (cancelled) return
        setDetail(d)
        if (isLoggedIn() && d.session.winnerId === getUser()?.id) {
          return getOrderBySession(sessionId).then((item) => {
            if (!cancelled) setOrderItem(item)
          })
        }
        return undefined
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
  }, [sessionId])

  if (!sessionId) return null

  return (
    <div className="recap-sheet" role="dialog" aria-modal="true" aria-label="场次回看">
      <button
        type="button"
        className="recap-sheet__backdrop"
        aria-label="关闭"
        onClick={onClose}
      />
      <div className="recap-sheet__panel">
        <header className="recap-sheet__head">
          <h2>场次回看</h2>
          <button type="button" className="btn-icon btn-icon--sm" onClick={onClose}>
            ✕
          </button>
        </header>

        {loading && <p className="user-hint">加载中…</p>}
        {error && <p className="user-error">{error}</p>}

        {detail && !loading && (
          <div className="recap-sheet__body">
            <div className="recap-sheet__hero">
              {detail.product.coverUrl && (
                <img src={detail.product.coverUrl} alt="" />
              )}
              <div>
                <h3>{detail.product.name}</h3>
                <p className={`recap-sheet__status${isWinner ? ' recap-sheet__status--win' : ''}`}>
                  {isWinner ? '您已中标' : '本场已成交'}
                </p>
              </div>
            </div>

            <dl className="detail-dl recap-sheet__dl">
              <dt>成交价</dt>
              <dd className="price-lg">{formatCents(detail.session.currentPrice)}</dd>
              <dt>出价次数</dt>
              <dd>{detail.session.bidCount}</dd>
              <dt>参与人数</dt>
              <dd>{detail.session.participantCount}</dd>
              {detail.session.settledAt && (
                <>
                  <dt>成交时间</dt>
                  <dd>{new Date(detail.session.settledAt).toLocaleString('zh-CN')}</dd>
                </>
              )}
            </dl>

            {isWinner && orderItem?.order.status === 'pending_pay' && (
              <Link
                to={`/app/orders/${orderItem.order.id}`}
                className="btn-primary btn-block"
                onClick={onClose}
              >
                去支付 {formatCents(orderItem.order.amount)}
              </Link>
            )}
            {isWinner && orderItem && orderItem.order.status !== 'pending_pay' && (
              <Link
                to={`/app/orders/${orderItem.order.id}`}
                className="btn-secondary btn-block"
                onClick={onClose}
              >
                查看订单
              </Link>
            )}
          </div>
        )}
      </div>
    </div>
  )
}
