import type { SessionStatus } from '../api/types'

/** 根据场次状态返回用户端最合适的入口路径 */
export function auctionEntryPath(
  session: { id: number; roomId: string; status: SessionStatus },
): string {
  switch (session.status) {
    case 'running':
    case 'pending':
      return `/app/live/${session.roomId}?session=${session.id}`
    case 'settled':
      return `/app/result/${session.id}`
    default:
      return `/app/auction/${session.id}`
  }
}

export function auctionEntryCta(status: SessionStatus): string {
  switch (status) {
    case 'running':
      return '进入直播间 →'
    case 'pending':
      return '预约围观 →'
    case 'settled':
      return '查看成交 →'
    default:
      return '查看详情 →'
  }
}
