import { useCallback, useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { listProducts } from '../../api/admin'
import type { ProductStatus, ProductView } from '../../api/types'
import { formatCents } from '../../utils/money'
import { StatusBadge } from '../components/StatusBadge'
import {
  PRODUCT_STATUS_LABEL,
  SESSION_STATUS_LABEL,
} from '../labels'

const STATUS_OPTIONS: { value: '' | ProductStatus; label: string }[] = [
  { value: '', label: '全部状态' },
  { value: 'draft', label: '草稿' },
  { value: 'listed', label: '已上架' },
  { value: 'auctioning', label: '竞拍中' },
  { value: 'sold', label: '已售出' },
  { value: 'off_shelf', label: '已下架' },
]

const SESSION_FILTER = [
  { value: '', label: '全部场次' },
  { value: 'pending', label: '未开始' },
  { value: 'running', label: '进行中' },
  { value: 'settled', label: '已成交' },
  { value: 'cancelled', label: '已取消' },
] as const

export function ProductListPage() {
  const [status, setStatus] = useState<'' | ProductStatus>('')
  const [sessionFilter, setSessionFilter] = useState('')
  const [page, setPage] = useState(1)
  const [items, setItems] = useState<ProductView[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const pageSize = 20

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const res = await listProducts({
        page,
        pageSize,
        status: status || undefined,
      })
      let filtered = res.items
      if (sessionFilter) {
        filtered = filtered.filter((p) => p.auction?.status === sessionFilter)
      }
      setItems(filtered)
      setTotal(sessionFilter ? filtered.length : res.total)
    } catch (err) {
      setError(err instanceof Error ? err.message : '加载失败')
    } finally {
      setLoading(false)
    }
  }, [page, status, sessionFilter])

  useEffect(() => {
    void load()
  }, [load])

  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  return (
    <div className="admin-page">
      <div className="admin-page__head">
        <div>
          <h2>商品管理</h2>
          <p className="page-desc">管理商品信息与竞拍场次进度</p>
        </div>
        <Link to="/admin/products/new" className="btn-primary">
          + 新建商品
        </Link>
      </div>

      <div className="filter-bar">
        <select
          value={status}
          onChange={(e) => {
            setStatus(e.target.value as '' | ProductStatus)
            setPage(1)
          }}
        >
          {STATUS_OPTIONS.map((o) => (
            <option key={o.value || 'all'} value={o.value}>
              {o.label}
            </option>
          ))}
        </select>
        <select
          value={sessionFilter}
          onChange={(e) => {
            setSessionFilter(e.target.value)
            setPage(1)
          }}
        >
          {SESSION_FILTER.map((o) => (
            <option key={o.value || 'all'} value={o.value}>
              {o.label}
            </option>
          ))}
        </select>
        <button type="button" className="btn-ghost" onClick={() => void load()}>
          刷新
        </button>
      </div>

      {error && <p className="form-error">{error}</p>}

      <div className="data-table-wrap">
        <table className="data-table">
          <thead>
            <tr>
              <th>商品</th>
              <th>商品状态</th>
              <th>竞拍</th>
              <th>当前价</th>
              <th>出价/人数</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <tr>
                <td colSpan={6} className="table-empty">
                  加载中…
                </td>
              </tr>
            ) : items.length === 0 ? (
              <tr>
                <td colSpan={6} className="table-empty">
                  暂无商品
                </td>
              </tr>
            ) : (
              items.map((p) => (
                <tr key={p.id}>
                  <td>
                    <div className="product-cell">
                      {p.coverUrl && (
                        <img src={p.coverUrl} alt="" className="product-thumb" />
                      )}
                      <div>
                        <strong>{p.name}</strong>
                        <span className="muted">ID {p.id}</span>
                      </div>
                    </div>
                  </td>
                  <td>
                    <StatusBadge
                      label={PRODUCT_STATUS_LABEL[p.status]}
                      variant="product"
                      tone={p.status}
                    />
                  </td>
                  <td>
                    {p.auction ? (
                      <StatusBadge
                        label={SESSION_STATUS_LABEL[p.auction.status]}
                        variant="session"
                        tone={p.auction.status}
                      />
                    ) : (
                      <span className="muted">—</span>
                    )}
                  </td>
                  <td>
                    {p.auction ? formatCents(p.auction.currentPrice) : '—'}
                  </td>
                  <td>
                    {p.auction
                      ? `${p.auction.bidCount} 笔 / ${p.auction.participantCount} 人`
                      : '—'}
                  </td>
                  <td className="table-actions">
                    <Link to={`/admin/products/${p.id}`}>详情</Link>
                    {(p.status === 'draft' || p.status === 'listed') && (
                      <Link to={`/admin/products/${p.id}/edit`}>编辑</Link>
                    )}
                    {p.status === 'listed' && !p.auction && (
                      <Link to={`/admin/products/${p.id}/auction`}>发布竞拍</Link>
                    )}
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      <div className="pagination">
        <button
          type="button"
          className="btn-ghost"
          disabled={page <= 1}
          onClick={() => setPage((p) => p - 1)}
        >
          上一页
        </button>
        <span>
          第 {page} / {totalPages} 页（共 {total} 条）
        </span>
        <button
          type="button"
          className="btn-ghost"
          disabled={page >= totalPages}
          onClick={() => setPage((p) => p + 1)}
        >
          下一页
        </button>
      </div>
    </div>
  )
}
