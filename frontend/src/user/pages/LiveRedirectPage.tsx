import { useEffect, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { listAuctions } from '../../api/user'
import { auctionEntryPath } from '../../utils/auctionNav'

const FALLBACK_ROOM = 'room_sess_1'

/** /live 入口：优先跳转第一个进行中场次，否则回首页 */
export function LiveRedirectPage() {
  const navigate = useNavigate()
  const [failed, setFailed] = useState(false)

  useEffect(() => {
    let cancelled = false
    listAuctions({ status: 'running', page: 1, pageSize: 1 })
      .then((res) => {
        if (cancelled) return
        const first = res.items[0]
        if (first) {
          navigate(auctionEntryPath(first.session), { replace: true })
          return
        }
        return listAuctions({ status: 'pending', page: 1, pageSize: 1 }).then((pending) => {
          if (cancelled) return
          const p = pending.items[0]
          if (p) {
            navigate(auctionEntryPath(p.session), { replace: true })
          } else {
            navigate(`/app/live/${FALLBACK_ROOM}`, { replace: true })
          }
        })
      })
      .catch(() => {
        if (!cancelled) setFailed(true)
      })
    return () => {
      cancelled = true
    }
  }, [navigate])

  if (failed) {
    return (
      <div className="user-page user-page--center">
        <p className="user-error">无法加载直播间</p>
        <Link to="/app" className="btn-primary">
          返回竞拍列表
        </Link>
      </div>
    )
  }

  return (
    <div className="user-page user-page--center">
      <p className="user-hint">正在进入直播间…</p>
    </div>
  )
}
