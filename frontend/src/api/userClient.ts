import { applyAuthHeaders } from './authHeaders'
import { ApiError } from './client'

interface ApiResponse<T> {
  code: number
  message: string
  data?: T
}

/** 用户端 REST：使用买家 Mock 鉴权 */
export async function userApiRequest<T>(
  path: string,
  options: RequestInit = {},
): Promise<T> {
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
