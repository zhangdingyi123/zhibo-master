import { useEffect, useMemo, useState } from 'react'

const INTERVAL_MS = 9000

function splitIntroLines(description: string, productTitle?: string): string[] {
  const raw = description.trim()
  if (!raw) {
    if (productTitle) {
      return [`正在介绍【${productTitle}】，欢迎参与竞拍`]
    }
    return []
  }
  const parts = raw
    .split(/[。！？；\n]+/)
    .map((s) => s.trim())
    .filter(Boolean)
  return parts.length > 0 ? parts : [raw]
}

export function useProductNarration(
  description: string | undefined,
  productTitle?: string,
  narrationEnabled = false,
  speak?: (text: string) => void | Promise<void>,
) {
  const lines = useMemo(
    () => splitIntroLines(description ?? '', productTitle),
    [description, productTitle],
  )
  const [index, setIndex] = useState(0)

  useEffect(() => {
    setIndex(0)
  }, [description, productTitle])

  const currentLine = lines.length > 0 ? lines[index % lines.length]! : ''

  useEffect(() => {
    if (lines.length <= 1) return
    const t = window.setInterval(() => {
      setIndex((i) => (i + 1) % lines.length)
    }, INTERVAL_MS)
    return () => clearInterval(t)
  }, [lines])

  useEffect(() => {
    if (!narrationEnabled || !currentLine || !speak) return
    void speak(currentLine)
  }, [currentLine, narrationEnabled, speak])

  return { currentLine, hasLines: lines.length > 0 }
}
