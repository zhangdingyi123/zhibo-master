import { NavLink, Outlet, useLocation } from 'react-router-dom'
import {
  AuctionIcon,
  HistoryIcon,
  OrderIcon,
  ProfileIcon,
} from '../../components/icons/NavIcons'
import { usePendingPayCount } from '../../hooks/usePendingPayCount'
import { useUnreadMessageCount } from '../../hooks/useUnreadMessageCount'

const IMMERSIVE_PATTERN = /^\/app\/(live|result)\//

const TABS = [
  { to: '/app', end: true, label: '竞拍', Icon: AuctionIcon },
  { to: '/app/history', label: '历史', Icon: HistoryIcon },
  { to: '/app/orders', label: '订单', Icon: OrderIcon },
  { to: '/app/profile', label: '我的', Icon: ProfileIcon },
] as const

export function MobileLayout() {
  const { pathname } = useLocation()
  const immersive = IMMERSIVE_PATTERN.test(pathname)
  const { count: pendingPayCount } = usePendingPayCount()
  const { count: unreadMessages } = useUnreadMessageCount()

  return (
    <div className={`mobile-shell${immersive ? ' mobile-shell--immersive' : ''}`}>
      <main className="mobile-main">
        <Outlet />
      </main>
      {!immersive && (
        <nav className="mobile-tabbar" aria-label="主导航">
          {TABS.map((tab) => {
            const Icon = tab.Icon
            return (
              <NavLink
                key={tab.to}
                to={tab.to}
                end={'end' in tab ? tab.end : false}
                className={({ isActive }) =>
                  `mobile-tab${isActive ? ' active' : ''}`
                }
              >
                <span className="mobile-tab__icon-wrap" aria-hidden>
                  <Icon className="mobile-tab__icon" />
                  {tab.to === '/app/orders' && pendingPayCount > 0 && (
                    <span className="mobile-tab__badge" aria-label={`${pendingPayCount} 笔待支付`}>
                      {pendingPayCount > 9 ? '9+' : pendingPayCount}
                    </span>
                  )}
                  {tab.to === '/app/profile' && unreadMessages > 0 && (
                    <span className="mobile-tab__badge mobile-tab__badge--msg" aria-label={`${unreadMessages} 条未读消息`}>
                      {unreadMessages > 9 ? '9+' : unreadMessages}
                    </span>
                  )}
                </span>
                <span className="mobile-tab__label">{tab.label}</span>
              </NavLink>
            )
          })}
        </nav>
      )}
    </div>
  )
}
