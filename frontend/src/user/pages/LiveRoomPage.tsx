import { useEffect, useState } from 'react'
import { useParams, useSearchParams } from 'react-router-dom'
import { getAuction } from '../../api/user'
import { AuctionLiveRoom } from '../../components/auction/AuctionLiveRoom'

export function LiveRoomPage() {
  const { roomId } = useParams<{ roomId: string }>()
  const [search] = useSearchParams()
  const sessionId = search.get('session')
  const [title, setTitle] = useState<string | undefined>()
  const [coverUrl, setCoverUrl] = useState<string | undefined>()

  useEffect(() => {
    const id = sessionId ? Number(sessionId) : NaN
    if (!Number.isFinite(id)) return
    let cancelled = false
    getAuction(id).then((d) => {
      if (!cancelled) {
        setTitle(d.product.name)
        setCoverUrl(d.product.coverUrl)
      }
    })
    return () => {
      cancelled = true
    }
  }, [sessionId])

  return (
    <AuctionLiveRoom
      roomId={roomId ?? 'room_sess_1'}
      sessionId={sessionId ? Number(sessionId) : undefined}
      productTitle={title}
      coverUrl={coverUrl}
    />
  )
}
