/** 金额单位：分 */

export function formatCents(cents: number): string {
  return `¥${(cents / 100).toLocaleString('zh-CN', {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  })}`
}

export function centsToYuanInput(cents: number): string {
  return (cents / 100).toFixed(2)
}

export function yuanInputToCents(yuan: string): number | null {
  const n = Number.parseFloat(yuan.replace(/,/g, ''))
  if (!Number.isFinite(n) || n < 0) return null
  return Math.round(n * 100)
}
