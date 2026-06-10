import { useCallback, useEffect, useMemo, useState } from 'react'
import { useParams, useSearchParams } from 'react-router-dom'
import { getAuction, getLiveRoom, type UserLiveRoomDetail } from '../../api/user'
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

  const multiSku = Boolean(roomId?.startsWith('room_live_'))

  useEffect(() => {
    if (!roomId) return
    let cancelled = false
    getLiveRoom(roomId)
      .then((d) => {
        if (cancelled) return
        setRoomDetail(d)
        setStripItems(buildStripItems(d))
        if (d.current) {
          setTitle(d.current.product.name)
          setDescription(d.current.product.description)
          setCoverUrl(d.current.product.coverUrl)
          setSessionId(d.current.session.id)
          setScheduledStartAt(d.current.session.scheduledStartAt)
        }
      })
      .catch(() => {
        const id = sessionIdParam ? Number(sessionIdParam) : NaN
        if (!Number.isFinite(id)) return
        getAuction(id).then((d) => {
          if (!cancelled) {
            setTitle(d.product.name)
            setDescription(d.product.description)
            setCoverUrl(d.product.coverUrl)
            setSessionId(d.session.id)
          }
        })
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
      setScheduledStartAt(s.scheduledStartAt)
    }
    setStripItems(items.sort((a, b) => a.seqInRoom - b.seqInRoom))
  }, [])

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

  return (
    <AuctionLiveRoom
      roomId={roomId ?? 'room_sess_1'}
      sessionId={currentSessionId}
      productTitle={title}
      productDescription={description}
      coverUrl={coverUrl}
      scheduledStartAt={scheduledStartAt}
      multiSku={multiSku}
      stripItems={stripItems}
      onSessionSwitch={handleSessionSwitch}
      onItemSettled={handleItemSettled}
    />
  )
}
