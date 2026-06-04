import { Link, useNavigate } from 'react-router-dom'
import { register } from '../../api/auth'
import { AuthForm } from '../../components/auth/AuthForm'
import { setSession } from '../../auth/session'

export function UserRegisterPage() {
  const navigate = useNavigate()

  return (
    <div className="user-page user-page--auth">
      <div className="auth-backdrop" aria-hidden />
      <header className="auth-header">
        <Link to="/app/login" className="auth-back">
          ← 返回登录
        </Link>
        <div className="auth-header__brand">
          <span className="auth-header__logo">拍</span>
          <div>
            <h1>创建账号</h1>
            <p>注册后即可参与直播竞拍</p>
          </div>
        </div>
      </header>
      <section className="user-card user-card--elevated">
        <AuthForm
          mode="register"
          defaultRole="buyer"
          onSubmit={async ({ phone, password, nickname, role }) => {
            const res = await register({
              phone,
              password,
              nickname: nickname!,
              role,
            })
            setSession(res.token, res.user)
            if (res.user.role === 'anchor') {
              navigate('/admin/products', { replace: true })
            } else {
              navigate('/app/profile', { replace: true })
            }
          }}
          footer={
            <p className="auth-switch">
              已有账号？<Link to="/app/login">去登录</Link>
            </p>
          }
        />
      </section>
    </div>
  )
}
