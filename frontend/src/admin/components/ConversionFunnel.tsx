import type { LiveRoomFunnel } from '../../api/types'

interface Props {
  funnel: LiveRoomFunnel
}

const STEPS = [
  { key: 'viewerCount' as const, label: '进房人数' },
  { key: 'bidderCount' as const, label: '出价人数' },
  { key: 'settledCount' as const, label: '成交' },
  { key: 'paidCount' as const, label: '支付完成' },
]

function rate(curr: number, prev: number): string | null {
  if (prev <= 0) return null
  return `${Math.round((curr / prev) * 100)}%`
}

export function ConversionFunnel({ funnel }: Props) {
  return (
    <section className="admin-card conversion-funnel">
      <h3>转化漏斗</h3>
      <p className="muted conversion-funnel__desc">
        进房 → 出价 → 成交 → 支付，判断是流量问题还是规则问题
      </p>
      <div className="conversion-funnel__steps">
        {STEPS.map((step, i) => {
          const value = funnel[step.key]
          const prev = i > 0 ? funnel[STEPS[i - 1].key] : null
          const conv = prev != null ? rate(value, prev) : null
          return (
            <div key={step.key} className="conversion-funnel__step">
              <span className="conversion-funnel__label">{step.label}</span>
              <strong className="conversion-funnel__value">{value}</strong>
              {conv != null && (
                <span className="conversion-funnel__rate">{conv}</span>
              )}
              {i < STEPS.length - 1 && (
                <span className="conversion-funnel__arrow" aria-hidden>
                  →
                </span>
              )}
            </div>
          )
        })}
      </div>
      {funnel.hint && (
        <p className="conversion-funnel__hint">{funnel.hint}</p>
      )}
    </section>
  )
}
