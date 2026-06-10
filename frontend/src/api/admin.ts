import { applyAuthHeaders } from './authHeaders'
import { apiRequest, ApiError } from './client'
import type {
  AuctionSession,
  LiveRoom,
  LiveRoomDetail,
  Order,
  Paginated,
  ProductBody,
  ProductStatus,
  ProductView,
  PublishAuctionBody,
  OrderStatus,
} from './types'

export function listProducts(params: {
  page?: number
  pageSize?: number
  status?: ProductStatus
}) {
  const q = new URLSearchParams()
  if (params.page) q.set('page', String(params.page))
  if (params.pageSize) q.set('pageSize', String(params.pageSize))
  if (params.status) q.set('status', params.status)
  const qs = q.toString()
  return apiRequest<Paginated<ProductView>>(
    `/admin/products${qs ? `?${qs}` : ''}`,
  )
}

export function getProduct(id: number) {
  return apiRequest<ProductView>(`/admin/products/${id}`)
}

export interface GenerateProductIntroResult {
  description: string
  source: 'llm' | 'template'
}

export interface UploadImageResult {
  url: string
}

export async function uploadImage(file: File): Promise<UploadImageResult> {
  const form = new FormData()
  form.append('file', file)
  const headers = new Headers()
  applyAuthHeaders(headers)

  const res = await fetch('/api/v1/admin/upload', {
    method: 'POST',
    body: form,
    headers,
  })
  let json: { code: number; message: string; data?: UploadImageResult }
  try {
    json = (await res.json()) as typeof json
  } catch {
    throw new ApiError('网络响应解析失败', -1)
  }
  if (json.code !== 0) {
    throw new ApiError(json.message || '上传失败', json.code)
  }
  return json.data as UploadImageResult
}

export function generateProductIntro(body: { name: string; keywords?: string }) {
  return apiRequest<GenerateProductIntroResult>('/admin/products/ai-intro', {
    method: 'POST',
    body: JSON.stringify(body),
  })
}

export function createProduct(body: ProductBody) {
  return apiRequest<ProductView>('/admin/products', {
    method: 'POST',
    body: JSON.stringify(body),
  })
}

export function updateProduct(id: number, body: ProductBody) {
  return apiRequest<ProductView>(`/admin/products/${id}`, {
    method: 'PUT',
    body: JSON.stringify(body),
  })
}

export function deleteProduct(id: number) {
  return apiRequest<{ deleted: boolean }>(`/admin/products/${id}`, {
    method: 'DELETE',
  })
}

export function publishAuction(productId: number, body: PublishAuctionBody) {
  return apiRequest<AuctionSession>(`/admin/products/${productId}/auctions`, {
    method: 'POST',
    body: JSON.stringify(body),
  })
}

export function getAuction(sessionId: number) {
  return apiRequest<AuctionSession>(`/admin/auctions/${sessionId}`)
}

export function updateAuctionRules(
  sessionId: number,
  body: PublishAuctionBody,
) {
  return apiRequest<AuctionSession>(`/admin/auctions/${sessionId}/rules`, {
    method: 'PUT',
    body: JSON.stringify(body),
  })
}

export function cancelAuction(sessionId: number, reason: string) {
  return apiRequest<AuctionSession>(`/admin/auctions/${sessionId}/cancel`, {
    method: 'POST',
    body: JSON.stringify({ reason }),
  })
}

export function listOrders(params: {
  page?: number
  pageSize?: number
  status?: OrderStatus
}) {
  const q = new URLSearchParams()
  if (params.page) q.set('page', String(params.page))
  if (params.pageSize) q.set('pageSize', String(params.pageSize))
  if (params.status) q.set('status', params.status)
  const qs = q.toString()
  return apiRequest<Paginated<Order>>(`/admin/orders${qs ? `?${qs}` : ''}`)
}

export function getOrder(id: number) {
  return apiRequest<Order>(`/admin/orders/${id}`)
}

export function createLiveRoom(title: string) {
  return apiRequest<LiveRoom>('/admin/live-rooms', {
    method: 'POST',
    body: JSON.stringify({ title }),
  })
}

export function listLiveRooms() {
  return apiRequest<{ items: LiveRoom[] }>('/admin/live-rooms')
}

export function getLiveRoom(id: number) {
  return apiRequest<LiveRoomDetail>(`/admin/live-rooms/${id}`)
}

export function startLiveRoom(id: number) {
  return apiRequest<LiveRoom>(`/admin/live-rooms/${id}/start`, { method: 'POST' })
}

export function endLiveRoom(id: number) {
  return apiRequest<LiveRoom>(`/admin/live-rooms/${id}/end`, { method: 'POST' })
}

export function addSessionToLiveRoom(
  liveRoomId: number,
  body: PublishAuctionBody & { productId: number },
) {
  return apiRequest<AuctionSession>(`/admin/live-rooms/${liveRoomId}/sessions`, {
    method: 'POST',
    body: JSON.stringify(body),
  })
}

export function addSessionsBatchToLiveRoom(
  liveRoomId: number,
  body: PublishAuctionBody & { productIds: number[] },
) {
  return apiRequest<{ items: AuctionSession[]; count: number }>(
    `/admin/live-rooms/${liveRoomId}/sessions/batch`,
    { method: 'POST', body: JSON.stringify(body) },
  )
}

export function listRoomCommentsAdmin(roomId: string) {
  return apiRequest<{ items: import('./social').RoomComment[] }>(
    `/admin/rooms/${encodeURIComponent(roomId)}/comments`,
  )
}

export function hideRoomComment(commentId: number) {
  return apiRequest<{ ok: boolean }>(`/admin/comments/${commentId}/hide`, {
    method: 'POST',
  })
}

export function endCurrentAndSwitch(liveRoomId: number) {
  return apiRequest<LiveRoomDetail>(`/admin/live-rooms/${liveRoomId}/end-current`, {
    method: 'POST',
  })
}

export function shipOrder(id: number, trackingNo?: string) {
  return apiRequest<Order>(`/admin/orders/${id}/ship`, {
    method: 'POST',
    body: JSON.stringify({ trackingNo: trackingNo ?? '' }),
  })
}

export function cancelOrder(id: number, reason: string) {
  return apiRequest<Order>(`/admin/orders/${id}/cancel`, {
    method: 'POST',
    body: JSON.stringify({ reason }),
  })
}

export function refundOrder(id: number, reason: string) {
  return apiRequest<Order>(`/admin/orders/${id}/refund`, {
    method: 'POST',
    body: JSON.stringify({ reason }),
  })
}
