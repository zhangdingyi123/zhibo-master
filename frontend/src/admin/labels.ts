import type { OrderStatus, ProductStatus, SessionStatus } from '../api/types'

export const PRODUCT_STATUS_LABEL: Record<ProductStatus, string> = {
  draft: '草稿',
  listed: '已上架',
  auctioning: '竞拍中',
  sold: '已售出',
  off_shelf: '已下架',
}

export const SESSION_STATUS_LABEL: Record<SessionStatus, string> = {
  pending: '未开始',
  running: '进行中',
  settled: '已成交',
  cancelled: '已取消',
  failed: '异常',
}

export const ORDER_STATUS_LABEL: Record<OrderStatus, string> = {
  pending_pay: '待支付',
  paid: '已支付',
  cancelled: '已取消',
  closed: '已关闭',
}
