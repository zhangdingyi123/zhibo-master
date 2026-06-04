import { useEffect, useState } from 'react'

type Props = {
  title?: string
  coverUrl?: string
  viewerCount?: number
}

/** 模拟直播画面：封面 + 动态扫描线 + 观看人数 */
export function LiveVideo({ title, coverUrl, viewerCount = 0 }: Props) {
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

  return (
    <div className="live-video">
      {coverUrl ? (
        <img className="live-video__cover" src={coverUrl} alt="" />
      ) : (
        <div className="live-video__gradient" />
      )}
      <div className="live-video__vignette" aria-hidden />
      <div className="live-video__scan" aria-hidden />
      <div className="live-video__shimmer" aria-hidden />
      <div className="live-video__badge">LIVE</div>
      <div className="live-video__viewers">
        <span className="live-video__viewers-dot" aria-hidden />
        {displayViewers.toLocaleString('zh-CN')} 在看
      </div>
      {title && <div className="live-video__title">{title}</div>}
    </div>
  )
}
