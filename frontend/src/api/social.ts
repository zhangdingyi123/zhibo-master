import { userApiRequest } from './userClient'

export interface AnchorBrief {
  id: number
  nickname: string
  avatar: string
  followerCount: number
}

export interface RoomSocialStats {
  likeCount: number
  commentCount: number
  isFollowing?: boolean
  isFavorited?: boolean
}

export interface RoomComment {
  id: number
  roomId: string
  userId: number
  nickname: string
  avatar: string
  content: string
  isHidden?: boolean
  createdAt: string
}

export function getRoomStats(roomId: string, productId?: number) {
  const q = productId ? `?productId=${productId}` : ''
  return userApiRequest<RoomSocialStats>(
    `/rooms/${encodeURIComponent(roomId)}/stats${q}`,
  )
}

export function listRoomComments(roomId: string) {
  return userApiRequest<{ items: RoomComment[] }>(
    `/rooms/${encodeURIComponent(roomId)}/comments`,
  )
}

export function postRoomComment(roomId: string, content: string) {
  return userApiRequest<RoomComment>(
    `/rooms/${encodeURIComponent(roomId)}/comments`,
    { method: 'POST', body: JSON.stringify({ content }) },
  )
}

export function likeRoom(roomId: string) {
  return userApiRequest<{ likeCount: number }>(
    `/rooms/${encodeURIComponent(roomId)}/like`,
    { method: 'POST' },
  )
}

export function toggleFavorite(productId: number) {
  return userApiRequest<{ favorited: boolean }>(
    `/products/${productId}/favorite`,
    { method: 'POST' },
  )
}

export function toggleFollow(anchorId: number) {
  return userApiRequest<{ following: boolean }>(
    `/anchors/${anchorId}/follow`,
    { method: 'POST' },
  )
}
