export async function synthesizeSpeech(text: string): Promise<Blob> {
  const res = await fetch('/api/v1/tts', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ text }),
  })
  if (!res.ok) {
    throw new Error(`TTS failed: ${res.status}`)
  }
  return res.blob()
}
