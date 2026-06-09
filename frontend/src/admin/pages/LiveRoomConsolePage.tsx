import { useCallback, useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import {
  addSessionToLiveRoom,
  cancelAuction,
  endCurrentAndSwitch,
  endLiveRoom,
  getLiveRoom,
  listProducts,
  startLiveRoom,
} from '../../api/admin'
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

  useEffect(() => {
    if (!liveRoomId || detail?.liveRoom.status !== 'live') return
    const id = window.setInterval(() => {
      void refresh()
    }, 15_000)
    return () => clearInterval(id)
  }, [liveRoomId, detail?.liveRoom.status, refresh])

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
            <h3>添加商品到队列</h3>
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
        </>
      )}
    </div>
  )
}
