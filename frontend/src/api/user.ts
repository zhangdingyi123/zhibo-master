import type {
  AuctionRules,
  AuctionSession,
  LiveRoom,
  OrderListItem,
  Paginated,
  Product,
  SessionStatus,
  SessionSummary,
  ShippingAddressBody,
} from './types'
import { userApiRequest } from './userClient'

export interface ProductBrief {
  id: number
  name: string
  description: string
  coverUrl: string
}

export interface UserAuctionListItem {
  session: AuctionSession
  product: ProductBrief
}

export interface SessionSnapshotDto {
  sessionId: number
  roomId: string
  status: SessionStatus
  currentPrice: number
  bidCount: number
  participantCount: number
  minNextBid: number
  rules: AuctionRules
  endAtMs?: number
  remainingMs: number
  serverTimeMs: number
  winnerId?: number
}

export interface UserAuctionDetail {
  session: AuctionSession
  product: Product
  snapshot: SessionSnapshotDto
}

export function listAuctions(params?: {
  status?: SessionStatus
  page?: number
  pageSize?: number
}) {
  const q = new URLSearchParams()
  if (params?.status) q.set('status', params.status)
  if (params?.page) q.set('page', String(params.page))
  if (params?.pageSize) q.set('pageSize', String(params.pageSize))
  const qs = q.toString()
  return userApiRequest<{
    items: UserAuctionListItem[]
    total: number
    page: number
    pageSize: number
  }>(`/auctions${qs ? `?${qs}` : ''}`)
}

export function getAuction(sessionId: number) {
  return userApiRequest<UserAuctionDetail>(`/auctions/${sessionId}`)
}

export interface AnchorBrief {
  id: number
  nickname: string
  avatar: string
  followerCount: number
}

export interface UserLiveRoomDetail {
  roomId: string
  liveRoom: LiveRoom
  anchor?: AnchorBrief
  current?: UserAuctionDetail
  history: SessionSummary[]
}

export function getLiveRoom(roomId: string) {
  return userApiRequest<UserLiveRoomDetail>(`/rooms/${encodeURIComponent(roomId)}`)
}

export function listMyOrders(params?: {
  status?: string
  page?: number
  pageSize?: number
}) {
  const q = new URLSearchParams()
  if (params?.status) q.set('status', params.status)
  if (params?.page) q.set('page', String(params.page))
  if (params?.pageSize) q.set('pageSize', String(params.pageSize))
  const qs = q.toString()
  return userApiRequest<Paginated<OrderListItem>>(`/orders${qs ? `?${qs}` : ''}`)
}

export function getOrder(orderId: number) {
  return userApiRequest<OrderListItem>(`/orders/${orderId}`)
}

export function getOrderBySession(sessionId: number) {
  return userApiRequest<OrderListItem>(`/auctions/${sessionId}/order`)
}

export function mockPayOrder(orderId: number) {
  return userApiRequest<OrderListItem>(`/orders/${orderId}/mock-pay`, { method: 'POST' })
}

export function submitShippingAddress(orderId: number, body: ShippingAddressBody) {
  return userApiRequest<OrderListItem>(`/orders/${orderId}/shipping-address`, {
    method: 'PUT',
    body: JSON.stringify(body),
  })
}

export function confirmReceive(orderId: number) {
  return userApiRequest<OrderListItem>(`/orders/${orderId}/confirm-receive`, {
    method: 'POST',
  })
}

export function cancelOrder(orderId: number, reason: string) {
  return userApiRequest<OrderListItem>(`/orders/${orderId}/cancel`, {
    method: 'POST',
    body: JSON.stringify({ reason }),
  })
}
