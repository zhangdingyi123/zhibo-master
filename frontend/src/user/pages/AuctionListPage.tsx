import { useCallback, useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { listAuctions } from '../../api/user'
import type { UserAuctionListItem } from '../../api/user'
import type { SessionStatus } from '../../api/types'
import { SESSION_STATUS_LABEL } from '../../admin/labels'
import { FeaturedCarousel } from '../../components/auction/FeaturedCarousel'
import { SearchBar } from '../../components/user/SearchBar'
import { auctionEntryCta, auctionEntryPath } from '../../utils/auctionNav'
import { useCountdown } from '../../hooks/useCountdown'
import { formatCents } from '../../utils/money'
import { formatRemainingMs } from '../../utils/time'

const TABS = [
  { key: '', label: '全部' },
  { key: 'pending', label: '待开始' },
  { key: 'running', label: '进行中' },
] as const

type SortKey = 'default' | 'price_desc' | 'ending_soon'

function RunningCountdown({ endAt }: { endAt?: string }) {
  const remainingMs = useCountdown(endAt, Boolean(endAt))
  if (remainingMs == null) return null
  const urgent = remainingMs > 0 && remainingMs <= 60_000
  return (
    <span className={`auction-card__countdown${urgent ? ' auction-card__countdown--urgent' : ''}`}>
      剩余 {formatRemainingMs(remainingMs)}
    </span>
  )
}

export function AuctionListPage() {
  const [status, setStatus] = useState<SessionStatus | ''>('')
  const [items, setItems] = useState<UserAuctionListItem[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [query, setQuery] = useState('')
  const [sort, setSort] = useState<SortKey>('default')

  const load = useCallback(() => {
    let cancelled = false
    setLoading(true)
    setError(null)
    listAuctions({
      status: status === '' ? undefined : status,
      page: 1,
      pageSize: 50,
    })
      .then((res) => {
        if (!cancelled) setItems(res.items)
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
  }, [status])

  useEffect(() => {
    const cancel = load()
    return cancel
  }, [load])

  const stats = useMemo(() => {
    const live = items.filter((x) => x.session.status === 'running').length
    const participants = items.reduce(
      (sum, x) => sum + (x.session.participantCount ?? 0),
      0,
    )
    return { live, participants, total: items.length }
  }, [items])

  const displayItems = useMemo(() => {
    let list = [...items]
    const q = query.trim().toLowerCase()
    if (q) {
      list = list.filter(
        (x) =>
          x.product.name.toLowerCase().includes(q) ||
          x.product.description.toLowerCase().includes(q),
      )
    }
    if (sort === 'price_desc') {
      list.sort((a, b) => b.session.currentPrice - a.session.currentPrice)
    } else if (sort === 'ending_soon') {
      list.sort((a, b) => {
        const ea = a.session.endAt ? new Date(a.session.endAt).getTime() : Infinity
        const eb = b.session.endAt ? new Date(b.session.endAt).getTime() : Infinity
        return ea - eb
      })
    }
    return list
  }, [items, query, sort])

  return (
    <div className="user-page user-page--home">
      <header className="page-hero">
        <div className="page-hero__content">
          <span className="page-hero__badge">直播竞拍</span>
          <h1 className="page-hero__title">好物专场</h1>
          <p className="page-hero__sub">精选场次，实时出价</p>
        </div>
        <button
          type="button"
          className="btn-icon"
          disabled={loading}
          onClick={() => load()}
          aria-label="刷新列表"
          title="刷新"
        >
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" aria-hidden>
            <path d="M21 12a9 9 0 11-2.64-6.36" />
            <path d="M21 3v6h-6" />
          </svg>
        </button>
      </header>

      {!loading && items.length > 0 && (
        <div className="home-stats" aria-label="场次概览">
          <div className="home-stat">
            <strong>{stats.live}</strong>
            <span>直播中</span>
          </div>
          <div className="home-stat">
            <strong>{stats.total}</strong>
            <span>场次</span>
          </div>
          <div className="home-stat">
            <strong>{stats.participants}</strong>
            <span>累计参与</span>
          </div>
        </div>
      )}

      <FeaturedCarousel items={items} />

      <SearchBar value={query} onChange={setQuery} />

      <div className="list-toolbar">
        <div className="tab-row tab-row--pills">
          {TABS.map((t) => (
            <button
              key={t.key}
              type="button"
              className={status === t.key ? 'tab active' : 'tab'}
              onClick={() => setStatus(t.key as SessionStatus | '')}
            >
              {t.label}
            </button>
          ))}
        </div>
        <select
          className="sort-select"
          value={sort}
          onChange={(e) => setSort(e.target.value as SortKey)}
          aria-label="排序方式"
        >
          <option value="default">默认排序</option>
          <option value="price_desc">价格从高到低</option>
          <option value="ending_soon">即将结束</option>
        </select>
      </div>

      {error && (
        <div className="inline-alert inline-alert--error">
          <p>{error}</p>
          <button type="button" className="btn-ghost btn-sm" onClick={() => load()}>
            重试
          </button>
        </div>
      )}

      {loading && items.length === 0 && (
        <ul className="auction-card-list" aria-busy="true">
          {[1, 2, 3].map((n) => (
            <li key={n} className="auction-card auction-card--skeleton" />
          ))}
        </ul>
      )}

      <ul className="auction-card-list">
        {displayItems.map(({ session, product }) => (
          <li key={session.id}>
            <Link
              to={auctionEntryPath(session)}
              className={`auction-card${session.status === 'running' ? ' auction-card--live' : ''}`}
            >
              <div className="auction-card__media">
                <img src={product.coverUrl} alt="" className="auction-card__img" />
                {session.status === 'running' && (
                  <span className="live-dot" aria-label="进行中">
                    LIVE
                  </span>
                )}
              </div>
              <div className="auction-card__body">
                <h2>{product.name}</h2>
                <p className="auction-card__desc">{product.description}</p>
                <div className="auction-card__meta">
                  <span className={`badge badge--${session.status}`}>
                    {SESSION_STATUS_LABEL[session.status]}
                  </span>
                  {session.status === 'running' && (
                    <RunningCountdown endAt={session.endAt} />
                  )}
                  <span className="price-sm">
                    {formatCents(session.currentPrice)}
                  </span>
                </div>
                <span className="auction-card__cta">
                  {auctionEntryCta(session.status)}
                </span>
              </div>
            </Link>
          </li>
        ))}
      </ul>

      {!loading && displayItems.length === 0 && !error && (
        <div className="empty-state">
          <div className="empty-state__icon" aria-hidden>
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
              <rect x="3" y="5" width="18" height="14" rx="2" />
              <path d="M7 9h10M7 13h6" />
            </svg>
          </div>
          <p className="empty-state__title">
            {query.trim() ? '没有匹配的商品' : '暂无竞拍场次'}
          </p>
          <p className="empty-state__desc">
            {query.trim() ? '试试其他关键词' : '稍后再来看看，或切换上方筛选'}
          </p>
        </div>
      )}
    </div>
  )
}
