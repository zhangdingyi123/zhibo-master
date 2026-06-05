import { useCallback, useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import {
  listMessages,
  markAllMessagesRead,
  markMessageRead,
  type MessageCategory,
  type UserMessage,
} from '../../api/messages'
import {
  IsoBellIcon,
  IsoInboxIcon,
  messageEventIcon,
} from '../../components/icons/IsometricIcons'
import { isLoggedIn } from '../../auth/session'
import { useUnreadMessageCount } from '../../hooks/useUnreadMessageCount'

type FilterKey = 'all' | 'unread' | MessageCategory

const FILTERS: { key: FilterKey; label: string }[] = [
  { key: 'all', label: '全部' },
  { key: 'unread', label: '未读' },
  { key: 'auction', label: '竞拍' },
]

function formatTime(iso: string) {
  const d = new Date(iso)
  const now = new Date()
  const diff = now.getTime() - d.getTime()
  if (diff < 60_000) return '刚刚'
  if (diff < 3_600_000) return `${Math.floor(diff / 60_000)} 分钟前`
  if (diff < 86_400_000) return `${Math.floor(diff / 3_600_000)} 小时前`
  return d.toLocaleDateString('zh-CN', { month: 'short', day: 'numeric' })
}

function bentoSpan(msg: UserMessage, index: number) {
  if (!msg.isRead && index === 0) return 'bento-card--hero'
  if (!msg.isRead) return 'bento-card--wide'
  if (msg.eventType === 'settled_win') return 'bento-card--tall'
  return ''
}

export function MessagesPage() {
  const loggedIn = isLoggedIn()
  const { count: unreadBadge, refresh: refreshBadge } = useUnreadMessageCount()
  const [filter, setFilter] = useState<FilterKey>('all')
  const [messages, setMessages] = useState<UserMessage[]>([])
  const [stats, setStats] = useState({ total: 0, unread: 0 })
  const [loading, setLoading] = useState(true)
  const [markingAll, setMarkingAll] = useState(false)

  const load = useCallback(async () => {
    if (!loggedIn) return
    setLoading(true)
    try {
      const res = await listMessages({
        page: 1,
        pageSize: 50,
        unread: filter === 'unread',
      })
      let items = res.items
      if (filter === 'auction') {
        items = items.filter((m) => m.category === 'auction')
      }
      setMessages(items)
      setStats({ total: res.total, unread: res.unread })
    } catch {
      setMessages([])
    } finally {
      setLoading(false)
    }
  }, [loggedIn, filter])

  useEffect(() => {
    void load()
  }, [load])

  const categoryCounts = useMemo(() => {
    const auction = messages.filter((m) => m.category === 'auction').length
    return { auction }
  }, [messages])

  async function handleRead(msg: UserMessage) {
    if (msg.isRead) return
    try {
      await markMessageRead(msg.id)
      setMessages((prev) =>
        prev.map((m) => (m.id === msg.id ? { ...m, isRead: true } : m)),
      )
      setStats((s) => ({ ...s, unread: Math.max(0, s.unread - 1) }))
      refreshBadge()
    } catch {
      /* 忽略 */
    }
  }

  async function handleReadAll() {
    setMarkingAll(true)
    try {
      await markAllMessagesRead()
      setMessages((prev) => prev.map((m) => ({ ...m, isRead: true })))
      setStats((s) => ({ ...s, unread: 0 }))
      refreshBadge()
    } catch {
      /* 忽略 */
    } finally {
      setMarkingAll(false)
    }
  }

  if (!loggedIn) {
    return (
      <div className="user-page user-page--messages">
        <div className="glass-panel glass-panel--center">
          <IsoInboxIcon className="iso-icon iso-icon--lg" />
          <h2>消息中心</h2>
          <p className="muted">登录后查看竞拍通知与系统消息</p>
          <Link to="/app/login" className="btn-primary btn-block">
            去登录
          </Link>
        </div>
      </div>
    )
  }

  return (
    <div className="user-page user-page--messages">
      <div className="messages-ambient" aria-hidden />

      <header className="messages-hero glass-panel">
        <div className="messages-hero__icon-wrap">
          <IsoBellIcon className="iso-icon iso-icon--hero" />
        </div>
        <div className="messages-hero__text">
          <h1>消息中心</h1>
          <p>写扩散收件箱 · 竞拍动态实时落库</p>
        </div>
        <div className="messages-hero__stats">
          <div className="messages-stat-pill">
            <strong>{stats.unread}</strong>
            <span>未读</span>
          </div>
          <div className="messages-stat-pill">
            <strong>{stats.total}</strong>
            <span>全部</span>
          </div>
        </div>
      </header>

      <div className="messages-toolbar">
        <div className="messages-filters" role="tablist" aria-label="消息筛选">
          {FILTERS.map((f) => (
            <button
              key={f.key}
              type="button"
              role="tab"
              aria-selected={filter === f.key}
              className={`messages-filter${filter === f.key ? ' active' : ''}`}
              onClick={() => setFilter(f.key)}
            >
              {f.label}
              {f.key === 'unread' && unreadBadge > 0 && (
                <span className="messages-filter__badge">{unreadBadge}</span>
              )}
              {f.key === 'auction' && categoryCounts.auction > 0 && (
                <span className="messages-filter__badge">{categoryCounts.auction}</span>
              )}
            </button>
          ))}
        </div>
        {stats.unread > 0 && (
          <button
            type="button"
            className="btn-ghost btn-sm"
            disabled={markingAll}
            onClick={() => void handleReadAll()}
          >
            {markingAll ? '处理中…' : '全部已读'}
          </button>
        )}
      </div>

      {loading ? (
        <div className="messages-empty glass-panel">
          <p className="muted">加载中…</p>
        </div>
      ) : messages.length === 0 ? (
        <div className="messages-empty glass-panel">
          <IsoInboxIcon className="iso-icon iso-icon--lg" />
          <p>暂无消息</p>
          <span className="muted">参与竞拍后，通知会写入您的收件箱</span>
          <Link to="/app" className="btn-secondary">
            去竞拍大厅
          </Link>
        </div>
      ) : (
        <div className="bento-grid bento-grid--messages">
          {messages.map((msg, i) => {
            const Icon = messageEventIcon(msg.eventType)
            const sessionId = msg.payload?.sessionId as number | undefined
            const span = bentoSpan(msg, i)
            return (
              <article
                key={msg.id}
                className={`bento-card glass-panel${msg.isRead ? '' : ' bento-card--unread'}${span ? ` ${span}` : ''}`}
                onClick={() => void handleRead(msg)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter') void handleRead(msg)
                }}
                role="button"
                tabIndex={0}
              >
                <div className="bento-card__icon">
                  <Icon className="iso-icon" />
                </div>
                <div className="bento-card__body">
                  <div className="bento-card__head">
                    <h3>{msg.title}</h3>
                    {!msg.isRead && <span className="bento-card__dot" aria-label="未读" />}
                  </div>
                  <p>{msg.body}</p>
                  <footer className="bento-card__foot">
                    <time dateTime={msg.createdAt}>{formatTime(msg.createdAt)}</time>
                    {sessionId != null && (
                      <Link
                        to={`/app/auction/${sessionId}`}
                        className="bento-card__link"
                        onClick={(e) => e.stopPropagation()}
                      >
                        查看场次
                      </Link>
                    )}
                  </footer>
                </div>
              </article>
            )
          })}
        </div>
      )}
    </div>
  )
}
