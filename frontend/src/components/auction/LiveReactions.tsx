import { useCallback, useState } from 'react'
import { HeartIcon } from '../icons/NavIcons'

type Particle = {
  id: number
  x: number
}

export function LiveReactions() {
  const [particles, setParticles] = useState<Particle[]>([])
  const [pulse, setPulse] = useState(false)

  const react = useCallback(() => {
    const id = Date.now() + Math.random()
    const x = 15 + Math.random() * 70
    setParticles((prev) => [...prev.slice(-12), { id, x }])
    setPulse(true)
    window.setTimeout(() => setPulse(false), 200)
    window.setTimeout(() => {
      setParticles((prev) => prev.filter((p) => p.id !== id))
    }, 2200)
  }, [])

  return (
    <div className="live-reactions">
      <div className="live-reactions__float" aria-hidden>
        {particles.map((p) => (
          <span
            key={p.id}
            className="live-reactions__particle"
            style={{ left: `${p.x}%` }}
          >
            <HeartIcon />
          </span>
        ))}
      </div>
      <button
        type="button"
        className={`live-reactions__btn${pulse ? ' live-reactions__btn--pulse' : ''}`}
        onClick={react}
        aria-label="点赞"
        title="点赞"
      >
        <HeartIcon />
      </button>
    </div>
  )
}
