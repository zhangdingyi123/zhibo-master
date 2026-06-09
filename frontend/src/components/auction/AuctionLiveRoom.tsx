import { useCallback, useEffect, useState } from 'react'
import { Link, useLocation, useNavigate } from 'react-router-dom'
import { getToken, getUser, isLoggedIn } from '../../auth/session'
import { useAuctionNotifications } from '../../hooks/useAuctionNotifications'
import { useBidThrottle } from '../../hooks/useBidThrottle'
import { useAuctionSocket } from '../../ws'
import { EventSessionSwitch, type SettledPayload } from '../../ws/types'
import { connectionLabel } from '../../utils/connectionLabel'
import { useSoundEnabled } from '../../hooks/useSoundEnabled'
import { ChevronLeftIcon, SoundOffIcon, SoundOnIcon } from '../icons/NavIcons'
import { BidPanel } from './BidPanel'
import { LiveDanmaku } from './LiveDanmaku'
import { LivePriceBoard } from './LivePriceBoard'
import { LiveReactions } from './LiveReactions'
import { LiveVideo } from './LiveVideo'
import { NextUpBanner } from './NextUpBanner'
import { RankLeaderboard } from './RankLeaderboard'
import { SessionRecapSheet } from './SessionRecapSheet'
import type { SessionSummary } from '../../api/types'
import type { SessionSwitchPayload } from '../../ws/types'
import { ProductStrip } from './ProductStrip'
import { ScheduledStartBanner } from './ScheduledStartBanner'
import { ToastStack } from './ToastStack'
import { WinnerPayBar } from './WinnerPayBar'
import { AICommentaryBar } from './AICommentaryBar'
import { useNarrationVoice } from '../../hooks/useNarrationVoice'
import { useProductNarration } from '../../hooks/useProductNarration'

type Props = {
  roomId?: string
  sessionId?: number
  productTitle?: string
  productDescription?: string
  coverUrl?: string
  scheduledStartAt?: string
  multiSku?: boolean
  stripItems?: SessionSummary[]
  onSessionSwitch?: (payload: SessionSwitchPayload) => void
  onItemSettled?: (sessionId: number, finalPrice: number, winnerId?: number) => void
}

export function AuctionLiveRoom({
  roomId: roomIdProp = 'room_sess_1',
  sessionId,
  productTitle,
  productDescription,
  coverUrl,
  scheduledStartAt,
  multiSku = false,
  stripItems = [],
  onSessionSwitch,
  onItemSettled,
}: Props) {
  const navigate = useNavigate()
  const location = useLocation()
  const loginReturnTo = `${location.pathname}${location.search}`
  const [roomId, setRoomId] = useState(roomIdProp)
  const [rankOpen, setRankOpen] = useState(() =>
    typeof window !== 'undefined' ? window.matchMedia('(min-width: 520px)').matches : false,
  )
  const user = getUser()
  const token = getToken()

  useEffect(() => {
    setRoomId(roomIdProp)
  }, [roomIdProp])

  const {
    connectionState,
    snapshot,
    rank,
    canBid,
    lastError,
    lastEvent,
    isBidding,
    bid,
    reconnect,
  } = useAuctionSocket({
    roomId,
    token,
    openId: user?.openId ?? null,
    userId: user ? String(user.id) : null,
    enabled: Boolean(roomId),
    onSessionSwitch,
  })

  const currentUserId = user?.id ?? null
  const needsReconnect =
    connectionState === 'closed' || connectionState === 'reconnecting'

  const [nextUp, setNextUp] = useState<{ name: string; coverUrl?: string } | null>(null)
  const [recapSessionId, setRecapSessionId] = useState<number | null>(null)
  const [winnerBar, setWinnerBar] = useState<{
    orderId: number
    amount: number
    productName?: string
  } | null>(null)

  const handleSettled = useCallback(
    (payload: SettledPayload) => {
      const sid = payload.session?.id ?? sessionId
      const price =
        payload.snapshot?.currentPrice ?? payload.session?.currentPrice ?? 0
      const winnerId = payload.session?.winnerId ?? payload.snapshot?.winnerId

      if (sid) {
        onItemSettled?.(sid, price, winnerId)
      }

      if (multiSku) {
        if (
          currentUserId != null &&
          winnerId === currentUserId &&
          payload.order
        ) {
          setWinnerBar({
            orderId: payload.order.id,
            amount: payload.order.amount,
            productName: productTitle,
          })
        }
        return
      }

      if (sid) {
        window.setTimeout(() => navigate(`/app/result/${sid}`), 1500)
      }
    },
    [navigate, sessionId, multiSku, onItemSettled, currentUserId, productTitle],
  )

  useEffect(() => {
    if (!multiSku || lastEvent?.type !== EventSessionSwitch) return
    const p = lastEvent.payload as SessionSwitchPayload | undefined
    const product = p?.current?.product
    if (!product?.name) return

    setNextUp({ name: product.name, coverUrl: product.coverUrl })
    setWinnerBar(null)
    const t = window.setTimeout(() => setNextUp(null), 3200)
    return () => clearTimeout(t)
  }, [lastEvent, multiSku])

  const { soundEnabled, toggleSound } = useSoundEnabled()
  const { narrationEnabled, toggleNarration, speak } = useNarrationVoice()
  const { currentLine, hasLines } = useProductNarration(
    productDescription,
    productTitle,
    narrationEnabled,
    speak,
  )

  const { toasts, dismiss, outbidFlash } = useAuctionNotifications({
    currentUserId,
    rank,
    lastEvent,
    soundEnabled,
    multiSku,
    onSettled: handleSettled,
  })

  const { run: throttledBid, cooling } = useBidThrottle(300)
  const [throttleHint, setThrottleHint] = useState<string | null>(null)

  useEffect(() => {
    if (!throttleHint) return
    const t = window.setTimeout(() => setThrottleHint(null), 2000)
    return () => clearTimeout(t)
  }, [throttleHint])

  const handleBid = useCallback(
    (amountCents: number) => {
      const ok = throttledBid(() => bid(amountCents))
      setThrottleHint(ok ? null : '操作过快，请稍候')
    },
    [bid, throttledBid],
  )

  const handleStripSelect = useCallback(
    (sid: number) => {
      const item = stripItems.find((i) => i.sessionId === sid)
      if (item?.status === 'settled') {
        setRecapSessionId(sid)
      }
    },
    [stripItems],
  )

  const connected = connectionState === 'connected'
  const displayError = lastError?.message ?? throttleHint

  return (
    <div className={`live-room ${outbidFlash ? 'live-room--outbid' : ''}`}>
      <header className="live-room__header">
        <Link to="/app" className="live-room__back" aria-label="返回列表">
          <ChevronLeftIcon />
        </Link>
        <div className="live-room__title-wrap">
          <h1 className="live-room__title">{productTitle ?? '直播间竞拍'}</h1>
          <span
            className={`live-room__conn conn-badge`}
            data-state={connectionState}
          >
            {connectionLabel(connectionState)}
          </span>
        </div>
        <div className="live-room__toolbar">
          <button
            type="button"
            className={`btn-icon btn-icon--sm${soundEnabled ? '' : ' btn-icon--muted'}`}
            onClick={toggleSound}
            aria-label={soundEnabled ? '关闭提示音' : '开启提示音'}
            title={soundEnabled ? '提示音开' : '提示音关'}
          >
            {soundEnabled ? <SoundOnIcon /> : <SoundOffIcon />}
          </button>
          <button
            type="button"
            className={`btn-ghost btn-sm live-room__narration${narrationEnabled ? ' live-room__narration--on' : ''}`}
            onClick={toggleNarration}
            aria-pressed={narrationEnabled}
            title={narrationEnabled ? 'AI 语音解说开' : 'AI 语音解说关'}
          >
            {narrationEnabled ? '解说开' : '解说关'}
          </button>
          {isLoggedIn() && user ? (
            <span className="live-room__user muted" title={user.nickname}>
              {user.nickname}
            </span>
          ) : (
            <Link to="/app/login" state={{ from: loginReturnTo }} className="btn-ghost btn-sm">
              登录
            </Link>
          )}
          {needsReconnect && (
            <button type="button" className="btn-ghost btn-sm" onClick={reconnect}>
              重连
            </button>
          )}
        </div>
      </header>

      <ToastStack toasts={toasts} onDismiss={dismiss} />

      {multiSku && (
        <NextUpBanner
          visible={nextUp != null}
          productName={nextUp?.name ?? ''}
          coverUrl={nextUp?.coverUrl}
        />
      )}

      {multiSku && stripItems.length > 0 && (
        <ProductStrip
          items={stripItems}
          currentSessionId={snapshot?.sessionId ?? sessionId}
          onSelect={handleStripSelect}
        />
      )}

      {scheduledStartAt && snapshot?.status === 'pending' && (
        <ScheduledStartBanner scheduledStartAt={scheduledStartAt} />
      )}

      <div className="live-room__body">
        <div className="live-room__main">
          <div className="live-room__video-wrap">
            <LiveVideo
              title={productTitle}
              coverUrl={coverUrl}
              viewerCount={snapshot?.participantCount}
            />
            <LiveDanmaku
              lastEvent={lastEvent}
              participantCount={snapshot?.participantCount ?? 0}
            />
            <AICommentaryBar
              line={currentLine}
              visible={hasLines}
              voiceOn={narrationEnabled}
            />
            <LiveReactions />
          </div>
          <LivePriceBoard snapshot={snapshot} connectionState={connectionState} />
          <button
            type="button"
            className="live-room__rank-toggle"
            aria-expanded={rankOpen}
            onClick={() => setRankOpen((v) => !v)}
          >
            {rankOpen ? '收起排名' : `查看排名 (${snapshot?.participantCount ?? 0})`}
          </button>
        </div>

        <aside
          className={`live-room__side${rankOpen ? ' live-room__side--open' : ''}`}
          aria-hidden={!rankOpen}
        >
          <RankLeaderboard
            items={rank}
            currentUserId={currentUserId}
            participantCount={snapshot?.participantCount ?? 0}
          />
        </aside>
      </div>

      <footer className="live-room__footer">
        {multiSku && winnerBar && (
          <WinnerPayBar
            amount={winnerBar.amount}
            orderId={winnerBar.orderId}
            productName={winnerBar.productName}
            onDismiss={() => setWinnerBar(null)}
          />
        )}
        <BidPanel
          snapshot={snapshot}
          canBid={canBid}
          isBidding={isBidding}
          cooling={cooling}
          connected={connected}
          error={displayError}
          showCatchUp={outbidFlash}
          loginReturnTo={loginReturnTo}
          multiSku={multiSku}
          onBid={handleBid}
        />
      </footer>

      {multiSku && (
        <SessionRecapSheet
          sessionId={recapSessionId}
          onClose={() => setRecapSessionId(null)}
        />
      )}
    </div>
  )
}
