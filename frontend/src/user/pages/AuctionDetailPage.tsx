import { useEffect, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { getAuction } from '../../api/user'
import type { UserAuctionDetail } from '../../api/user'
import { AuctionRulesCard } from '../../components/auction/AuctionRulesCard'
import { SESSION_STATUS_LABEL } from '../../admin/labels'
import { auctionEntryPath } from '../../utils/auctionNav'
import { useCountdown } from '../../hooks/useCountdown'
import { formatCents } from '../../utils/money'
import { formatRemainingMs } from '../../utils/time'

function DetailCountdown({ endAt, running }: { endAt?: string; running: boolean }) {
  const remainingMs = useCountdown(endAt, running && Boolean(endAt))
  if (!running || remainingMs == null) return null
  const urgent = remainingMs > 0 && remainingMs <= 60_000
  return (
    <div className={`detail-countdown${urgent ? ' detail-countdown--urgent' : ''}`}>
      <span className="detail-countdown__label">距结束</span>
      <strong>{formatRemainingMs(remainingMs)}</strong>
    </div>
  )
}

export function AuctionDetailPage() {
  const { sessionId } = useParams<{ sessionId: string }>()
  const navigate = useNavigate()
  const [detail, setDetail] = useState<UserAuctionDetail | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const id = Number(sessionId)
    if (!Number.isFinite(id)) return
    let cancelled = false
    getAuction(id)
      .then((d) => {
        if (!cancelled) setDetail(d)
      })
      .catch((e) => {
        if (!cancelled) setError(e instanceof Error ? e.message : '加载失败')
      })
    return () => {
      cancelled = true
    }
  }, [sessionId])

  if (error) {
    return (
      <div className="user-page">
        <div className="inline-alert inline-alert--error">
          <p>{error}</p>
        </div>
        <Link to="/app" className="btn-secondary">
          返回列表
        </Link>
      </div>
    )
  }

  if (!detail) {
    return (
      <div className="user-page">
        <div className="detail-skeleton detail-skeleton--hero" />
        <div className="detail-skeleton detail-skeleton--line" />
        <div className="detail-skeleton detail-skeleton--line short" />
      </div>
    )
  }

  const { session, product, snapshot } = detail
  const canEnterLive =
    session.status === 'pending' || session.status === 'running'
  const isRunning = session.status === 'running'

  return (
    <div className="user-page user-page--detail">
      <Link to="/app" className="back-link back-link--overlay">
        ← 返回
      </Link>

      <div className="detail-hero-wrap">
        <img src={product.coverUrl} alt="" className="detail-hero detail-hero--full" />
        <div className="detail-hero__shade" />
        {isRunning && (
          <span className="detail-hero__live">LIVE</span>
        )}
      </div>

      <div className="detail-body">
        <h1 className="detail-title">{product.name}</h1>
        <p className="page-desc detail-desc">{product.description}</p>

        <DetailCountdown endAt={session.endAt} running={isRunning} />

        <div className="snapshot-strip snapshot-strip--rich">
          <div className="snapshot-strip__item">
            <span className="stat-label">当前价</span>
            <strong className="price-lg">{formatCents(snapshot.currentPrice)}</strong>
          </div>
          <div className="snapshot-strip__item">
            <span className="stat-label">出价 / 参与</span>
            <strong>
              {snapshot.bidCount} / {snapshot.participantCount}
            </strong>
          </div>
          <div className="snapshot-strip__item">
            <span className="stat-label">状态</span>
            <strong className={`badge-inline badge--${session.status}`}>
              {SESSION_STATUS_LABEL[session.status]}
            </strong>
          </div>
        </div>

        <AuctionRulesCard
          rules={session.rules}
          status={SESSION_STATUS_LABEL[session.status]}
        />

        <div className="user-page__actions user-page__actions--sticky">
          {canEnterLive && (
            <button
              type="button"
              className="btn-primary btn-block btn-glow"
              onClick={() => navigate(auctionEntryPath(session))}
            >
              {isRunning ? '进入直播间出价' : '进入直播间围观'}
            </button>
          )}
          {session.status === 'settled' && (
            <button
              type="button"
              className="btn-primary btn-block"
              onClick={() => navigate(`/app/result/${session.id}`)}
            >
              查看成交结果
            </button>
          )}
        </div>
      </div>
    </div>
  )
}
