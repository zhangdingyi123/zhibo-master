import type { SessionSnapshot } from '../../ws/types'
import { connectionLabel } from '../../utils/connectionLabel'
import { formatCents } from '../../utils/money'
import { formatRemainingMs } from '../../utils/time'

type Props = {
  snapshot: SessionSnapshot | null
  connectionState: string
}

const STATUS_LABEL: Record<string, string> = {
  pending: '待开始',
  running: '竞拍中',
  settled: '已成交',
  cancelled: '已取消',
  failed: '异常结束',
}

export function LivePriceBoard({ snapshot, connectionState }: Props) {
  if (!snapshot) {
    return (
      <section className="price-board price-board--empty">
        <p>等待场次数据…</p>
        <span className="conn-badge" data-state={connectionState}>
          {connectionLabel(connectionState)}
        </span>
      </section>
    )
  }

  const isRunning = snapshot.status === 'running'
  const urgent =
    isRunning && snapshot.remainingMs > 0 && snapshot.remainingMs <= 10_000

  return (
    <section className={`price-board ${urgent ? 'price-board--urgent' : ''}`}>
      <div className="price-board__meta">
        <span className={`status-pill status-pill--${snapshot.status}`}>
          {STATUS_LABEL[snapshot.status] ?? snapshot.status}
        </span>
        <span className="conn-badge" data-state={connectionState}>
          {connectionLabel(connectionState)}
        </span>
      </div>

      <p className="price-board__label">当前最高价</p>
      <p className="price-board__price">{formatCents(snapshot.currentPrice)}</p>

      <div className="price-board__stats">
        <div>
          <span className="stat-label">下一手最低</span>
          <strong>{formatCents(snapshot.minNextBid)}</strong>
        </div>
        <div>
          <span className="stat-label">出价 / 参与</span>
          <strong>
            {snapshot.bidCount} / {snapshot.participantCount}
          </strong>
        </div>
        {isRunning && (
          <div className="price-board__countdown">
            <span className="stat-label">剩余</span>
            <strong className="countdown-value">
              {formatRemainingMs(snapshot.remainingMs)}
            </strong>
          </div>
        )}
      </div>

      {snapshot.rules.capPrice != null && (
        <p className="price-board__cap">
          封顶 {formatCents(snapshot.rules.capPrice)}
        </p>
      )}
    </section>
  )
}
