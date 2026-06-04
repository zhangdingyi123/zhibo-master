import { useState, type ReactNode } from 'react'

type Mode = 'login' | 'register'

interface AuthFormProps {
  mode: Mode
  defaultRole?: 'buyer' | 'anchor'
  onSubmit: (values: {
    phone: string
    password: string
    nickname?: string
    role?: 'buyer' | 'anchor'
  }) => Promise<void>
  footer?: ReactNode
}

export function AuthForm({
  mode,
  defaultRole = 'buyer',
  onSubmit,
  footer,
}: AuthFormProps) {
  const [phone, setPhone] = useState('')
  const [password, setPassword] = useState('')
  const [nickname, setNickname] = useState('')
  const [role, setRole] = useState<'buyer' | 'anchor'>(defaultRole)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError(null)
    setLoading(true)
    try {
      await onSubmit({
        phone,
        password,
        nickname: mode === 'register' ? nickname : undefined,
        role: mode === 'register' ? role : undefined,
      })
    } catch (err) {
      setError(err instanceof Error ? err.message : '操作失败')
    } finally {
      setLoading(false)
    }
  }

  return (
    <form className="auth-form" onSubmit={handleSubmit}>
      <label>
        手机号
        <input
          type="tel"
          inputMode="numeric"
          autoComplete="tel"
          placeholder="11 位手机号"
          value={phone}
          onChange={(e) => setPhone(e.target.value)}
          required
        />
      </label>
      <label>
        密码
        <input
          type="password"
          autoComplete={mode === 'login' ? 'current-password' : 'new-password'}
          placeholder="至少 6 位"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          minLength={6}
          required
        />
      </label>
      {mode === 'register' && (
        <>
          <label>
            昵称
            <input
              value={nickname}
              onChange={(e) => setNickname(e.target.value)}
              placeholder="展示名称"
              required
            />
          </label>
          {defaultRole === 'buyer' && (
            <label>
              注册身份
              <select
                value={role}
                onChange={(e) => setRole(e.target.value as 'buyer' | 'anchor')}
              >
                <option value="buyer">买家（参与竞拍）</option>
                <option value="anchor">主播/商家（管理后台）</option>
              </select>
            </label>
          )}
        </>
      )}
      {error && <p className="form-error">{error}</p>}
      <button type="submit" className="btn-primary btn-block" disabled={loading}>
        {loading ? '请稍候…' : mode === 'login' ? '登录' : '注册'}
      </button>
      {footer}
    </form>
  )
}
