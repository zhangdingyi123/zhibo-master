import { useCallback, useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { listOrders } from '../../api/admin'
import type { Order, OrderStatus } from '../../api/types'
import { formatCents } from '../../utils/money'
import { StatusBadge } from '../components/StatusBadge'
import { ORDER_STATUS_LABEL } from '../labels'

const STATUS_OPTIONS: { value: '' | OrderStatus; label: string }[] = [
  { value: '', label: '全部状态' },
  { value: 'pending_pay', label: '待支付' },
  { value: 'paid', label: '已支付' },
  { value: 'cancelled', label: '已取消' },
  { value: 'closed', label: '已关闭' },
]

export function OrderListPage() {
  const [status, setStatus] = useState<'' | OrderStatus>('')
  const [page, setPage] = useState(1)
  const [items, setItems] = useState<Order[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const pageSize = 20

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const res = await listOrders({
        page,
        pageSize,
        status: status || undefined,
      })
      setItems(res.items)
      setTotal(res.total)
    } catch (err) {
      setError(err instanceof Error ? err.message : '加载失败')
    } finally {
      setLoading(false)
    }
  }, [page, status])

  useEffect(() => {
    void load()
  }, [load])

  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  return (
    <div className="admin-page">
      <div className="admin-page__head">
        <div>
          <h2>订单管理</h2>
          <p className="page-desc">竞拍成交后自动生成的订单</p>
        </div>
      </div>

      <div className="filter-bar">
        <select
          value={status}
          onChange={(e) => {
            setStatus(e.target.value as '' | OrderStatus)
            setPage(1)
          }}
        >
          {STATUS_OPTIONS.map((o) => (
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
              <th>订单号</th>
              <th>场次</th>
              <th>商品</th>
              <th>买家</th>
              <th>金额</th>
              <th>状态</th>
              <th>创建时间</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <tr>
                <td colSpan={8} className="table-empty">
                  加载中…
                </td>
              </tr>
            ) : items.length === 0 ? (
              <tr>
                <td colSpan={8} className="table-empty">
                  暂无订单
                </td>
              </tr>
            ) : (
              items.map((o) => (
                <tr key={o.id}>
                  <td>
                    <code>{o.orderNo}</code>
                  </td>
                  <td>{o.sessionId}</td>
                  <td>{o.productId}</td>
                  <td>用户 #{o.buyerId}</td>
                  <td>{formatCents(o.amount)}</td>
                  <td>
                    <StatusBadge
                      label={ORDER_STATUS_LABEL[o.status]}
                      variant="order"
                      tone={o.status}
                    />
                  </td>
                  <td>{new Date(o.createdAt).toLocaleString('zh-CN')}</td>
                  <td>
                    <Link to={`/admin/orders/${o.id}`}>详情</Link>
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
