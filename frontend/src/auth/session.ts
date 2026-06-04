export type UserRole = 'buyer' | 'anchor' | 'admin'

export interface AuthUser {
  id: number
  openId: string
  phone?: string
  nickname: string
  avatar: string
  role: UserRole
}

const TOKEN_KEY = 'zhibo_token'
const USER_KEY = 'zhibo_user'

export function getToken(): string | null {
  return localStorage.getItem(TOKEN_KEY)
}

export function getUser(): AuthUser | null {
  const raw = localStorage.getItem(USER_KEY)
  if (!raw) return null
  try {
    return JSON.parse(raw) as AuthUser
  } catch {
    return null
  }
}

export function setSession(token: string, user: AuthUser): void {
  localStorage.setItem(TOKEN_KEY, token)
  localStorage.setItem(USER_KEY, JSON.stringify(user))
  // 兼容旧 Mock 头 / WS openId
  localStorage.setItem('zhibo_mock_open_id', user.openId)
  if (user.role === 'anchor' || user.role === 'admin') {
    localStorage.setItem('zhibo_admin_open_id', user.openId)
  }
}

export function clearSession(): void {
  localStorage.removeItem(TOKEN_KEY)
  localStorage.removeItem(USER_KEY)
  localStorage.removeItem('zhibo_mock_open_id')
  localStorage.removeItem('zhibo_admin_open_id')
}

export function isLoggedIn(): boolean {
  return Boolean(getToken()?.trim())
}

export function isAnchorOrAdmin(): boolean {
  const u = getUser()
  return u?.role === 'anchor' || u?.role === 'admin'
}
