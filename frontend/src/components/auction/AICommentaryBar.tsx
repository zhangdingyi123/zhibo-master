type Props = {
  line: string
  visible: boolean
  voiceOn: boolean
}

export function AICommentaryBar({ line, visible, voiceOn }: Props) {
  if (!visible || !line) return null

  return (
    <div className="ai-commentary" role="status" aria-live="polite">
      <span className="ai-commentary__badge">AI 解说</span>
      <p className="ai-commentary__text">{line}</p>
      {voiceOn && <span className="ai-commentary__voice" aria-hidden>🔊</span>}
    </div>
  )
}
