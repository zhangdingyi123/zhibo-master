import { useState } from 'react'
import type { AuctionRules } from '../../api/types'
import { centsToYuanInput, yuanInputToCents } from '../../utils/money'

export interface AuctionRulesFormValues {
  startingPrice: number
  bidIncrement: number
  capPrice: number | null
  durationSec: number
  extendThresholdSec: number
  extendSec: number
  scheduledStartAt: string
}

interface AuctionRulesFormProps {
  initial?: Partial<AuctionRules> & { scheduledStartAt?: string }
  onSubmit: (values: AuctionRulesFormValues) => Promise<void>
  submitLabel: string
}

function toLocalDatetime(iso?: string): string {
  if (!iso) return ''
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return ''
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}

function localDatetimeToISO(local: string): string | undefined {
  if (!local.trim()) return undefined
  const d = new Date(local)
  if (Number.isNaN(d.getTime())) return undefined
  return d.toISOString()
}

export function AuctionRulesForm({
  initial,
  onSubmit,
  submitLabel,
}: AuctionRulesFormProps) {
  const [startingYuan, setStartingYuan] = useState(
    centsToYuanInput(initial?.startingPrice ?? 0),
  )
  const [incrementYuan, setIncrementYuan] = useState(
    centsToYuanInput(initial?.bidIncrement ?? 1000),
  )
  const [capYuan, setCapYuan] = useState(
    initial?.capPrice != null ? centsToYuanInput(initial.capPrice) : '',
  )
  const [noCap, setNoCap] = useState(initial?.capPrice == null)
  const [durationSec, setDurationSec] = useState(String(initial?.durationSec ?? 120))
  const [extendThreshold, setExtendThreshold] = useState(
    String(initial?.extendThresholdSec ?? 10),
  )
  const [extendSec, setExtendSec] = useState(String(initial?.extendSec ?? 30))
  const [scheduled, setScheduled] = useState(toLocalDatetime(initial?.scheduledStartAt))
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError(null)

    const startingPrice = yuanInputToCents(startingYuan)
    const bidIncrement = yuanInputToCents(incrementYuan)
    if (startingPrice == null || bidIncrement == null) {
      setError('请填写有效的金额')
      return
    }
    if (bidIncrement <= 0) {
      setError('加价幅度须大于 0')
      return
    }

    let capPrice: number | null = null
    if (!noCap) {
      const cap = yuanInputToCents(capYuan)
      if (cap == null) {
        setError('请填写封顶价或勾选无封顶')
        return
      }
      capPrice = cap
    }

    const dur = Number.parseInt(durationSec, 10)
    const th = Number.parseInt(extendThreshold, 10)
    const ext = Number.parseInt(extendSec, 10)
    if (!dur || dur < 10) {
      setError('竞拍时长至少 10 秒')
      return
    }
    if (!th || ext < 10 || ext > 30) {
      setError('延时秒数须在 10–30 之间')
      return
    }

    setLoading(true)
    try {
      await onSubmit({
        startingPrice,
        bidIncrement,
        capPrice,
        durationSec: dur,
        extendThresholdSec: th,
        extendSec: ext,
        scheduledStartAt: localDatetimeToISO(scheduled) ?? '',
      })
    } catch (err) {
      setError(err instanceof Error ? err.message : '提交失败')
    } finally {
      setLoading(false)
    }
  }

  return (
    <form className="admin-form" onSubmit={handleSubmit}>
      <div className="admin-form__grid">
        <label>
          起拍价（元）
          <input
            type="number"
            min="0"
            step="0.01"
            value={startingYuan}
            onChange={(e) => setStartingYuan(e.target.value)}
          />
          <span className="field-hint">0 表示 0 元起拍</span>
        </label>
        <label>
          加价幅度（元）
          <input
            type="number"
            min="0.01"
            step="0.01"
            required
            value={incrementYuan}
            onChange={(e) => setIncrementYuan(e.target.value)}
          />
        </label>
        <label className="admin-form__span2">
          <span className="checkbox-row">
            <input
              type="checkbox"
              checked={noCap}
              onChange={(e) => setNoCap(e.target.checked)}
            />
            无封顶价
          </span>
          {!noCap && (
            <input
              type="number"
              min="0"
              step="0.01"
              placeholder="封顶价（元）"
              value={capYuan}
              onChange={(e) => setCapYuan(e.target.value)}
            />
          )}
        </label>
        <label>
          竞拍时长（秒）
          <input
            type="number"
            min="10"
            value={durationSec}
            onChange={(e) => setDurationSec(e.target.value)}
          />
        </label>
        <label>
          延时触发窗口（秒）
          <input
            type="number"
            min="1"
            value={extendThreshold}
            onChange={(e) => setExtendThreshold(e.target.value)}
          />
          <span className="field-hint">结束前 N 秒内有出价则延时</span>
        </label>
        <label>
          单次延时（秒）
          <input
            type="number"
            min="10"
            max="30"
            value={extendSec}
            onChange={(e) => setExtendSec(e.target.value)}
          />
        </label>
        <label>
          计划开拍时间（可选）
          <input
            type="datetime-local"
            value={scheduled}
            onChange={(e) => setScheduled(e.target.value)}
          />
        </label>
      </div>
      {error && <p className="form-error">{error}</p>}
      <div className="admin-form__actions">
        <button type="submit" className="btn-primary" disabled={loading}>
          {loading ? '提交中…' : submitLabel}
        </button>
      </div>
    </form>
  )
}
