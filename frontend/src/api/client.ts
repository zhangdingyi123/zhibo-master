import { applyAuthHeaders } from './authHeaders'

export class ApiError extends Error {
  readonly code: number

  constructor(message: string, code: number) {
    super(message)
    this.name = 'ApiError'
    this.code = code
  }
}

interface ApiResponse<T> {
  code: number
  message: string
  data?: T
}

export async function apiRequest<T>(
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
