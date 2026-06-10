import { useCallback, useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { isLoggedIn } from '../../auth/session'
import {
  likeRoom,
  postRoomComment,
  toggleFavorite,
} from '../../api/social'
import { HeartIcon } from '../icons/NavIcons'

type Props = {
  roomId: string
  productId?: number
  likeCount: number
  isFavorited?: boolean
  loginReturnTo: string
  onLikeCount?: (n: number) => void
  onCommentSent?: (text: string) => void
  onFavoriteChange?: (favorited: boolean) => void
}

export function LiveInteractionDock({
  roomId,
  productId,
  likeCount,
  isFavorited = false,
  loginReturnTo,
  onLikeCount,
  onCommentSent,
  onFavoriteChange,
}: Props) {
  const [text, setText] = useState('')
  const [likes, setLikes] = useState(likeCount)
  const [favorited, setFavorited] = useState(isFavorited)

  useEffect(() => {
    setLikes(likeCount)
  }, [likeCount])

  useEffect(() => {
    setFavorited(isFavorited)
  }, [isFavorited])
  const [sending, setSending] = useState(false)
  const [pulse, setPulse] = useState(false)

  const handleLike = useCallback(async () => {
    if (!isLoggedIn()) return
    setPulse(true)
    window.setTimeout(() => setPulse(false), 200)
    try {
      const res = await likeRoom(roomId)
      setLikes(res.likeCount)
      onLikeCount?.(res.likeCount)
    } catch {
      /* ignore */
    }
  }, [roomId, onLikeCount])

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    const content = text.trim()
    if (!content || !isLoggedIn()) return
    setSending(true)
    try {
      const c = await postRoomComment(roomId, content)
      setText('')
      onCommentSent?.(c.content)
    } finally {
      setSending(false)
    }
  }

  async function handleFavorite() {
    if (!productId || !isLoggedIn()) return
    try {
      const res = await toggleFavorite(productId)
      setFavorited(res.favorited)
      onFavoriteChange?.(res.favorited)
    } catch {
      /* ignore */
    }
  }

  const loggedIn = isLoggedIn()

  return (
    <div className="live-interaction-dock">
      <form className="live-interaction-dock__comment" onSubmit={handleSubmit}>
        {loggedIn ? (
          <>
            <input
              value={text}
              onChange={(e) => setText(e.target.value)}
              placeholder="说点什么…"
              maxLength={200}
              disabled={sending}
              aria-label="发送评论"
            />
            <button type="submit" className="btn-primary btn-sm" disabled={sending || !text.trim()}>
              发送
            </button>
          </>
        ) : (
          <Link
            to="/app/login"
            state={{ from: loginReturnTo }}
            className="live-interaction-dock__login-hint"
          >
            登录后参与评论互动
          </Link>
        )}
      </form>
      <div className="live-interaction-dock__actions">
        <button
          type="button"
          className={`live-interaction-dock__action${pulse ? ' live-interaction-dock__action--pulse' : ''}`}
          onClick={() => void handleLike()}
          aria-label="点赞"
          title="点赞"
        >
          <HeartIcon />
          <span>{likes > 0 ? likes.toLocaleString('zh-CN') : '赞'}</span>
        </button>
        {productId != null && (
          <button
            type="button"
            className={`live-interaction-dock__action live-interaction-dock__action--fav${favorited ? ' live-interaction-dock__action--fav-on' : ''}`}
            onClick={() => void handleFavorite()}
            aria-label={favorited ? '取消收藏' : '收藏商品'}
            title={favorited ? '已收藏' : '收藏'}
          >
            <span className="live-interaction-dock__fav-icon" aria-hidden>
              {favorited ? '★' : '☆'}
            </span>
            <span>{favorited ? '已藏' : '收藏'}</span>
          </button>
        )}
      </div>
    </div>
  )
}
