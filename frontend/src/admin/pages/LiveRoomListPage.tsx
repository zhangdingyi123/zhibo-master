import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { createLiveRoom, listLiveRooms } from '../../api/admin'
import type { LiveRoom } from '../../api/types'

export function LiveRoomListPage() {
  const [rooms, setRooms] = useState<LiveRoom[]>([])
  const [title, setTitle] = useState('')
  const [loading, setLoading] = useState(true)
  const [creating, setCreating] = useState(false)
  const [error, setError] = useState<string | null>(null)

  function refresh() {
    return listLiveRooms()
      .then((res) => setRooms(res.items))
      .catch((e) => setError(e instanceof Error ? e.message : '加载失败'))
  }

  useEffect(() => {
    refresh().finally(() => setLoading(false))
  }, [])

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault()
    if (!title.trim()) return
    setCreating(true)
    setError(null)
    try {
      const lr = await createLiveRoom(title.trim())
      setTitle('')
      await refresh()
      window.location.href = `/admin/live-rooms/${lr.id}`
    } catch (err) {
      setError(err instanceof Error ? err.message : '创建失败')
    } finally {
      setCreating(false)
    }
  }

  return (
    <div className="admin-page">
      <div className="admin-page__head">
        <div>
          <h2>直播连拍</h2>
          <p className="page-desc">一场直播串联多个 SKU，结束当前品后切换下一品</p>
        </div>
      </div>

      <div className="admin-card">
        <form className="live-room-create" onSubmit={handleCreate}>
          <label>
            直播标题
            <input
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder="例如：618 福利专场"
              required
            />
          </label>
          <button type="submit" className="btn-primary" disabled={creating}>
            {creating ? '创建中…' : '创建直播'}
          </button>
        </form>
        {error && <p className="form-error">{error}</p>}
      </div>

      {loading ? (
        <p className="muted">加载中…</p>
      ) : rooms.length === 0 ? (
        <p className="muted">暂无直播，先创建一个连拍专场吧</p>
      ) : (
        <ul className="live-room-list">
          {rooms.map((lr) => (
            <li key={lr.id}>
              <Link to={`/admin/live-rooms/${lr.id}`} className="live-room-list__item">
                <div>
                  <strong>{lr.title || `直播 #${lr.id}`}</strong>
                  <span className="muted">{lr.roomId}</span>
                </div>
                <span className={`status-pill status-pill--${lr.status}`}>{lr.status}</span>
              </Link>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
