import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import type { UserAuctionListItem } from '../../api/user'
import { auctionEntryPath } from '../../utils/auctionNav'
import { formatCents } from '../../utils/money'
import { useCountdown } from '../../hooks/useCountdown'
import { formatRemainingMs } from '../../utils/time'

type Props = {
  items: UserAuctionListItem[]
}

function SlideCountdown({ endAt }: { endAt?: string }) {
  const remainingMs = useCountdown(endAt, Boolean(endAt))
  if (remainingMs == null) return <span className="featured-slide__tag">火热竞拍中</span>
  return (
    <span className="featured-slide__tag featured-slide__tag--time">
      剩余 {formatRemainingMs(remainingMs)}
    </span>
  )
}

export function FeaturedCarousel({ items }: Props) {
  const slides = items.filter((x) => x.session.status === 'running').slice(0, 5)
  const [index, setIndex] = useState(0)

  useEffect(() => {
    if (slides.length <= 1) return
    const t = window.setInterval(() => {
      setIndex((i) => (i + 1) % slides.length)
    }, 4500)
    return () => clearInterval(t)
  }, [slides.length])

  if (slides.length === 0) return null

  const { session, product } = slides[index]!

  return (
    <section className="featured-carousel" aria-label="热门直播竞拍">
      <div className="featured-carousel__glow" aria-hidden />
      <Link to={auctionEntryPath(session)} className="featured-slide">
        <img src={product.coverUrl} alt="" className="featured-slide__bg" />
        <div className="featured-slide__overlay" />
        <div className="featured-slide__content">
          <span className="featured-slide__live">
            <span className="featured-slide__live-dot" aria-hidden />
            正在直播
          </span>
          <h2 className="featured-slide__title">{product.name}</h2>
          <p className="featured-slide__price">{formatCents(session.currentPrice)}</p>
          <div className="featured-slide__foot">
            <SlideCountdown endAt={session.endAt} />
            <span className="featured-slide__cta">立即进入 ›</span>
          </div>
        </div>
      </Link>
      {slides.length > 1 && (
        <div className="featured-carousel__dots" role="tablist" aria-label="轮播指示">
          {slides.map((s, i) => (
            <button
              key={s.session.id}
              type="button"
              role="tab"
              aria-selected={i === index}
              aria-label={`第 ${i + 1} 场`}
              className={i === index ? 'featured-dot active' : 'featured-dot'}
              onClick={() => setIndex(i)}
            />
          ))}
        </div>
      )}
    </section>
  )
}
