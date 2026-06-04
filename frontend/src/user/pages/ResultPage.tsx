import { useEffect, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { getAuction, getOrderBySession } from '../../api/user'
import type { UserAuctionDetail } from '../../api/user'
import type { Order } from '../../api/types'
import { getUser, isLoggedIn } from '../../auth/session'
import { formatCents } from '../../utils/money'

export function ResultPage() {
  const { sessionId } = useParams<{ sessionId: string }>()
  const navigate = useNavigate()
  const [detail, setDetail] = useState<UserAuctionDetail | null>(null)
  const [order, setOrder] = useState<Order | null>(null)
  const [error, setError] = useState<string | null>(null)

  const userId = getUser()?.id ?? null
  const isWinner = userId != null && detail?.session.winnerId === userId

  useEffect(() => {
    const id = Number(sessionId)
    if (!Number.isFinite(id)) return
    let cancelled = false

    getAuction(id)
      .then((d) => {
        if (!cancelled) setDetail(d)
        if (!cancelled && isLoggedIn() && d.session.winnerId === getUser()?.id) {
          return getOrderBySession(id).then((o) => {
            if (!cancelled) setOrder(o)
          })
        }
        return undefined
      })
      .catch((e) => {
        if (!cancelled) setError(e instanceof Error ? e.message : '加载失败')
      })

    return () => {
      cancelled = true
    }
  }, [sessionId])

  if (error || !detail) {
    return (
      <div className="user-page">
        {error ? <p className="user-error">{error}</p> : <p className="user-hint">加载中…</p>}
        <Link to="/app/history">返回历史</Link>
      </div>
    )
  }

  const { session, product } = detail

  return (
    <div className="user-page result-page">
      <div className={`result-banner ${isWinner ? 'result-banner--win' : ''}`}>
        <h1>{isWinner ? '恭喜中标！' : '竞拍已结束'}</h1>
        <p>{product.name}</p>
      </div>

      <img src={product.coverUrl} alt="" className="detail-hero" />

      <dl className="detail-dl">
        <dt>成交价</dt>
        <dd className="price-lg">{formatCents(session.currentPrice)}</dd>
        <dt>出价次数</dt>
        <dd>{session.bidCount}</dd>
        <dt>参与人数</dt>
        <dd>{session.participantCount}</dd>
        {session.settledAt && (
          <>
            <dt>成交时间</dt>
            <dd>{new Date(session.settledAt).toLocaleString('zh-CN')}</dd>
          </>
        )}
      </dl>

      {isWinner && order && (
        <div className="user-page__actions">
          {order.status === 'pending_pay' ? (
            <button
              type="button"
              className="btn-primary btn-block"
              onClick={() => navigate(`/app/orders/${order.id}`)}
            >
              去支付 {formatCents(order.amount)}
            </button>
          ) : (
            <p className="form-info">订单已支付</p>
          )}
        </div>
      )}

      {isWinner && !order && isLoggedIn() && (
        <p className="user-hint">订单生成中，请稍后刷新</p>
      )}

      {!isLoggedIn() && isWinner && (
        <p className="user-hint">
          <Link to="/app/login">登录</Link> 后可查看订单并支付
        </p>
      )}

      {!isWinner && (
        <Link to="/app" className="btn-secondary btn-block">
          去看看其他竞拍
        </Link>
      )}

      <Link to="/app/history" className="back-link">
        返回历史记录
      </Link>
    </div>
  )
}
