import { Link, useNavigate } from 'react-router-dom'
import { register } from '../../api/auth'
import { AuthForm } from '../../components/auth/AuthForm'
import { setSession } from '../../auth/session'

export function AdminRegisterPage() {
  const navigate = useNavigate()

  return (
    <div className="admin-login">
      <div className="admin-login__card">
        <h1>主播 / 商家注册</h1>
        <p className="admin-login__hint">注册后可进入管理后台发布商品与竞拍</p>
        <AuthForm
          mode="register"
          defaultRole="anchor"
          onSubmit={async ({ phone, password, nickname }) => {
            const res = await register({
              phone,
              password,
              nickname: nickname!,
              role: 'anchor',
            })
            setSession(res.token, res.user)
            navigate('/admin/products', { replace: true })
          }}
          footer={
            <p className="admin-login__footer">
              已有账号？<Link to="/admin/login">去登录</Link>
            </p>
          }
        />
      </div>
    </div>
  )
}
