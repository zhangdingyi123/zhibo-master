/** 种子数据 openId → userId，用于排行榜高亮「我」 */
const OPEN_ID_USER: Record<string, number> = {
  buyer_001: 2,
  buyer_002: 3,
  buyer_003: 4,
  anchor_001: 1,
  admin_001: 5,
}

export function userIdFromOpenId(openId: string | null | undefined): number | null {
  if (!openId) return null
  return OPEN_ID_USER[openId] ?? null
}
