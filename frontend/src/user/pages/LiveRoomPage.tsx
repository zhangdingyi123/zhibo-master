import { useCallback, useEffect, useMemo, useState } from 'react'
import { useParams, useSearchParams } from 'react-router-dom'
import { getAuction, getLiveRoom, type UserLiveRoomDetail } from '../../api/user'
import { getRoomStats, type RoomSocialStats } from '../../api/social'
import type { AnchorBrief } from '../../api/social'
import type { SessionSummary } from '../../api/types'
import { AuctionLiveRoom } from '../../components/auction/AuctionLiveRoom'
import type { SessionSwitchPayload } from '../../ws/types'

function buildStripItems(detail: UserLiveRoomDetail): SessionSummary[] {
  const items = [...detail.history]
  if (detail.current) {
    const s = detail.current.session
    const p = detail.current.product
    const current: SessionSummary = {
      sessionId: s.id,
      productId: s.productId,
      productName: p.name,
      coverUrl: p.coverUrl,
      status: s.status,
      finalPrice: s.currentPrice,
      winnerId: s.winnerId,
      seqInRoom: s.seqInRoom ?? 1,
    }
    if (!items.some((h) => h.sessionId === current.sessionId)) {
      items.push(current)
    }
  }
  return items.sort((a, b) => a.seqInRoom - b.seqInRoom)
}

export function LiveRoomPage() {
  const { roomId } = useParams<{ roomId: string }>()
  const [search] = useSearchParams()
  const sessionIdParam = search.get('session')
  const [roomDetail, setRoomDetail] = useState<UserLiveRoomDetail | null>(null)
  const [stripItems, setStripItems] = useState<SessionSummary[]>([])
  const [title, setTitle] = useState<string | undefined>()
  const [description, setDescription] = useState<string | undefined>()
  const [coverUrl, setCoverUrl] = useState<string | undefined>()
  const [sessionId, setSessionId] = useState<number | undefined>(
    sessionIdParam ? Number(sessionIdParam) : undefined,
  )
  const [scheduledStartAt, setScheduledStartAt] = useState<string | undefined>()
  const [anchor, setAnchor] = useState<AnchorBrief | null>(null)
  const [liveRoomTitle, setLiveRoomTitle] = useState<string | undefined>()
  const [productId, setProductId] = useState<number | undefined>()
  const [roomStats, setRoomStats] = useState<RoomSocialStats | null>(null)

  const multiSku = Boolean(roomId?.startsWith('room_live_'))
  const [loadError, setLoadError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    if (!roomId) return
    let cancelled = false
    setLoading(true)
    setLoadError(null)

    getLiveRoom(roomId)
      .then(async (d) => {
        if (cancelled) return
        setRoomDetail(d)
        setStripItems(buildStripItems(d))
        setAnchor(d.anchor ?? null)
        setLiveRoomTitle(d.liveRoom.title)
        if (d.current) {
          setTitle(d.current.product.name)
          setDescription(d.current.product.description)
          setCoverUrl(d.current.product.coverUrl)
          setSessionId(d.current.session.id)
          setProductId(d.current.product.id)
          setScheduledStartAt(d.current.session.scheduledStartAt)
        }
        try {
          const stats = await getRoomStats(roomId, d.current?.product.id)
          if (!cancelled) setRoomStats(stats)
        } catch {
          /* 社交接口未就绪时不影响进房 */
        }
      })
      .catch(() => {
        const id = sessionIdParam ? Number(sessionIdParam) : NaN
        if (!Number.isFinite(id)) {
          if (!cancelled) setLoadError('直播间加载失败，请稍后重试')
          return
        }
        return getAuction(id)
          .then((d) => {
            if (!cancelled) {
              setTitle(d.product.name)
              setDescription(d.product.description)
              setCoverUrl(d.product.coverUrl)
              setSessionId(d.session.id)
              setProductId(d.product.id)
            }
          })
          .catch(() => {
            if (!cancelled) setLoadError('直播间加载失败，请稍后重试')
          })
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })

    return () => {
      cancelled = true
    }
  }, [roomId, sessionIdParam])

  const handleSessionSwitch = useCallback((payload: SessionSwitchPayload) => {
    const items: SessionSummary[] = (payload.history ?? []).map((h) => ({
      sessionId: h.sessionId,
      productId: h.productId,
      productName: h.productName,
      coverUrl: h.coverUrl,
      status: h.status,
      finalPrice: h.finalPrice,
      winnerId: h.winnerId,
      seqInRoom: h.seqInRoom,
    }))
    if (payload.current) {
      const s = payload.current.session
      const p = payload.current.product
      if (!items.some((i) => i.sessionId === s.id)) {
        const nextSeq =
          items.length > 0
            ? Math.max(...items.map((i) => i.seqInRoom)) + 1
            : 1
        items.push({
          sessionId: s.id,
          productId: s.productId,
          productName: p.name,
          coverUrl: p.coverUrl,
          status: s.status,
          finalPrice: s.currentPrice,
          seqInRoom: nextSeq,
        })
      }
      setTitle(p.name)
      setDescription(p.description)
      setCoverUrl(p.coverUrl)
      setSessionId(s.id)
      setProductId(p.id)
      setScheduledStartAt(s.scheduledStartAt)
      if (roomId) {
        void getRoomStats(roomId, p.id).then(setRoomStats).catch(() => {})
      }
    }
    setStripItems(items.sort((a, b) => a.seqInRoom - b.seqInRoom))
  }, [roomId])

  const handleItemSettled = useCallback(
    (sid: number, finalPrice: number, winnerId?: number) => {
      setStripItems((prev) =>
        prev.map((item) =>
          item.sessionId === sid
            ? { ...item, status: 'settled' as const, finalPrice, winnerId }
            : item,
        ),
      )
    },
    [],
  )

  const currentSessionId = useMemo(
    () => sessionId ?? roomDetail?.current?.session.id,
    [sessionId, roomDetail],
  )

  if (loading) {
    return (
      <div className="live-room live-room--v2 live-room--loading">
        <p className="muted">正在进入直播间…</p>
      </div>
    )
  }

  if (loadError) {
    return (
      <div className="live-room live-room--v2 live-room--error">
        <p className="form-error">{loadError}</p>
        <p className="muted">请确认已在中控台添加商品并开播，或稍后刷新重试。</p>
      </div>
    )
  }

  return (
    <AuctionLiveRoom
      liveRoomStatus={roomDetail?.liveRoom.status}
      roomId={roomId ?? 'room_sess_1'}
      sessionId={currentSessionId}
      productId={productId}
      productTitle={title}
      productDescription={description}
      coverUrl={coverUrl}
      liveRoomTitle={liveRoomTitle}
      anchor={anchor}
      roomStats={roomStats}
      scheduledStartAt={scheduledStartAt}
      multiSku={multiSku}
      stripItems={stripItems}
      onSessionSwitch={handleSessionSwitch}
      onItemSettled={handleItemSettled}
    />
  )
}
