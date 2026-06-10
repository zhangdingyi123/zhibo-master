import { useCallback, useEffect, useState } from 'react'
import { Link, useLocation, useNavigate } from 'react-router-dom'
import { getToken, getUser, isLoggedIn } from '../../auth/session'
import { listRoomComments, type AnchorBrief, type RoomSocialStats } from '../../api/social'
import { useAuctionNotifications } from '../../hooks/useAuctionNotifications'
import { useBidThrottle } from '../../hooks/useBidThrottle'
import { useAuctionSocket } from '../../ws'
import {
  EventLikeUpdate,
  EventSessionSwitch,
  type LikeUpdatePayload,
  type SettledPayload,
} from '../../ws/types'
import { connectionLabel } from '../../utils/connectionLabel'
import { useSoundEnabled } from '../../hooks/useSoundEnabled'
import { ChevronLeftIcon, SoundOffIcon, SoundOnIcon } from '../icons/NavIcons'
import { BidPanel } from './BidPanel'
import { LiveDanmaku } from './LiveDanmaku'
import { LiveHostBar } from './LiveHostBar'
import { LiveInteractionDock } from './LiveInteractionDock'
import { LivePriceBoard } from './LivePriceBoard'
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
import { LiveRoomEmptyBanner } from './LiveRoomEmptyBanner'
import type { LiveRoomStatus } from '../../api/types'

type Props = {
  roomId?: string
  sessionId?: number
  productId?: number
  productTitle?: string
  productDescription?: string
  coverUrl?: string
  liveRoomTitle?: string
  liveRoomStatus?: LiveRoomStatus
  anchor?: AnchorBrief | null
  roomStats?: RoomSocialStats | null
  scheduledStartAt?: string
  multiSku?: boolean
  stripItems?: SessionSummary[]
  onSessionSwitch?: (payload: SessionSwitchPayload) => void
  onItemSettled?: (sessionId: number, finalPrice: number, winnerId?: number) => void
}

export function AuctionLiveRoom({
  roomId: roomIdProp = 'room_sess_1',
  sessionId,
  productId,
  productTitle,
  productDescription,
  coverUrl,
  liveRoomTitle,
  liveRoomStatus,
  anchor,
  roomStats,
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
  const [rankOpen, setRankOpen] = useState(false)
  const [likeCount, setLikeCount] = useState(roomStats?.likeCount ?? 0)
  const [isFollowing, setIsFollowing] = useState(roomStats?.isFollowing ?? false)
  const [isFavorited, setIsFavorited] = useState(roomStats?.isFavorited ?? false)
  const [seedComments, setSeedComments] = useState<
    { id: number; nickname: string; content: string }[]
  >([])
  const user = getUser()
  const token = getToken()

  useEffect(() => {
    setRoomId(roomIdProp)
  }, [roomIdProp])

  useEffect(() => {
    if (roomStats) {
      setLikeCount(roomStats.likeCount)
      setIsFollowing(roomStats.isFollowing ?? false)
      setIsFavorited(roomStats.isFavorited ?? false)
    }
  }, [roomStats])

  useEffect(() => {
    if (!roomId) return
    listRoomComments(roomId)
      .then((res) =>
        setSeedComments(
          res.items
            .filter((c) => !c.isHidden)
            .slice(0, 12)
            .map((c) => ({ id: c.id, nickname: c.nickname, content: c.content })),
        ),
      )
      .catch(() => {})
  }, [roomId])

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

  useEffect(() => {
    if (lastEvent?.type === EventLikeUpdate) {
      const p = lastEvent.payload as LikeUpdatePayload | undefined
      if (p?.likeCount != null) setLikeCount(p.likeCount)
    }
  }, [lastEvent])

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
    <div className={`live-room live-room--v2 ${outbidFlash ? 'live-room--outbid' : ''}`}>
      <div className="live-room__stage">
        <div className="live-room__video-wrap">
          <LiveVideo
            title={productTitle}
            coverUrl={coverUrl}
            viewerCount={snapshot?.participantCount}
          />
          <div className="live-room__overlay-top">
            <Link to="/app" className="live-room__back live-room__back--float" aria-label="返回列表">
              <ChevronLeftIcon />
            </Link>
            <LiveHostBar
              anchor={anchor}
              liveTitle={liveRoomTitle ?? productTitle}
              isFollowing={isFollowing}
              loginReturnTo={loginReturnTo}
              onFollowChange={setIsFollowing}
            />
            <div className="live-room__overlay-tools">
              <span
                className="live-room__conn conn-badge live-room__conn--float"
                data-state={connectionState}
              >
                {connectionLabel(connectionState)}
              </span>
              <button
                type="button"
                className={`btn-icon btn-icon--sm btn-icon--glass${soundEnabled ? '' : ' btn-icon--muted'}`}
                onClick={toggleSound}
                aria-label={soundEnabled ? '关闭提示音' : '开启提示音'}
              >
                {soundEnabled ? <SoundOnIcon /> : <SoundOffIcon />}
              </button>
              <button
                type="button"
                className={`btn-ghost btn-sm live-room__narration live-room__narration--float${narrationEnabled ? ' live-room__narration--on' : ''}`}
                onClick={toggleNarration}
                aria-pressed={narrationEnabled}
              >
                {narrationEnabled ? '解说' : '解说'}
              </button>
            </div>
          </div>
          <LiveDanmaku
            lastEvent={lastEvent}
            seedComments={seedComments}
          />
          <AICommentaryBar
            line={currentLine}
            visible={hasLines}
            voiceOn={narrationEnabled}
          />
        </div>

        {multiSku && stripItems.length > 0 && (
          <ProductStrip
            items={stripItems}
            currentSessionId={snapshot?.sessionId ?? sessionId}
            onSelect={handleStripSelect}
          />
        )}

        <div className="live-room__auction-panel">
          {multiSku && !productTitle && !snapshot && (
            <LiveRoomEmptyBanner liveTitle={liveRoomTitle} status={liveRoomStatus} />
          )}
          <LivePriceBoard snapshot={snapshot} connectionState={connectionState} />
          <button
            type="button"
            className="live-room__rank-toggle"
            aria-expanded={rankOpen}
            onClick={() => setRankOpen((v) => !v)}
          >
            {rankOpen ? '收起排名' : `实时排名 · ${snapshot?.participantCount ?? 0} 人参与`}
          </button>
          {rankOpen && (
            <div className="live-room__rank-sheet">
              <RankLeaderboard
                items={rank}
                currentUserId={currentUserId}
                participantCount={snapshot?.participantCount ?? 0}
              />
            </div>
          )}
        </div>
      </div>

      <ToastStack toasts={toasts} onDismiss={dismiss} />

      {multiSku && (
        <NextUpBanner
          visible={nextUp != null}
          productName={nextUp?.name ?? ''}
          coverUrl={nextUp?.coverUrl}
        />
      )}

      {scheduledStartAt && snapshot?.status === 'pending' && (
        <ScheduledStartBanner scheduledStartAt={scheduledStartAt} />
      )}

      <footer className="live-room__footer live-room__footer--v2">
        {multiSku && winnerBar && (
          <WinnerPayBar
            amount={winnerBar.amount}
            orderId={winnerBar.orderId}
            productName={winnerBar.productName}
            onDismiss={() => setWinnerBar(null)}
          />
        )}
        <LiveInteractionDock
          roomId={roomId}
          productId={productId}
          likeCount={likeCount}
          isFavorited={isFavorited}
          loginReturnTo={loginReturnTo}
          onLikeCount={setLikeCount}
          onFavoriteChange={setIsFavorited}
          onCommentSent={() => {
            /* danmaku picks up via WS comment.new */
          }}
        />
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
        <div className="live-room__footer-meta">
          {isLoggedIn() && user ? (
            <span className="muted">{user.nickname}</span>
          ) : (
            <Link to="/app/login" state={{ from: loginReturnTo }} className="btn-ghost btn-sm">
              登录出价
            </Link>
          )}
          {needsReconnect && (
            <button type="button" className="btn-ghost btn-sm" onClick={reconnect}>
              重连
            </button>
          )}
        </div>
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
