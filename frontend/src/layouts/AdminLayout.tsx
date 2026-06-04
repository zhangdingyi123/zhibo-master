import { NavLink, Outlet, useNavigate } from 'react-router-dom'
import { ADMIN_ACCOUNTS, getAdminOpenId } from '../auth/adminAuth'
import { clearSession, getUser } from '../auth/session'

export function AdminLayout() {
  const navigate = useNavigate()
  const sessionUser = getUser()
  const openId = getAdminOpenId()
  const account =
    sessionUser ??
    ADMIN_ACCOUNTS.find((a) => a.openId === openId)

  function logout() {
    clearSession()
    navigate('/admin/login', { replace: true })
  }

  return (
    <div className="admin-shell">
      <aside className="admin-sidebar">
        <div className="admin-sidebar__brand">
          <span className="admin-sidebar__mark" aria-hidden>拍</span>
          <div>
            <span className="admin-sidebar__logo">直播竞拍</span>
            <span className="admin-sidebar__sub">商家管理后台</span>
          </div>
        </div>
        <nav className="admin-nav">
          <NavLink to="/admin" end>
            数据概览
          </NavLink>
          <NavLink to="/admin/products">商品管理</NavLink>
          <NavLink to="/admin/orders">订单管理</NavLink>
          <a href="/app" className="admin-nav__external" target="_blank" rel="noreferrer">
            用户端 H5 →
          </a>
        </nav>
        <div className="admin-sidebar__foot">
          <p className="admin-user">
            {sessionUser
              ? sessionUser.nickname
              : account && 'label' in account
                ? account.label
                : openId ?? '未登录'}
          </p>
          <button type="button" className="btn-ghost btn-sm" onClick={logout}>
            退出登录
          </button>
        </div>
      </aside>
      <div className="admin-main">
        <header className="admin-topbar">
          <h1 className="admin-topbar__title">主播工作台</h1>
        </header>
        <div className="admin-content">
          <Outlet />
        </div>
      </div>
    </div>
  )
}
