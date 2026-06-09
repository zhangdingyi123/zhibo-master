import { userApiRequest } from './userClient'

export type MessageEventType =
  | 'outbid'
  | 'extended'
  | 'settled_win'
  | 'settled'
  | 'cancelled'
  | 'order_shipped'
  | 'order_cancelled'
  | 'order_refunded'

export type MessageCategory = 'auction' | 'order' | 'system'

export interface UserMessage {
  id: number
  userId: number
  eventType: MessageEventType
  category: MessageCategory
  title: string
  body: string
  payload?: Record<string, unknown>
  isRead: boolean
  createdAt: string
}

export interface MessageListResult {
  items: UserMessage[]
  total: number
  unread: number
  page: number
  pageSize: number
}

export function listMessages(params?: {
  page?: number
  pageSize?: number
  unread?: boolean
}) {
  const q = new URLSearchParams()
  if (params?.page) q.set('page', String(params.page))
  if (params?.pageSize) q.set('pageSize', String(params.pageSize))
  if (params?.unread) q.set('unread', '1')
  const qs = q.toString()
  return userApiRequest<MessageListResult>(`/messages${qs ? `?${qs}` : ''}`)
}

export function getUnreadMessageCount() {
  return userApiRequest<{ count: number }>('/messages/unread-count')
}

export function markMessageRead(id: number) {
  return userApiRequest<{ ok: boolean }>(`/messages/${id}/read`, { method: 'POST' })
}

export function markAllMessagesRead() {
  return userApiRequest<{ updated: number }>('/messages/read-all', { method: 'POST' })
}
