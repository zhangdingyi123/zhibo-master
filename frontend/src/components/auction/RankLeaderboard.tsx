import { useEffect, useRef } from 'react'
import type { RankEntry } from '../../ws/types'
import { formatCents } from '../../utils/money'

type Props = {
  items: RankEntry[]
  currentUserId: number | null
  participantCount: number
}

export function RankLeaderboard({
  items,
  currentUserId,
  participantCount,
}: Props) {
  const prevTopRef = useRef<number | null>(null)

  useEffect(() => {
    if (items.length > 0) {
      prevTopRef.current = items[0].userId
    }
  }, [items])

  const myRank = currentUserId
    ? items.find((r) => r.userId === currentUserId)
    : undefined

  return (
    <section className="rank-panel">
      <header className="rank-panel__head">
        <h2>实时排名</h2>
        <span className="rank-panel__count">{participantCount} 人参与</span>
      </header>

      {myRank && (
        <div className="rank-me">
          <span>我的排名</span>
          <strong>
            第 {myRank.rank} 名 · {formatCents(myRank.amount)}
          </strong>
        </div>
      )}

      {items.length === 0 ? (
        <p className="rank-empty">暂无出价，抢先手！</p>
      ) : (
        <ul className="rank-list">
          {items.map((row) => {
            const isMe = currentUserId != null && row.userId === currentUserId
            const isTop = row.rank === 1
            return (
              <li
                key={row.userId}
                className={[
                  'rank-item',
                  isMe ? 'rank-item--me' : '',
                  isTop ? 'rank-item--top' : '',
                ]
                  .filter(Boolean)
                  .join(' ')}
              >
                <span className={`rank-badge rank-badge--${row.rank}`}>
                  {row.rank}
                </span>
                <img
                  className="rank-avatar"
                  src={row.avatar || 'https://picsum.photos/seed/default/80'}
                  alt=""
                  width={40}
                  height={40}
                />
                <div className="rank-info">
                  <span className="rank-name">
                    {row.nickname}
                    {isMe && <em>我</em>}
                    {isTop && <em className="rank-crown">领先</em>}
                  </span>
                </div>
                <span className="rank-amount">{formatCents(row.amount)}</span>
              </li>
            )
          })}
        </ul>
      )}
    </section>
  )
}
