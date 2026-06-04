import { useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import type { SessionSnapshot } from '../../ws/types'
import { formatCents, centsToYuanInput, yuanInputToCents } from '../../utils/money'

type Props = {
  snapshot: SessionSnapshot | null
  canBid: boolean
  isBidding: boolean
  cooling: boolean
  connected: boolean
  error: string | null
  /** 被超越时突出「按最低加价」按钮 */
  showCatchUp?: boolean
  loginReturnTo?: string
  onBid: (amountCents: number) => void
}

export function BidPanel({
  snapshot,
  canBid,
  isBidding,
  cooling,
  connected,
  error,
  showCatchUp = false,
  loginReturnTo = '/app',
  onBid,
}: Props) {
  const minNext = snapshot?.minNextBid ?? 0
  const increment = snapshot?.rules.bidIncrement ?? 1000
  const cap = snapshot?.rules.capPrice

  const [customYuan, setCustomYuan] = useState('')
  const [localError, setLocalError] = useState<string | null>(null)

  useEffect(() => {
    if (minNext > 0) {
      setCustomYuan(centsToYuanInput(minNext))
    }
  }, [minNext])

  useEffect(() => {
    if (error) setLocalError(null)
  }, [error])

  const quickOptions = useMemo(() => {
    const opts = [
      { label: '最低出价', cents: minNext },
      { label: `+1 档`, cents: minNext + increment },
      { label: `+2 档`, cents: minNext + increment * 2 },
    ]
    if (cap != null) {
      return opts.filter((o) => o.cents <= cap)
    }
    return opts
  }, [minNext, increment, cap])

  const biddable =
    canBid &&
    connected &&
    snapshot &&
    (snapshot.status === 'pending' || snapshot.status === 'running')

  const disabled = !biddable || isBidding || cooling

  const submitCustom = () => {
    const cents = yuanInputToCents(customYuan)
    if (cents == null) {
      setLocalError('请输入有效金额')
      return
    }
    if (cents < minNext) {
      setLocalError(`出价不能低于 ${formatCents(minNext)}`)
      return
    }
    if (cap != null && cents > cap) {
      setLocalError(`出价不能超过封顶 ${formatCents(cap)}`)
      return
    }
    setLocalError(null)
    onBid(cents)
  }

  if (!canBid) {
    return (
      <section className="bid-panel bid-panel--guest">
        <p>登录后可出价</p>
        <span className="bid-panel__hint">当前为围观模式，可查看实时排名</span>
        <Link
          to="/app/login"
          state={{ from: loginReturnTo }}
          className="btn-primary btn-block bid-panel__login"
        >
          登录参与竞拍
        </Link>
      </section>
    )
  }

  if (!snapshot) return null

  if (snapshot.status === 'settled') {
    return (
      <section className="bid-panel bid-panel--ended">
        <p>竞拍已结束</p>
        {snapshot.winnerId != null && (
          <span>成交价 {formatCents(snapshot.currentPrice)}</span>
        )}
      </section>
    )
  }

  if (snapshot.status === 'cancelled') {
    return (
      <section className="bid-panel bid-panel--ended">
        <p>本场竞拍已取消</p>
      </section>
    )
  }

  return (
    <section className="bid-panel">
      <h2 className="bid-panel__title">出价</h2>
      <p className="bid-panel__min">
        最低 <strong>{formatCents(minNext)}</strong>
        {increment > 0 && (
          <span> · 加价幅度 {formatCents(increment)}</span>
        )}
      </p>

      {showCatchUp && minNext > 0 && (
        <button
          type="button"
          className="bid-catch-up"
          disabled={disabled}
          onClick={() => onBid(minNext)}
        >
          <span className="bid-catch-up__title">夺回领先</span>
          <span className="bid-catch-up__price">出价 {formatCents(minNext)}</span>
        </button>
      )}

      <div className="quick-bids">
        {quickOptions.map((opt) => (
          <button
            key={opt.label}
            type="button"
            className="quick-bid-btn"
            disabled={disabled}
            onClick={() => onBid(opt.cents)}
          >
            <span className="quick-bid-btn__label">{opt.label}</span>
            <span className="quick-bid-btn__price">{formatCents(opt.cents)}</span>
          </button>
        ))}
      </div>

      <div className="custom-bid">
        <label htmlFor="bid-yuan">自定义（元）</label>
        <div className="custom-bid__row">
          <input
            id="bid-yuan"
            type="text"
            inputMode="decimal"
            value={customYuan}
            onChange={(e) => setCustomYuan(e.target.value)}
            disabled={disabled}
          />
          <button
            type="button"
            className="bid-submit"
            disabled={disabled}
            onClick={submitCustom}
          >
            {isBidding ? '提交中…' : cooling ? '请稍候…' : '确认出价'}
          </button>
        </div>
      </div>

      {(localError || error) && (
        <p className="bid-error" role="alert">
          {localError ?? error}
        </p>
      )}
    </section>
  )
}
