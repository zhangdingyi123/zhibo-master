import { useEffect, useState } from 'react'
import { getStreamInfo, type StreamInfo } from '../../api/stream'

type Props = {
  roomId: string
}

export function StreamPushInfo({ roomId }: Props) {
  const [info, setInfo] = useState<StreamInfo | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [copied, setCopied] = useState(false)

  useEffect(() => {
    let cancelled = false
    void getStreamInfo(roomId)
      .then((data) => {
        if (!cancelled) setInfo(data)
      })
      .catch((e: unknown) => {
        if (!cancelled) {
          setError(e instanceof Error ? e.message : '加载失败')
        }
      })
    return () => {
      cancelled = true
    }
  }, [roomId])

  async function copyPushUrl() {
    if (!info) return
    try {
      await navigator.clipboard.writeText(info.pushUrl)
      setCopied(true)
      window.setTimeout(() => setCopied(false), 2000)
    } catch {
      setCopied(false)
    }
  }

  if (error) return <p className="form-error">{error}</p>
  if (!info) return <p className="muted">加载推流地址…</p>

  return (
    <div className="stream-push-info">
      <p className="muted stream-push-info__hint">
        用 OBS / ffmpeg 向下方地址推流，用户端将自动播放 HLS。
      </p>
      <dl className="detail-dl">
        <dt>推流地址</dt>
        <dd>
          <code className="stream-push-info__url">{info.pushUrl}</code>
          <button type="button" className="btn-ghost btn-sm" onClick={() => void copyPushUrl()}>
            {copied ? '已复制' : '复制'}
          </button>
        </dd>
        <dt>拉流 (HLS)</dt>
        <dd>
          <code>{info.hlsUrl}</code>
        </dd>
      </dl>
    </div>
  )
}
