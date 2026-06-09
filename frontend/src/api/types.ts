export type ProductStatus =
  | 'draft'
  | 'listed'
  | 'auctioning'
  | 'sold'
  | 'off_shelf'

export type SessionStatus =
  | 'pending'
  | 'running'
  | 'settled'
  | 'cancelled'
  | 'failed'

export type OrderStatus =
  | 'pending_pay'
  | 'paid'
  | 'shipped'
  | 'completed'
  | 'cancelled'
  | 'closed'
  | 'refunded'

export type OrderActor = 'buyer' | 'seller' | 'system'

export interface AuctionRules {
  startingPrice: number
  bidIncrement: number
  capPrice?: number | null
  durationSec: number
  extendThresholdSec: number
  extendSec: number
}

export interface Product {
  id: number
  anchorId: number
  name: string
  description: string
  coverUrl: string
  images?: string[]
  status: ProductStatus
  createdAt: string
  updatedAt: string
}

export interface AuctionProgress {
  sessionId: number
  roomId: string
  status: SessionStatus
  currentPrice: number
  bidCount: number
  participantCount: number
  scheduledStartAt?: string
  startedAt?: string
  endAt?: string
  settledAt?: string
  winnerId?: number
  cancelReason?: string
  order?: Order
}

export interface ProductView extends Product {
  auction?: AuctionProgress
}

export type LiveRoomStatus = 'idle' | 'live' | 'ended'

export interface LiveRoom {
  id: number
  anchorId: number
  title: string
  roomId: string
  status: LiveRoomStatus
  currentSessionId?: number
  createdAt: string
  updatedAt: string
}

export interface LiveRoomSessionItem {
  session: AuctionSession
  product: {
    id: number
    name: string
    description: string
    coverUrl: string
  }
}

export interface LiveRoomFunnel {
  viewerCount: number
  bidderCount: number
  settledCount: number
  paidCount: number
  hint?: string
}

export interface LiveRoomDetail {
  liveRoom: LiveRoom
  currentSession?: LiveRoomSessionItem
  queue: LiveRoomSessionItem[]
  history: LiveRoomSessionItem[]
  funnel: LiveRoomFunnel
}

export interface SessionSummary {
  sessionId: number
  productId: number
  productName: string
  coverUrl: string
  status: SessionStatus
  finalPrice: number
  winnerId?: number
  seqInRoom: number
}

export interface AuctionSession {
  id: number
  productId: number
  anchorId: number
  liveRoomId?: number
  seqInRoom?: number
  roomId: string
  status: SessionStatus
  rules: AuctionRules
  currentPrice: number
  bidCount: number
  participantCount: number
  winnerId?: number
  scheduledStartAt?: string
  startedAt?: string
  endAt?: string
  settledAt?: string
  cancelReason?: string
  createdAt: string
  updatedAt: string
}

export interface Order {
  id: number
  orderNo: string
  sessionId: number
  productId: number
  buyerId: number
  sellerId: number
  amount: number
  status: OrderStatus
  payExpireAt?: string
  paidAt?: string
  receiverName?: string
  receiverPhone?: string
  receiverAddress?: string
  trackingNo?: string
  shippedAt?: string
  completedAt?: string
  cancelReason?: string
  cancelledBy?: OrderActor
  cancelledAt?: string
  refundedAt?: string
  createdAt: string
  updatedAt: string
}

export interface AftersaleBody {
  reason: string
}

export interface ShippingAddressBody {
  receiverName: string
  receiverPhone: string
  receiverAddress: string
}

export interface OrderListItem {
  order: Order
  product: {
    id: number
    name: string
    description: string
    coverUrl: string
  }
}

export interface Paginated<T> {
  items: T[]
  total: number
  page: number
  pageSize: number
}

export interface ProductBody {
  name: string
  description: string
  coverUrl: string
  images: string[]
}

export interface PublishAuctionBody {
  startingPrice: number
  bidIncrement: number
  capPrice?: number | null
  durationSec: number
  extendThresholdSec?: number
  extendSec?: number
  scheduledStartAt?: string
}
