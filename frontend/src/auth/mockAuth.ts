const STORAGE_KEY = 'zhibo_mock_open_id'

/** 当前 Mock 登录的 openId；空表示围观（未登录） */
export function getMockOpenId(): string | null {
  return localStorage.getItem(STORAGE_KEY)
}

export function setMockOpenId(openId: string | null): void {
  if (openId) {
    localStorage.setItem(STORAGE_KEY, openId)
  } else {
    localStorage.removeItem(STORAGE_KEY)
  }
}

export function isLoggedIn(): boolean {
  return Boolean(getMockOpenId()?.trim())
}
