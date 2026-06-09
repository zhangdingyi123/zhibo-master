import { useCallback, useState } from 'react'

type Props = {
  value: string
  label?: string
  className?: string
}

export function CopyButton({ value, label = '复制', className = 'btn-ghost btn-sm' }: Props) {
  const [copied, setCopied] = useState(false)

  const handleCopy = useCallback(async () => {
    const text = value.trim()
    if (!text) return
    try {
      await navigator.clipboard.writeText(text)
    } catch {
      const ta = document.createElement('textarea')
      ta.value = text
      ta.style.position = 'fixed'
      ta.style.opacity = '0'
      document.body.appendChild(ta)
      ta.select()
      document.execCommand('copy')
      document.body.removeChild(ta)
    }
    setCopied(true)
    window.setTimeout(() => setCopied(false), 2000)
  }, [value])

  return (
    <button
      type="button"
      className={className}
      onClick={() => void handleCopy()}
      aria-label={copied ? '已复制' : label}
    >
      {copied ? '已复制' : label}
    </button>
  )
}
