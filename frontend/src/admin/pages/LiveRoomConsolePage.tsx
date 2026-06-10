import { useCallback, useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import {
  addSessionToLiveRoom,
  addSessionsBatchToLiveRoom,
  cancelAuction,
  createProduct,
  endCurrentAndSwitch,
  endLiveRoom,
  getLiveRoom,
  hideRoomComment,
  listProducts,
  listRoomCommentsAdmin,
  startLiveRoom,
} from '../../api/admin'
import type { RoomComment } from '../../api/social'
import type { LiveRoomDetail, ProductView, PublishAuctionBody } from '../../api/types'
import { formatCents } from '../../utils/money'
import { formatRemainingMs } from '../../utils/time'
import { SESSION_STATUS_LABEL } from '../labels'
import { AuctionRulesForm } from '../components/AuctionRulesForm'
import { ConversionFunnel } from '../components/ConversionFunnel'
import { useAuctionSocket } from '../../ws'

export function LiveRoomConsolePage() {
  const { id } = useParams()
  const liveRoomId = Number(id)
  const [detail, setDetail] = useState<LiveRoomDetail | null>(null)
  const [products, setProducts] = useState<ProductView[]>([])
  const [productId, setProductId] = useState<number | ''>('')
  const [batchIds, setBatchIds] = useState<number[]>([])
  const [comments, setComments] = useState<RoomComment[]>([])
  const [quickName, setQuickName] = useState('')
  const [quickPrice, setQuickPrice] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  const [showEmergency, setShowEmergency] = useState(false)
  const [cancelReason, setCancelReason] = useState('')

  const roomId = detail?.liveRoom.roomId ?? ''
  const currentSession = detail?.currentSession?.session
  const currentSessionId = currentSession?.id

  const { snapshot } = useAuctionSocket({
    roomId,
    enabled: Boolean(roomId && detail?.liveRoom.status === 'live'),
  })

  const refresh = useCallback(async () => {
    if (!liveRoomId) return
    const d = await getLiveRoom(liveRoomId)
    setDetail(d)
  }, [liveRoomId])

  useEffect(() => {
    if (!liveRoomId) return
    let cancelled = false
    Promise.all([
      getLiveRoom(liveRoomId),
      listProducts({ page: 1, pageSize: 100, status: 'listed' }),
    ])
      .then(([d, pRes]) => {
        if (!cancelled) {
          setDetail(d)
          setProducts(pRes.items)
        }
      })
      .catch((e) => {
        if (!cancelled) setError(e instanceof Error ? e.message : '加载失败')
      })
    return () => {
      cancelled = true
    }
  }, [liveRoomId])

  const refreshComments = useCallback(async () => {
    if (!roomId) return
    try {
      const res = await listRoomCommentsAdmin(roomId)
      setComments(res.items)
    } catch {
      /* ignore */
    }
  }, [roomId])

  useEffect(() => {
    if (!liveRoomId || detail?.liveRoom.status !== 'live') return
    const id = window.setInterval(() => {
      void refresh()
      void refreshComments()
    }, 15_000)
    return () => clearInterval(id)
  }, [liveRoomId, detail?.liveRoom.status, refresh, refreshComments])

  useEffect(() => {
    if (roomId) void refreshComments()
  }, [roomId, refreshComments])

  if (!liveRoomId) {
    return <p className="form-error">无效的直播 ID</p>
  }

  const lr = detail?.liveRoom
  const livePrice = snapshot?.currentPrice ?? currentSession?.currentPrice ?? 0
  const liveStatus = snapshot?.status ?? currentSession?.status
  const isRunning = liveStatus === 'running'
  const remainingMs = snapshot?.remainingMs ?? 0
  const urgent = isRunning && remainingMs > 0 && remainingMs <= 10_000
  const canEmergencyCancel =
    currentSession &&
    (currentSession.status === 'pending' || currentSession.status === 'running')

  async function runAction(fn: () => Promise<unknown>) {
    setBusy(true)
    setError(null)
    try {
      await fn()
      await refresh()
    } catch (e) {
      setError(e instanceof Error ? e.message : '操作失败')
    } finally {
      setBusy(false)
    }
  }

  async function handleEmergencyCancel() {
    if (!currentSessionId || !cancelReason.trim()) {
      setError('请填写下架原因')
      return
    }
    await runAction(async () => {
      await cancelAuction(currentSessionId, cancelReason.trim())
      setShowEmergency(false)
      setCancelReason('')
    })
  }

  return (
    <div className="admin-page live-room-console">
      <div className="admin-page__head">
        <div>
          <h2>{lr?.title ?? '直播中控台'}</h2>
          <p className="page-desc">
            {lr?.roomId}{' '}
            {lr && (
              <a href={`/app/live/${lr.roomId}`} target="_blank" rel="noreferrer">
                打开用户端直播间 →
              </a>
            )}
          </p>
          <Link to="/admin/live-rooms" className="breadcrumb">
            ← 返回直播列表
          </Link>
        </div>
        <div className="live-room-console__actions">
          {lr?.status === 'idle' && (
            <button
              type="button"
              className="btn-primary"
              disabled={busy}
              onClick={() => runAction(() => startLiveRoom(liveRoomId))}
            >
              开播
            </button>
          )}
          {lr?.status === 'live' && (
            <>
              <button
                type="button"
                className="btn-primary"
                disabled={busy || !detail?.currentSession}
                onClick={() => runAction(() => endCurrentAndSwitch(liveRoomId))}
              >
                一键切品
              </button>
              {canEmergencyCancel && (
                <button
                  type="button"
                  className="btn-danger"
                  disabled={busy}
                  onClick={() => setShowEmergency((v) => !v)}
                >
                  紧急下架
                </button>
              )}
              <button
                type="button"
                className="btn-ghost"
                disabled={busy}
                onClick={() => runAction(() => endLiveRoom(liveRoomId))}
              >
                下播
              </button>
            </>
          )}
        </div>
      </div>

      {error && <p className="form-error">{error}</p>}

      {showEmergency && canEmergencyCancel && (
        <section className="admin-card admin-card--danger">
          <h3>紧急下架当前品</h3>
          <p className="muted">立即取消本场竞拍并通知房间内用户，适用于误上架或违规商品。</p>
          <label>
            下架原因
            <input
              value={cancelReason}
              onChange={(e) => setCancelReason(e.target.value)}
              placeholder="如：商品信息有误"
            />
          </label>
          <div className="admin-form__actions">
            <button
              type="button"
              className="btn-danger"
              disabled={busy}
              onClick={() => void handleEmergencyCancel()}
            >
              确认下架
            </button>
            <button
              type="button"
              className="btn-ghost"
              onClick={() => setShowEmergency(false)}
            >
              取消
            </button>
          </div>
        </section>
      )}

      {!detail ? (
        <p className="muted">加载中…</p>
      ) : (
        <>
          <ConversionFunnel funnel={detail.funnel ?? { viewerCount: 0, bidderCount: 0, settledCount: 0, paidCount: 0 }} />

          <section className="admin-card live-room-console__current">
            <h3>当前品 · 实时看板</h3>
            {detail.currentSession ? (
              <div className="live-room-console__board">
                <div className="live-room-console__product">
                  {detail.currentSession.product.coverUrl && (
                    <img
                      src={detail.currentSession.product.coverUrl}
                      alt=""
                      className="live-room-console__thumb"
                    />
                  )}
                  <div>
                    <strong>{detail.currentSession.product.name}</strong>
                    <p className="muted">
                      {SESSION_STATUS_LABEL[liveStatus ?? currentSession!.status]}
                      {snapshot && (
                        <>
                          {' '}
                          · {snapshot.bidCount} 笔 / {snapshot.participantCount} 人
                        </>
                      )}
                    </p>
                  </div>
                </div>
                <div className="live-room-console__metrics">
                  <div className="console-metric">
                    <span className="console-metric__label">当前价</span>
                    <strong className="console-metric__value console-metric__value--price">
                      {formatCents(livePrice)}
                    </strong>
                  </div>
                  {isRunning && (
                    <div
                      className={`console-metric console-metric--countdown${urgent ? ' console-metric--urgent' : ''}`}
                    >
                      <span className="console-metric__label">倒计时</span>
                      <strong className="console-metric__value">
                        {formatRemainingMs(remainingMs)}
                      </strong>
                    </div>
                  )}
                  {currentSession?.scheduledStartAt &&
                    currentSession.status === 'pending' && (
                      <div className="console-metric">
                        <span className="console-metric__label">预约开拍</span>
                        <strong className="console-metric__value">
                          {new Date(currentSession.scheduledStartAt).toLocaleString('zh-CN')}
                        </strong>
                      </div>
                    )}
                </div>
              </div>
            ) : (
              <p className="muted">暂无进行中的品，从队列上架或添加新商品</p>
            )}
          </section>

          <section className="admin-card">
            <h3>待上架队列</h3>
            {detail.queue.length === 0 ? (
              <p className="muted">队列为空</p>
            ) : (
              <ul className="live-room-queue">
                {detail.queue.map((item) => (
                  <li key={item.session.id}>
                    <span>{item.product.name}</span>
                    <span className="muted">
                      #{item.session.seqInRoom ?? item.session.id}
                      {item.session.scheduledStartAt && (
                        <> · 预约 {new Date(item.session.scheduledStartAt).toLocaleString('zh-CN', { hour: '2-digit', minute: '2-digit' })}</>
                      )}
                    </span>
                  </li>
                ))}
              </ul>
            )}
          </section>

          <section className="admin-card">
            <h3>已拍完</h3>
            {detail.history.length === 0 ? (
              <p className="muted">暂无历史成交</p>
            ) : (
              <ul className="live-room-queue">
                {detail.history.map((item) => (
                  <li key={item.session.id}>
                    <span>{item.product.name}</span>
                    <span>{formatCents(item.session.currentPrice)}</span>
                  </li>
                ))}
              </ul>
            )}
          </section>

          <section className="admin-card">
            <h3>直播中快速上架</h3>
            <p className="muted">创建简易商品并自动上架，可立即加入连拍队列</p>
            <div className="live-room-quick-create">
              <input
                value={quickName}
                onChange={(e) => setQuickName(e.target.value)}
                placeholder="商品名称"
              />
              <input
                value={quickPrice}
                onChange={(e) => setQuickPrice(e.target.value)}
                placeholder="起拍价（元）"
                inputMode="decimal"
              />
              <button
                type="button"
                className="btn-secondary"
                disabled={busy || !quickName.trim()}
                onClick={() =>
                  runAction(async () => {
                    const yuan = parseFloat(quickPrice) || 1
                    const p = await createProduct({
                      name: quickName.trim(),
                      description: `${quickName.trim()} — 直播专场`,
                      coverUrl: '',
                      images: [],
                    })
                    const cents = Math.round(yuan * 100)
                    await addSessionToLiveRoom(liveRoomId, {
                      productId: p.id,
                      startingPrice: cents,
                      bidIncrement: Math.max(100, Math.round(cents * 0.1)),
                      durationSec: 120,
                      extendThresholdSec: 10,
                      extendSec: 15,
                    })
                    setQuickName('')
                    setQuickPrice('')
                    const pRes = await listProducts({ page: 1, pageSize: 100, status: 'listed' })
                    setProducts(pRes.items)
                  })
                }
              >
                创建并加队列
              </button>
            </div>
          </section>

          <section className="admin-card">
            <h3>批量添加商品到队列</h3>
            <p className="muted">勾选多个已上架商品，共用一套竞拍规则一次性入队</p>
            <ul className="live-room-batch-list">
              {products.map((p) => (
                <li key={p.id}>
                  <label>
                    <input
                      type="checkbox"
                      checked={batchIds.includes(p.id)}
                      onChange={(e) => {
                        setBatchIds((prev) =>
                          e.target.checked
                            ? [...prev, p.id]
                            : prev.filter((id) => id !== p.id),
                        )
                      }}
                    />
                    {p.name}
                  </label>
                </li>
              ))}
            </ul>
            {batchIds.length > 0 && (
              <AuctionRulesForm
                submitLabel={`批量加入 (${batchIds.length} 件)`}
                onSubmit={async (values) => {
                  await addSessionsBatchToLiveRoom(liveRoomId, {
                    productIds: batchIds,
                    startingPrice: values.startingPrice,
                    bidIncrement: values.bidIncrement,
                    capPrice: values.capPrice,
                    durationSec: values.durationSec,
                    extendThresholdSec: values.extendThresholdSec,
                    extendSec: values.extendSec,
                    scheduledStartAt: values.scheduledStartAt || undefined,
                  })
                  setBatchIds([])
                  await refresh()
                }}
              />
            )}
          </section>

          <section className="admin-card">
            <h3>单个添加商品</h3>
            <label>
              选择商品
              <select
                value={productId}
                onChange={(e) =>
                  setProductId(e.target.value ? Number(e.target.value) : '')
                }
              >
                <option value="">请选择</option>
                {products.map((p) => (
                  <option key={p.id} value={p.id}>
                    {p.name}
                  </option>
                ))}
              </select>
            </label>
            {productId !== '' && (
              <AuctionRulesForm
                submitLabel="加入队列"
                onSubmit={async (values) => {
                  const body: PublishAuctionBody & { productId: number } = {
                    productId: productId as number,
                    startingPrice: values.startingPrice,
                    bidIncrement: values.bidIncrement,
                    capPrice: values.capPrice,
                    durationSec: values.durationSec,
                    extendThresholdSec: values.extendThresholdSec,
                    extendSec: values.extendSec,
                    scheduledStartAt: values.scheduledStartAt || undefined,
                  }
                  await addSessionToLiveRoom(liveRoomId, body)
                  setProductId('')
                  await refresh()
                }}
              />
            )}
          </section>

          <section className="admin-card">
            <h3>评论管理</h3>
            <p className="muted">屏蔽不当评论，房间内用户将不再看到</p>
            {comments.length === 0 ? (
              <p className="muted">暂无评论</p>
            ) : (
              <ul className="live-room-comments-admin">
                {comments.map((c) => (
                  <li key={c.id} className={c.isHidden ? 'live-room-comments-admin__hidden' : ''}>
                    <div>
                      <strong>{c.nickname}</strong>
                      <span className="muted"> · {new Date(c.createdAt).toLocaleTimeString('zh-CN')}</span>
                      <p>{c.content}</p>
                    </div>
                    {!c.isHidden && (
                      <button
                        type="button"
                        className="btn-ghost btn-sm"
                        disabled={busy}
                        onClick={() =>
                          runAction(async () => {
                            await hideRoomComment(c.id)
                            await refreshComments()
                          })
                        }
                      >
                        屏蔽
                      </button>
                    )}
                  </li>
                ))}
              </ul>
            )}
          </section>
        </>
      )}
    </div>
  )
}
