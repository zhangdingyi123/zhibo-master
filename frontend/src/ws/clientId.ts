const CLIENT_ID_KEY = 'zhibo_ws_client_id'
const LAST_SEQ_PREFIX = 'zhibo_ws_last_seq_'

function randomId(): string {
  if (typeof crypto !== 'undefined' && crypto.randomUUID) {
    return `c_${crypto.randomUUID().replace(/-/g, '').slice(0, 16)}`
  }
  return `c_${Date.now().toString(36)}_${Math.random().toString(36).slice(2, 10)}`
}

/** 稳定 clientId，用于心跳与重连（localStorage） */
export function getOrCreateClientId(): string {
  let id = localStorage.getItem(CLIENT_ID_KEY)
  if (!id) {
    id = randomId()
    localStorage.setItem(CLIENT_ID_KEY, id)
  }
  return id
}

export function getStoredLastSeq(roomId: string): number {
  const raw = localStorage.getItem(`${LAST_SEQ_PREFIX}${roomId}`)
  if (!raw) return 0
  const n = Number.parseInt(raw, 10)
  return Number.isFinite(n) && n >= 0 ? n : 0
}

export function setStoredLastSeq(roomId: string, seq: number): void {
  localStorage.setItem(`${LAST_SEQ_PREFIX}${roomId}`, String(seq))
}
