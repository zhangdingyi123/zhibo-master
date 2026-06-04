import type { AuthUser } from '../auth/session'
import { applyAuthHeaders } from './authHeaders'
import { ApiError } from './client'

interface ApiResponse<T> {
  code: number
  message: string
  data?: T
}

export interface AuthResult {
  token: string
  user: AuthUser
}

async function authRequest<T>(path: string, options: RequestInit = {}): Promise<T> {
  const headers = new Headers(options.headers)
  if (!headers.has('Content-Type') && options.body) {
    headers.set('Content-Type', 'application/json')
  }
  applyAuthHeaders(headers)

  const res = await fetch(`/api/v1${path}`, { ...options, headers })
  let json: ApiResponse<T>
  try {
    json = (await res.json()) as ApiResponse<T>
  } catch {
    throw new ApiError('网络响应解析失败', -1)
  }
  if (json.code !== 0) {
    throw new ApiError(json.message || '请求失败', json.code)
  }
  return json.data as T
}

export function login(phone: string, password: string) {
  return authRequest<AuthResult>('/auth/login', {
    method: 'POST',
    body: JSON.stringify({ phone, password }),
  })
}

export function register(input: {
  phone: string
  password: string
  nickname: string
  role?: 'buyer' | 'anchor'
}) {
  return authRequest<AuthResult>('/auth/register', {
    method: 'POST',
    body: JSON.stringify(input),
  })
}

export function fetchMe() {
  return authRequest<AuthUser>('/auth/me')
}
