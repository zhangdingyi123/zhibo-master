import type { SessionSummary } from '../../api/types'
import { formatCents } from '../../utils/money'

type Props = {
  items: SessionSummary[]
  currentSessionId?: number
  onSelect?: (sessionId: number) => void
}

export function ProductStrip({ items, currentSessionId, onSelect }: Props) {
  if (items.length === 0) return null

  return (
    <div className="product-strip" role="tablist" aria-label="直播商品列表">
      {items.map((item) => {
        const isCurrent = item.sessionId === currentSessionId
        const isSettled = item.status === 'settled'
        return (
          <button
            key={item.sessionId}
            type="button"
            role="tab"
            aria-selected={isCurrent}
            className={`product-strip__item${isCurrent ? ' product-strip__item--active' : ''}${isSettled ? ' product-strip__item--done' : ''}`}
            onClick={() => onSelect?.(item.sessionId)}
          >
            {item.coverUrl && (
              <img src={item.coverUrl} alt="" className="product-strip__thumb" />
            )}
            <span className="product-strip__name">{item.productName}</span>
            {isSettled ? (
              <>
                <span className="product-strip__price">{formatCents(item.finalPrice)}</span>
                <span className="product-strip__badge product-strip__badge--recap">回看</span>
              </>
            ) : isCurrent ? (
              <span className="product-strip__badge">进行中</span>
            ) : (
              <span className="product-strip__badge product-strip__badge--wait">待拍</span>
            )}
          </button>
        )
      })}
    </div>
  )
}
