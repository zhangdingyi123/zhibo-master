import { getMockOpenId } from '../auth/mockAuth'
import { getToken } from '../auth/session'

/** 为 API 请求附加鉴权头：优先 JWT，其次开发 Mock */
export function applyAuthHeaders(headers: Headers): void {
  const token = getToken()
  if (token) {
    headers.set('Authorization', `Bearer ${token}`)
    return
  }
  const openId = getMockOpenId()
  if (openId) {
    headers.set('X-Mock-Open-Id', openId)
  }
}
