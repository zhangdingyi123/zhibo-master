import { useCallback, useEffect, useState } from 'react'
import { Link, useLocation, useNavigate } from 'react-router-dom'
import { getToken, getUser, isLoggedIn } from '../../auth/session'
import { useAuctionNotifications } from '../../hooks/useAuctionNotifications'
import { useBidThrottle } from '../../hooks/useBidThrottle'
import { useAuctionSocket } from '../../ws'
import type { SettledPayload } from '../../ws/types'
import { connectionLabel } from '../../utils/connectionLabel'
import { useSoundEnabled } from '../../hooks/useSoundEnabled'
import { ChevronLeftIcon, SoundOffIcon, SoundOnIcon } from '../icons/NavIcons'
import { BidPanel } from './BidPanel'
import { LiveDanmaku } from './LiveDanmaku'
import { LivePriceBoard } from './LivePriceBoard'
import { LiveReactions } from './LiveReactions'
import { LiveVideo } from './LiveVideo'
import { RankLeaderboard } from './RankLeaderboard'
import { ToastStack } from './ToastStack'

type Props = {
  roomId?: string
  sessionId?: number
  productTitle?: string
  coverUrl?: string
}

export function AuctionLiveRoom({
  roomId: roomIdProp = 'room_sess_1',
  sessionId,
  productTitle,
  coverUrl,
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
  })

  const currentUserId = user?.id ?? null
  const needsReconnect =
    connectionState === 'closed' || connectionState === 'reconnecting'

  const handleSettled = useCallback(
    (payload: SettledPayload) => {
      const sid = payload.session?.id ?? sessionId
      if (sid) {
        window.setTimeout(() => navigate(`/app/result/${sid}`), 1500)
      }
    },
    [navigate, sessionId],
  )

  const { soundEnabled, toggleSound } = useSoundEnabled()

  const { toasts, dismiss, outbidFlash } = useAuctionNotifications({
    currentUserId,
    rank,
    lastEvent,
    soundEnabled,
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
        <BidPanel
          snapshot={snapshot}
          canBid={canBid}
          isBidding={isBidding}
          cooling={cooling}
          connected={connected}
          error={displayError}
          showCatchUp={outbidFlash}
          loginReturnTo={loginReturnTo}
          onBid={handleBid}
        />
      </footer>
    </div>
  )
}
