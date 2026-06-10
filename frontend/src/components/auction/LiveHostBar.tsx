import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { isLoggedIn } from '../../auth/session'
import { toggleFollow, type AnchorBrief } from '../../api/social'

type Props = {
  anchor?: AnchorBrief | null
  liveTitle?: string
  isFollowing?: boolean
  loginReturnTo: string
  onFollowChange?: (following: boolean) => void
}

export function LiveHostBar({
  anchor,
  liveTitle,
  isFollowing = false,
  loginReturnTo,
  onFollowChange,
}: Props) {
  const [following, setFollowing] = useState(isFollowing)
  const [busy, setBusy] = useState(false)

  useEffect(() => {
    setFollowing(isFollowing)
  }, [isFollowing])

  if (!anchor) return null

  async function handleFollow() {
    if (!isLoggedIn()) return
    setBusy(true)
    try {
      const res = await toggleFollow(anchor!.id)
      setFollowing(res.following)
      onFollowChange?.(res.following)
    } finally {
      setBusy(false)
    }
  }

  const initial = anchor.nickname.slice(0, 1) || '主'

  return (
    <div className="live-host-bar">
      <div className="live-host-bar__avatar" aria-hidden>
        {anchor.avatar ? (
          <img src={anchor.avatar} alt="" />
        ) : (
          <span>{initial}</span>
        )}
      </div>
      <div className="live-host-bar__meta">
        <strong className="live-host-bar__name">{anchor.nickname}</strong>
        <span className="live-host-bar__sub">
          {liveTitle ?? '直播中'}
          {anchor.followerCount > 0 && (
            <> · {anchor.followerCount.toLocaleString('zh-CN')} 粉丝</>
          )}
        </span>
      </div>
      {isLoggedIn() ? (
        <button
          type="button"
          className={`live-host-bar__follow${following ? ' live-host-bar__follow--on' : ''}`}
          disabled={busy}
          onClick={() => void handleFollow()}
        >
          {following ? '已关注' : '+ 关注'}
        </button>
      ) : (
        <Link
          to="/app/login"
          state={{ from: loginReturnTo }}
          className="live-host-bar__follow"
        >
          + 关注
        </Link>
      )}
    </div>
  )
}
