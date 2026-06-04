import type { ConnectionState } from '../ws/types'

const LABELS: Record<ConnectionState, string> = {
  idle: '未连接',
  connecting: '连接中…',
  connected: '实时',
  reconnecting: '重连中…',
  closed: '已断开',
}

export function connectionLabel(state: string): string {
  return LABELS[state as ConnectionState] ?? state
}
