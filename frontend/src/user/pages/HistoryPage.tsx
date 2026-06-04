import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { listAuctions } from '../../api/user'
import type { UserAuctionListItem } from '../../api/user'
import { formatCents } from '../../utils/money'

export function HistoryPage() {
  const [items, setItems] = useState<UserAuctionListItem[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    let cancelled = false
    listAuctions({ status: 'settled', page: 1, pageSize: 50 })
      .then((res) => {
        if (!cancelled) setItems(res.items)
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [])

  const totalAmount = items.reduce((s, x) => s + x.session.currentPrice, 0)

  return (
    <div className="user-page user-page--history">
      <header className="page-hero page-hero--compact">
        <div className="page-hero__content">
          <span className="page-hero__badge">成交档案</span>
          <h1 className="page-hero__title">历史竞拍</h1>
          <p className="page-hero__sub">已成交场次与成交价记录</p>
        </div>
      </header>

      {!loading && items.length > 0 && (
        <div className="home-stats home-stats--compact">
          <div className="home-stat">
            <strong>{items.length}</strong>
            <span>成交场次</span>
          </div>
          <div className="home-stat home-stat--wide">
            <strong>{formatCents(totalAmount)}</strong>
            <span>累计成交额</span>
          </div>
        </div>
      )}

      {loading && (
        <ul className="auction-card-list" aria-busy="true">
          {[1, 2].map((n) => (
            <li key={n} className="auction-card auction-card--skeleton" />
          ))}
        </ul>
      )}

      <ul className="auction-card-list auction-card-list--compact">
        {items.map(({ session, product }) => (
          <li key={session.id}>
            <Link
              to={`/app/result/${session.id}`}
              className="auction-card auction-card--row auction-card--history"
            >
              <img src={product.coverUrl} alt="" className="auction-card__img" />
              <div className="auction-card__body">
                <h2>{product.name}</h2>
                <p className="price-sm">成交价 {formatCents(session.currentPrice)}</p>
                {session.settledAt && (
                  <span className="muted auction-card__time">
                    {new Date(session.settledAt).toLocaleString('zh-CN')}
                  </span>
                )}
              </div>
              <span className="auction-card__chevron" aria-hidden>›</span>
            </Link>
          </li>
        ))}
      </ul>

      {!loading && items.length === 0 && (
        <div className="empty-state">
          <div className="empty-state__icon" aria-hidden>
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
              <path d="M12 8v4l3 2M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
          </div>
          <p className="empty-state__title">暂无历史记录</p>
          <p className="empty-state__desc">参与竞拍并成交后，记录会出现在这里</p>
          <Link to="/app" className="btn-secondary">
            去看看正在直播的场次
          </Link>
        </div>
      )}
    </div>
  )
}
