import { useCallback, useEffect, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import {
  cancelAuction,
  deleteProduct,
  getAuction,
  getProduct,
  updateAuctionRules,
} from '../../api/admin'
import type { AuctionSession, ProductView } from '../../api/types'
import { formatCents } from '../../utils/money'
import { AuctionRulesForm } from '../components/AuctionRulesForm'
import { StatusBadge } from '../components/StatusBadge'
import {
  ORDER_STATUS_LABEL,
  PRODUCT_STATUS_LABEL,
  SESSION_STATUS_LABEL,
} from '../labels'

export function ProductDetailPage() {
  const { id } = useParams()
  const navigate = useNavigate()
  const [product, setProduct] = useState<ProductView | null>(null)
  const [session, setSession] = useState<AuctionSession | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [actionMsg, setActionMsg] = useState<string | null>(null)
  const [showRulesEdit, setShowRulesEdit] = useState(false)
  const [cancelReason, setCancelReason] = useState('')
  const [showCancel, setShowCancel] = useState(false)

  const load = useCallback(async () => {
    const pid = Number(id)
    if (!pid) return
    setError(null)
    try {
      const p = await getProduct(pid)
      setProduct(p)
      if (p.auction?.sessionId) {
        const s = await getAuction(p.auction.sessionId)
        setSession(s)
      } else {
        setSession(null)
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : '加载失败')
    }
  }, [id])

  useEffect(() => {
    void load()
  }, [load])

  async function handleDelete() {
    if (!product || !confirm('确定删除/下架该商品？')) return
    try {
      await deleteProduct(product.id)
      navigate('/admin/products')
    } catch (e) {
      setActionMsg(e instanceof Error ? e.message : '操作失败')
    }
  }

  async function handleCancel() {
    if (!session || !cancelReason.trim()) {
      setActionMsg('请填写取消原因')
      return
    }
    try {
      await cancelAuction(session.id, cancelReason.trim())
      setShowCancel(false)
      setCancelReason('')
      setActionMsg('场次已取消')
      await load()
    } catch (e) {
      setActionMsg(e instanceof Error ? e.message : '取消失败')
    }
  }

  if (error) return <p className="form-error">{error}</p>
  if (!product) return <p className="muted">加载中…</p>

  const auction = product.auction
  const canEditRules =
    session?.status === 'pending' && (session.bidCount ?? 0) === 0
  const canCancel =
    session?.status === 'pending' || session?.status === 'running'
  const canPublish = product.status === 'listed' && !auction
  const canEditProduct =
    product.status === 'draft' || product.status === 'listed'

  return (
    <div className="admin-page">
      <div className="admin-page__head">
        <div>
          <Link to="/admin/products" className="breadcrumb">
            ← 商品列表
          </Link>
          <h2>{product.name}</h2>
          <div className="badge-row">
            <StatusBadge
              label={PRODUCT_STATUS_LABEL[product.status]}
              variant="product"
              tone={product.status}
            />
            {auction && (
              <StatusBadge
                label={SESSION_STATUS_LABEL[auction.status]}
                variant="session"
                tone={auction.status}
              />
            )}
          </div>
        </div>
        <div className="head-actions">
          {canEditProduct && (
            <Link to={`/admin/products/${product.id}/edit`} className="btn-ghost">
              编辑商品
            </Link>
          )}
          {canPublish && (
            <Link
              to={`/admin/products/${product.id}/auction`}
              className="btn-primary"
            >
              发布竞拍
            </Link>
          )}
          {product.status === 'draft' && (
            <button type="button" className="btn-danger" onClick={() => void handleDelete()}>
              删除
            </button>
          )}
        </div>
      </div>

      {actionMsg && <p className="form-info">{actionMsg}</p>}

      <div className="detail-grid">
        <section className="admin-card">
          <h3>商品信息</h3>
          {product.coverUrl && (
            <img src={product.coverUrl} alt="" className="detail-cover" />
          )}
          <p>{product.description || '暂无介绍'}</p>
          {product.images && product.images.length > 1 && (
            <div className="image-gallery">
              {product.images.map((url) => (
                <img key={url} src={url} alt="" />
              ))}
            </div>
          )}
        </section>

        <section className="admin-card">
          <h3>竞拍进度</h3>
          {!auction ? (
            <p className="muted">暂无竞拍场次</p>
          ) : (
            <dl className="detail-dl">
              <dt>场次 ID</dt>
              <dd>{auction.sessionId}</dd>
              <dt>房间号</dt>
              <dd>
                <code>{auction.roomId}</code>
              </dd>
              <dt>当前价</dt>
              <dd>{formatCents(auction.currentPrice)}</dd>
              <dt>出价 / 参与</dt>
              <dd>
                {auction.bidCount} 笔 / {auction.participantCount} 人
              </dd>
              {auction.scheduledStartAt && (
                <>
                  <dt>计划开拍</dt>
                  <dd>{new Date(auction.scheduledStartAt).toLocaleString('zh-CN')}</dd>
                </>
              )}
              {auction.endAt && (
                <>
                  <dt>结束时间</dt>
                  <dd>{new Date(auction.endAt).toLocaleString('zh-CN')}</dd>
                </>
              )}
              {auction.cancelReason && (
                <>
                  <dt>取消原因</dt>
                  <dd>{auction.cancelReason}</dd>
                </>
              )}
            </dl>
          )}

          {session && (
            <dl className="detail-dl rules-summary">
              <dt>起拍价</dt>
              <dd>{formatCents(session.rules.startingPrice)}</dd>
              <dt>加价幅度</dt>
              <dd>{formatCents(session.rules.bidIncrement)}</dd>
              <dt>封顶价</dt>
              <dd>
                {session.rules.capPrice != null
                  ? formatCents(session.rules.capPrice)
                  : '无封顶'}
              </dd>
              <dt>时长 / 延时</dt>
              <dd>
                {session.rules.durationSec}s · 结束前 {session.rules.extendThresholdSec}s 触发 +{session.rules.extendSec}s
              </dd>
            </dl>
          )}

          <div className="detail-actions">
            {canEditRules && (
              <button
                type="button"
                className="btn-ghost"
                onClick={() => setShowRulesEdit((v) => !v)}
              >
                {showRulesEdit ? '收起规则编辑' : '修改竞拍规则'}
              </button>
            )}
            {canCancel && (
              <button
                type="button"
                className="btn-danger"
                onClick={() => setShowCancel((v) => !v)}
              >
                取消场次
              </button>
            )}
          </div>

          {showRulesEdit && session && (
            <div className="inline-form-block">
              <AuctionRulesForm
                initial={{
                  ...session.rules,
                  scheduledStartAt: session.scheduledStartAt,
                }}
                submitLabel="保存规则"
                onSubmit={async (values) => {
                  await updateAuctionRules(session.id, {
                    startingPrice: values.startingPrice,
                    bidIncrement: values.bidIncrement,
                    capPrice: values.capPrice,
                    durationSec: values.durationSec,
                    extendThresholdSec: values.extendThresholdSec,
                    extendSec: values.extendSec,
                    scheduledStartAt: values.scheduledStartAt || undefined,
                  })
                  setShowRulesEdit(false)
                  setActionMsg('规则已更新')
                  await load()
                }}
              />
            </div>
          )}

          {showCancel && (
            <div className="inline-form-block cancel-box">
              <label>
                取消原因 *
                <input
                  value={cancelReason}
                  onChange={(e) => setCancelReason(e.target.value)}
                  placeholder="如：主播临时下架"
                />
              </label>
              <button type="button" className="btn-danger" onClick={() => void handleCancel()}>
                确认取消
              </button>
            </div>
          )}
        </section>

        {auction?.order && (
          <section className="admin-card">
            <h3>成交订单</h3>
            <dl className="detail-dl">
              <dt>订单号</dt>
              <dd>
                <Link to={`/admin/orders/${auction.order!.id}`}>
                  {auction.order!.orderNo}
                </Link>
              </dd>
              <dt>成交金额</dt>
              <dd>{formatCents(auction.order!.amount)}</dd>
              <dt>状态</dt>
              <dd>
                <StatusBadge
                  label={ORDER_STATUS_LABEL[auction.order!.status]}
                  variant="order"
                  tone={auction.order!.status}
                />
              </dd>
            </dl>
          </section>
        )}
      </div>
    </div>
  )
}
