import type { AuctionRulesFormValues } from './components/AuctionRulesForm'

export interface AuctionRuleTemplate {
  id: string
  label: string
  description: string
  values: Omit<AuctionRulesFormValues, 'scheduledStartAt'>
}

/** 常用场次规则模板，一键套用减少重复配置 */
export const AUCTION_RULE_TEMPLATES: AuctionRuleTemplate[] = [
  {
    id: 'zero-start-10s-extend',
    label: '0 元起拍 + 10 秒延时',
    description: '2 分钟竞拍，结束前 10 秒内出价延时 10 秒',
    values: {
      startingPrice: 0,
      bidIncrement: 1000,
      capPrice: null,
      durationSec: 120,
      extendThresholdSec: 10,
      extendSec: 10,
    },
  },
  {
    id: 'flash-sale',
    label: '快节奏秒杀',
    description: '1 分钟短场，5 秒窗口延时',
    values: {
      startingPrice: 100,
      bidIncrement: 100,
      capPrice: null,
      durationSec: 60,
      extendThresholdSec: 5,
      extendSec: 10,
    },
  },
  {
    id: 'cap-deal',
    label: '封顶一口价',
    description: '0 元起拍，封顶 ¥999 立即成交',
    values: {
      startingPrice: 0,
      bidIncrement: 1000,
      capPrice: 99900,
      durationSec: 180,
      extendThresholdSec: 10,
      extendSec: 15,
    },
  },
]
