/** 将剩余毫秒格式化为 m:ss */
export function formatRemainingMs(ms: number): string {
  const sec = Math.max(0, Math.ceil(ms / 1000))
  const m = Math.floor(sec / 60)
  const s = sec % 60
  return `${m}:${s.toString().padStart(2, '0')}`
}

/** 根据 endAt ISO 字符串计算剩余毫秒（相对当前时间） */
export function remainingMsFromEndAt(endAt: string | undefined): number | null {
  if (!endAt) return null
  const end = new Date(endAt).getTime()
  if (Number.isNaN(end)) return null
  return Math.max(0, end - Date.now())
}
