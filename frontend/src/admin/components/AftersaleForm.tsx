import { useState } from 'react'
import { AFTERSALE_REASON_PRESETS } from '../labels'

type Props = {
  actionLabel: string
  busyLabel: string
  hint: string
  variant?: 'danger' | 'primary'
  disabled?: boolean
  onSubmit: (reason: string) => Promise<void>
}

export function AftersaleForm({
  actionLabel,
  busyLabel,
  hint,
  variant = 'danger',
  disabled,
  onSubmit,
}: Props) {
  const [preset, setPreset] = useState<string>(AFTERSALE_REASON_PRESETS[0])
  const [reason, setReason] = useState(AFTERSALE_REASON_PRESETS[0])
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function handleSubmit() {
    const text = reason.trim()
    if (!text) {
      setError('请填写原因')
      return
    }
    setBusy(true)
    setError(null)
    try {
      await onSubmit(text)
    } catch (e) {
      setError(e instanceof Error ? e.message : '操作失败')
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="aftersale-form">
      <p className="muted">{hint}</p>
      <label className="form-field">
        <span>原因模板</span>
        <select
          value={preset}
          onChange={(e) => {
            const v = e.target.value
            setPreset(v)
            if (v !== 'custom') setReason(v)
          }}
        >
          {AFTERSALE_REASON_PRESETS.map((p) => (
            <option key={p} value={p}>
              {p}
            </option>
          ))}
          <option value="custom">自定义…</option>
        </select>
      </label>
      <label className="form-field">
        <span>原因说明</span>
        <textarea
          value={reason}
          onChange={(e) => {
            setReason(e.target.value)
            setPreset('custom')
          }}
          rows={2}
          placeholder="如：买家误拍，双方协商取消"
        />
      </label>
      <button
        type="button"
        className={variant === 'danger' ? 'btn-danger' : 'btn-primary'}
        disabled={disabled || busy}
        onClick={() => void handleSubmit()}
      >
        {busy ? busyLabel : actionLabel}
      </button>
      {error && <p className="form-error">{error}</p>}
    </div>
  )
}
