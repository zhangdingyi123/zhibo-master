import { useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { listOrders, listProducts } from '../../api/admin'
import type { Order, ProductView } from '../../api/types'
import { formatCents } from '../../utils/money'
import { SESSION_STATUS_LABEL } from '../labels'

export function DashboardPage() {
  const [products, setProducts] = useState<ProductView[]>([])
  const [orders, setOrders] = useState<Order[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    let cancelled = false
    Promise.all([
      listProducts({ page: 1, pageSize: 100 }),
      listOrders({ page: 1, pageSize: 50 }),
    ])
      .then(([pRes, oRes]) => {
        if (!cancelled) {
          setProducts(pRes.items)
          setOrders(oRes.items)
        }
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [])

  const stats = useMemo(() => {
    const running = products.filter((p) => p.auction?.status === 'running').length
    const pending = products.filter((p) => p.auction?.status === 'pending').length
    const settled = products.filter((p) => p.auction?.status === 'settled').length
    const pendingPay = orders.filter((o) => o.status === 'pending_pay').length
    const revenue = orders
      .filter((o) => o.status === 'paid' || o.status === 'closed')
      .reduce((sum, o) => sum + o.amount, 0)
    return { running, pending, settled, pendingPay, revenue, totalProducts: products.length }
  }, [products, orders])

  const liveProducts = products
    .filter((p) => p.auction?.status === 'running')
    .slice(0, 4)

  return (
    <div className="admin-page admin-page--dashboard">
      <div className="admin-page__head">
        <div>
          <h2>数据概览</h2>
          <p className="page-desc">实时掌握商品、场次与订单情况</p>
        </div>
        <Link to="/admin/products/new" className="btn-primary">
          + 发布新商品
        </Link>
      </div>

      {loading ? (
        <p className="page-desc">加载中…</p>
      ) : (
        <>
          <div className="dash-stats">
            <div className="dash-stat dash-stat--accent">
              <span className="dash-stat__label">进行中场次</span>
              <strong className="dash-stat__value">{stats.running}</strong>
            </div>
            <div className="dash-stat">
              <span className="dash-stat__label">待开始</span>
              <strong className="dash-stat__value">{stats.pending}</strong>
            </div>
            <div className="dash-stat">
              <span className="dash-stat__label">已成交场次</span>
              <strong className="dash-stat__value">{stats.settled}</strong>
            </div>
            <div className="dash-stat">
              <span className="dash-stat__label">待支付订单</span>
              <strong className="dash-stat__value">{stats.pendingPay}</strong>
            </div>
            <div className="dash-stat dash-stat--wide">
              <span className="dash-stat__label">已收款合计（模拟）</span>
              <strong className="dash-stat__value dash-stat__value--money">
                {formatCents(stats.revenue)}
              </strong>
            </div>
            <div className="dash-stat">
              <span className="dash-stat__label">商品总数</span>
              <strong className="dash-stat__value">{stats.totalProducts}</strong>
            </div>
          </div>

          <section className="dash-panel">
            <div className="dash-panel__head">
              <h3>正在直播竞拍</h3>
              <Link to="/admin/products">查看全部</Link>
            </div>
            {liveProducts.length === 0 ? (
              <p className="muted dash-panel__empty">暂无进行中场次，去发布一场竞拍吧</p>
            ) : (
              <ul className="dash-live-list">
                {liveProducts.map((p) => (
                  <li key={p.id}>
                    <Link to={`/admin/products/${p.id}`} className="dash-live-item">
                      {p.coverUrl && (
                        <img src={p.coverUrl} alt="" className="dash-live-item__thumb" />
                      )}
                      <div>
                        <strong>{p.name}</strong>
                        <span className="muted">
                          {p.auction
                            ? `${SESSION_STATUS_LABEL[p.auction.status]} · ${formatCents(p.auction.currentPrice)}`
                            : ''}
                        </span>
                      </div>
                      <span className="dash-live-item__arrow" aria-hidden>›</span>
                    </Link>
                  </li>
                ))}
              </ul>
            )}
          </section>

          <div className="dash-quick">
            <Link to="/admin/products" className="dash-quick__card">
              <span className="dash-quick__title">商品管理</span>
              <span className="muted">上架、编辑、发布竞拍</span>
            </Link>
            <Link to="/admin/orders" className="dash-quick__card">
              <span className="dash-quick__title">订单管理</span>
              <span className="muted">成交订单与支付状态</span>
            </Link>
            <a href="/app" className="dash-quick__card dash-quick__card--outline" target="_blank" rel="noreferrer">
              <span className="dash-quick__title">打开用户端</span>
              <span className="muted">预览 H5 直播间体验</span>
            </a>
          </div>
        </>
      )}
    </div>
  )
}
