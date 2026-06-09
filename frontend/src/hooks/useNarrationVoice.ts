import { useCallback, useEffect, useRef, useState } from 'react'
import { synthesizeSpeech } from '../api/tts'

const STORAGE_KEY = 'zhibo-narration-voice'

function canUseSpeechSynthesis(): boolean {
  return typeof window !== 'undefined' && 'speechSynthesis' in window
}

export function useNarrationVoice() {
  const [enabled, setEnabled] = useState(() => {
    if (typeof window === 'undefined') return false
    return localStorage.getItem(STORAGE_KEY) === '1'
  })
  const audioRef = useRef<HTMLAudioElement | null>(null)
  const speakingRef = useRef(false)

  const stop = useCallback(() => {
    speakingRef.current = false
    if (audioRef.current) {
      audioRef.current.pause()
      audioRef.current = null
    }
    if (canUseSpeechSynthesis()) {
      window.speechSynthesis.cancel()
    }
  }, [])

  const toggle = useCallback(() => {
    setEnabled((prev) => {
      const next = !prev
      localStorage.setItem(STORAGE_KEY, next ? '1' : '0')
      if (!next) stop()
      return next
    })
  }, [stop])

  const speakWithBrowser = useCallback((text: string) => {
    if (!canUseSpeechSynthesis()) return
    window.speechSynthesis.cancel()
    const utter = new SpeechSynthesisUtterance(text)
    utter.lang = 'zh-CN'
    utter.rate = 1.05
    utter.onend = () => {
      speakingRef.current = false
    }
    utter.onerror = () => {
      speakingRef.current = false
    }
    speakingRef.current = true
    window.speechSynthesis.speak(utter)
  }, [])

  const speak = useCallback(
    async (text: string) => {
      const line = text.trim()
      if (!line || !enabled) return
      stop()

      try {
        const blob = await synthesizeSpeech(line)
        const url = URL.createObjectURL(blob)
        const audio = new Audio(url)
        audioRef.current = audio
        speakingRef.current = true
        audio.onended = () => {
          speakingRef.current = false
          URL.revokeObjectURL(url)
          audioRef.current = null
        }
        audio.onerror = () => {
          speakingRef.current = false
          URL.revokeObjectURL(url)
          audioRef.current = null
          speakWithBrowser(line)
        }
        await audio.play()
      } catch {
        speakWithBrowser(line)
      }
    },
    [enabled, stop, speakWithBrowser],
  )

  useEffect(() => () => stop(), [stop])

  return { narrationEnabled: enabled, toggleNarration: toggle, speak, stop }
}
