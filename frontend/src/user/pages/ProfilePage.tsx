import { useEffect, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { listMyOrders } from '../../api/user'
import {
  HistoryIcon,
  LiveIcon,
  OrderIcon,
  ProfileIcon,
  SettingsIcon,
} from '../../components/icons/NavIcons'
import { IsoInboxIcon } from '../../components/icons/IsometricIcons'
import { useUnreadMessageCount } from '../../hooks/useUnreadMessageCount'
import { clearSession, getUser, isAnchorOrAdmin, isLoggedIn } from '../../auth/session'

export function ProfilePage() {
  const navigate = useNavigate()
  const user = getUser()
  const loggedIn = isLoggedIn()
  const { count: unreadMessages } = useUnreadMessageCount()
  const [orderStats, setOrderStats] = useState({
    total: 0,
    pendingPay: 0,
    paid: 0,
  })

  useEffect(() => {
    if (!loggedIn) return
    let cancelled = false
    listMyOrders({ page: 1, pageSize: 100 })
      .then((res) => {
        if (cancelled) return
        setOrderStats({
          total: res.total,
          pendingPay: res.items.filter((o) => o.order.status === 'pending_pay').length,
          paid: res.items.filter(
            (o) => o.order.status === 'paid' || o.order.status === 'closed',
          ).length,
        })
      })
      .catch(() => {})
    return () => {
      cancelled = true
    }
  }, [loggedIn])

  function logout() {
    clearSession()
    navigate('/app/login', { replace: true })
  }

  if (!loggedIn || !user) {
    return (
      <div className="user-page user-page--profile">
        <header className="page-hero page-hero--compact">
          <div className="page-hero__content">
            <h1 className="page-hero__title">我的</h1>
            <p className="page-hero__sub">登录后参与竞拍与管理订单</p>
          </div>
        </header>
        <section className="user-card user-card--center user-card--elevated">
          <div className="avatar-placeholder" aria-hidden>
            <ProfileIcon className="avatar-placeholder__icon" />
          </div>
          <p className="user-card__lead">登录后可出价、查看订单与支付</p>
          <Link to="/app/login" className="btn-primary btn-block">
            登录
          </Link>
          <Link to="/app/register" className="btn-secondary btn-block">
            注册新账号
          </Link>
        </section>
      </div>
    )
  }

  const roleLabel =
    user.role === 'buyer' ? '买家' : user.role === 'anchor' ? '主播' : '管理员'

  return (
    <div className="user-page user-page--profile">
      <header className="profile-hero profile-hero--rich">
        <div className="profile-hero__bg" aria-hidden />
        <div className="profile-hero__avatar-wrap">
          {user.avatar ? (
            <img src={user.avatar} alt="" className="profile-avatar profile-avatar--lg" />
          ) : (
            <div className="avatar-placeholder avatar-placeholder--lg" aria-hidden>
              <span>{user.nickname.slice(0, 1)}</span>
            </div>
          )}
        </div>
        <div className="profile-hero__info">
          <h1>{user.nickname}</h1>
          <p className="profile-hero__meta">
            {user.phone ?? user.openId}
            <span className="role-chip">{roleLabel}</span>
          </p>
        </div>
        <button type="button" className="btn-ghost btn-sm" onClick={logout}>
          退出
        </button>
      </header>

      <div className="profile-stats">
        <Link to="/app/orders" className="profile-stat">
          <strong>{orderStats.total}</strong>
          <span>全部订单</span>
        </Link>
        <Link to="/app/orders" className="profile-stat profile-stat--warn">
          <strong>{orderStats.pendingPay}</strong>
          <span>待支付</span>
        </Link>
        <div className="profile-stat">
          <strong>{orderStats.paid}</strong>
          <span>已完成</span>
        </div>
      </div>

      <section className="menu-card">
        <h3 className="menu-card__title">快捷入口</h3>
        <ul className="menu-list">
          <li>
            <Link to="/app/messages" className="menu-list__item menu-list__item--glass">
              <span className="menu-list__icon menu-list__icon--iso" aria-hidden>
                <IsoInboxIcon />
              </span>
              <span className="menu-list__label">消息中心</span>
              {unreadMessages > 0 && (
                <span className="menu-list__badge">{unreadMessages}</span>
              )}
              <span className="menu-list__arrow" aria-hidden>›</span>
            </Link>
          </li>
          <li>
            <Link to="/app/orders" className="menu-list__item">
              <span className="menu-list__icon" aria-hidden>
                <OrderIcon />
              </span>
              <span className="menu-list__label">我的订单</span>
              {orderStats.pendingPay > 0 && (
                <span className="menu-list__badge">{orderStats.pendingPay}</span>
              )}
              <span className="menu-list__arrow" aria-hidden>›</span>
            </Link>
          </li>
          <li>
            <Link to="/app/history" className="menu-list__item">
              <span className="menu-list__icon" aria-hidden>
                <HistoryIcon />
              </span>
              <span className="menu-list__label">历史竞拍</span>
              <span className="menu-list__arrow" aria-hidden>›</span>
            </Link>
          </li>
          <li>
            <Link to="/app" className="menu-list__item">
              <span className="menu-list__icon" aria-hidden>
                <LiveIcon />
              </span>
              <span className="menu-list__label">直播竞拍大厅</span>
              <span className="menu-list__arrow" aria-hidden>›</span>
            </Link>
          </li>
          {isAnchorOrAdmin() && (
            <li>
              <Link to="/admin" className="menu-list__item menu-list__item--accent">
                <span className="menu-list__icon" aria-hidden>
                  <SettingsIcon />
                </span>
                <span className="menu-list__label">商家管理后台</span>
                <span className="menu-list__arrow" aria-hidden>›</span>
              </Link>
            </li>
          )}
        </ul>
      </section>
    </div>
  )
}
