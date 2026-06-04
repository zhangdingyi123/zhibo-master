import { getToken, getUser } from './session'

const STORAGE_KEY = 'zhibo_admin_open_id'

export const ADMIN_ACCOUNTS = [
  { openId: 'anchor_001', label: '主播小美', phone: '13800000001' },
  { openId: 'admin_001', label: '运营管理员', phone: '13800000005' },
] as const

export function getAdminOpenId(): string | null {
  return localStorage.getItem(STORAGE_KEY)
}

export function setAdminOpenId(openId: string | null): void {
  if (openId) {
    localStorage.setItem(STORAGE_KEY, openId)
  } else {
    localStorage.removeItem(STORAGE_KEY)
  }
}

export function isAdminLoggedIn(): boolean {
  const token = getToken()
  if (token) {
    const u = getUser()
    return u?.role === 'anchor' || u?.role === 'admin'
  }
  return Boolean(getAdminOpenId()?.trim())
}
