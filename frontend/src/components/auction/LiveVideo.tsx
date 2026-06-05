import Hls from 'hls.js'
import { useEffect, useRef, useState } from 'react'
import { hlsPlayUrl } from '../../api/stream'

type Props = {
  roomId?: string
  title?: string
  coverUrl?: string
  viewerCount?: number
}

/** 直播画面：有推流时 HLS 播放，否则封面占位 */
export function LiveVideo({ roomId, title, coverUrl, viewerCount = 0 }: Props) {
  const videoRef = useRef<HTMLVideoElement>(null)
  const [isLive, setIsLive] = useState(false)
  const [displayViewers, setDisplayViewers] = useState(() =>
    Math.max(viewerCount, 12) + Math.floor(Math.random() * 40),
  )

  useEffect(() => {
    const base = Math.max(viewerCount, 8)
    setDisplayViewers(base + Math.floor(Math.random() * 30))
    const t = window.setInterval(() => {
      setDisplayViewers((v) => {
        const delta = Math.random() > 0.5 ? 1 : -1
        const next = v + delta
        return Math.max(base, Math.min(base + 80, next))
      })
    }, 2800)
    return () => clearInterval(t)
  }, [viewerCount])

  useEffect(() => {
    if (!roomId) return

    const video = videoRef.current
    if (!video) return

    const src = hlsPlayUrl(roomId)
    let hls: Hls | null = null
    let retryTimer: number | undefined
    let disposed = false

    const markOffline = () => {
      if (!disposed) setIsLive(false)
    }

    const attach = () => {
      if (disposed) return

      if (Hls.isSupported()) {
        hls?.destroy()
        hls = new Hls({
          enableWorker: true,
          lowLatencyMode: true,
        })
        hls.loadSource(src)
        hls.attachMedia(video)
        hls.on(Hls.Events.MANIFEST_PARSED, () => {
          if (disposed) return
          setIsLive(true)
          void video.play().catch(() => {})
        })
        hls.on(Hls.Events.ERROR, (_event, data) => {
          if (disposed || !data.fatal) return
          hls?.destroy()
          hls = null
          markOffline()
          scheduleRetry()
        })
        return
      }

      if (video.canPlayType('application/vnd.apple.mpegurl')) {
        video.src = src
        const onReady = () => {
          if (disposed) return
          setIsLive(true)
          void video.play().catch(() => {})
        }
        const onError = () => {
          markOffline()
          scheduleRetry()
        }
        video.addEventListener('loadedmetadata', onReady)
        video.addEventListener('error', onError)
        return () => {
          video.removeEventListener('loadedmetadata', onReady)
          video.removeEventListener('error', onError)
        }
      }
    }

    const scheduleRetry = () => {
      if (disposed || retryTimer) return
      retryTimer = window.setTimeout(() => {
        retryTimer = undefined
        attach()
      }, 5000)
    }

    attach()

    return () => {
      disposed = true
      if (retryTimer) window.clearTimeout(retryTimer)
      hls?.destroy()
      video.removeAttribute('src')
      video.load()
      setIsLive(false)
    }
  }, [roomId])

  const showFallback = !isLive

  return (
    <div className="live-video">
      <video
        ref={videoRef}
        className={`live-video__player${isLive ? ' live-video__player--active' : ''}`}
        playsInline
        muted
        autoPlay
      />
      {showFallback &&
        (coverUrl ? (
          <img className="live-video__cover" src={coverUrl} alt="" />
        ) : (
          <div className="live-video__gradient" />
        ))}
      {showFallback && (
        <>
          <div className="live-video__vignette" aria-hidden />
          <div className="live-video__scan" aria-hidden />
          <div className="live-video__shimmer" aria-hidden />
        </>
      )}
      <div className="live-video__badge">{isLive ? 'LIVE' : '等待推流'}</div>
      <div className="live-video__viewers">
        <span className="live-video__viewers-dot" aria-hidden />
        {displayViewers.toLocaleString('zh-CN')} 在看
      </div>
      {title && <div className="live-video__title">{title}</div>}
    </div>
  )
}
