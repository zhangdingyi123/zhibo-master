import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { login } from '../../api/auth'
import { ADMIN_ACCOUNTS, setAdminOpenId } from '../../auth/adminAuth'
import { setSession } from '../../auth/session'
import { AuthForm } from '../../components/auth/AuthForm'

export function LoginPage() {
  const navigate = useNavigate()
  const [showMock, setShowMock] = useState(false)
  const [mockOpenId, setMockOpenId] = useState<string>(ADMIN_ACCOUNTS[0].openId)

  function handleMockLogin(e: React.FormEvent) {
    e.preventDefault()
    setAdminOpenId(mockOpenId)
    navigate('/admin/products', { replace: true })
  }

  return (
    <div className="admin-login">
      <div className="admin-login__card">
        <h1>商家 / 主播登录</h1>
        <p className="admin-login__hint">使用手机号与密码登录（JWT）</p>
        <AuthForm
          mode="login"
          onSubmit={async ({ phone, password }) => {
            const res = await login(phone, password)
            if (res.user.role !== 'anchor' && res.user.role !== 'admin') {
              throw new Error('该账号不是主播/商家，请使用买家端登录')
            }
            setSession(res.token, res.user)
            navigate('/admin/products', { replace: true })
          }}
          footer={
            <p className="admin-login__footer">
              没有账号？<Link to="/admin/register">主播注册</Link>
              {' · '}
              <a href="/app">用户端 H5</a>
            </p>
          }
        />
        <details
          className="auth-demo"
          open={showMock}
          onToggle={(e) => setShowMock((e.target as HTMLDetailsElement).open)}
        >
          <summary>开发演示：快速 Mock 登录</summary>
          <form onSubmit={handleMockLogin} className="mock-quick-form">
            <select value={mockOpenId} onChange={(e) => setMockOpenId(e.target.value)}>
              {ADMIN_ACCOUNTS.map((a) => (
                <option key={a.openId} value={a.openId}>
                  {a.label}
                </option>
              ))}
            </select>
            <button type="submit" className="btn-ghost btn-block">
              Mock 进入后台
            </button>
          </form>
        </details>
        <p className="admin-login__hint" style={{ marginTop: '0.75rem' }}>
          演示账号：13800000001，密码 123456
        </p>
      </div>
    </div>
  )
}
