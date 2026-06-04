import { Link, useLocation, useNavigate } from 'react-router-dom'
import { login } from '../../api/auth'
import { AuthForm } from '../../components/auth/AuthForm'
import { setSession } from '../../auth/session'

export function UserLoginPage() {
  const navigate = useNavigate()
  const location = useLocation()
  const redirectTo =
    (location.state as { from?: string } | null)?.from ?? '/app/profile'

  return (
    <div className="user-page user-page--auth">
      <div className="auth-backdrop" aria-hidden />
      <header className="auth-header">
        <Link to="/app" className="auth-back">
          ← 返回
        </Link>
        <div className="auth-header__brand">
          <span className="auth-header__logo">拍</span>
          <div>
            <h1>欢迎回来</h1>
            <p>登录后继续参与直播竞拍</p>
          </div>
        </div>
      </header>
      <section className="user-card user-card--elevated">
        <AuthForm
          mode="login"
          onSubmit={async ({ phone, password }) => {
            const res = await login(phone, password)
            setSession(res.token, res.user)
            navigate(redirectTo, { replace: true })
          }}
          footer={
            <p className="auth-switch">
              还没有账号？<Link to="/app/register">立即注册</Link>
            </p>
          }
        />
        <details className="auth-demo">
          <summary>演示账号（密码均为 123456）</summary>
          <ul>
            <li>买家：13800000002 / 13800000003</li>
            <li>主播：13800000001</li>
          </ul>
        </details>
      </section>
    </div>
  )
}
