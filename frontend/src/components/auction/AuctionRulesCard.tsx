import type { AuctionRules } from '../../api/types'
import { formatCents } from '../../utils/money'

type Props = {
  rules: AuctionRules
  status: string
}

export function AuctionRulesCard({ rules, status }: Props) {
  return (
    <section className="rules-card">
      <h3>竞拍规则</h3>
      <dl>
        <dt>起拍价</dt>
        <dd>{formatCents(rules.startingPrice)}</dd>
        <dt>加价幅度</dt>
        <dd>{formatCents(rules.bidIncrement)}</dd>
        <dt>竞拍时长</dt>
        <dd>{rules.durationSec} 秒</dd>
        <dt>延时规则</dt>
        <dd>
          结束前 {rules.extendThresholdSec ?? 10} 秒内有出价，延长{' '}
          {rules.extendSec ?? 30} 秒
        </dd>
        {rules.capPrice != null && (
          <>
            <dt>封顶价</dt>
            <dd>{formatCents(rules.capPrice)}</dd>
          </>
        )}
        <dt>状态</dt>
        <dd>{status}</dd>
      </dl>
    </section>
  )
}
