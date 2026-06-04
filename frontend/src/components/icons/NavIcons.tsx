type IconProps = { className?: string }

const base = { viewBox: '0 0 24 24', fill: 'none', stroke: 'currentColor', strokeWidth: 2, strokeLinecap: 'round' as const, strokeLinejoin: 'round' as const }

export function AuctionIcon({ className }: IconProps) {
  return (
    <svg className={className} {...base} aria-hidden>
      <path d="M14 4l6 6-8 8H6v-6l8-8z" />
      <path d="M9.5 9.5L4 15" />
      <circle cx="18" cy="6" r="2" fill="currentColor" stroke="none" />
    </svg>
  )
}

export function HistoryIcon({ className }: IconProps) {
  return (
    <svg className={className} {...base} aria-hidden>
      <circle cx="12" cy="12" r="9" />
      <path d="M12 7v5l3 2" />
    </svg>
  )
}

export function OrderIcon({ className }: IconProps) {
  return (
    <svg className={className} {...base} aria-hidden>
      <path d="M6 4h12l2 4v12a2 2 0 01-2 2H6a2 2 0 01-2-2V8l2-4z" />
      <path d="M6 8h14" />
      <path d="M9 12h6" />
    </svg>
  )
}

export function ProfileIcon({ className }: IconProps) {
  return (
    <svg className={className} {...base} aria-hidden>
      <circle cx="12" cy="8" r="4" />
      <path d="M5 20c0-4 3.5-6 7-6s7 2 7 6" />
    </svg>
  )
}

export function HeartIcon({ className }: IconProps) {
  return (
    <svg className={className} viewBox="0 0 24 24" fill="currentColor" aria-hidden>
      <path d="M12 21s-7-4.6-9.5-8.8C.4 8.8 2.6 5 6.2 5c2 0 3.2 1.2 3.8 2.2.6-1 1.8-2.2 3.8-2.2 3.6 0 5.8 3.8 3.7 7.2C19 16.4 12 21 12 21z" />
    </svg>
  )
}

export function SoundOnIcon({ className }: IconProps) {
  return (
    <svg className={className} {...base} aria-hidden>
      <path d="M11 5L6 9H3v6h3l5 4V5z" />
      <path d="M15.5 8.5a5 5 0 010 7M18 6a8 8 0 010 12" />
    </svg>
  )
}

export function SoundOffIcon({ className }: IconProps) {
  return (
    <svg className={className} {...base} aria-hidden>
      <path d="M11 5L6 9H3v6h3l5 4V5z" />
      <path d="M22 9l-6 6M16 9l6 6" />
    </svg>
  )
}

export function ChevronLeftIcon({ className }: IconProps) {
  return (
    <svg className={className} {...base} aria-hidden>
      <path d="M15 18l-6-6 6-6" />
    </svg>
  )
}

export function SettingsIcon({ className }: IconProps) {
  return (
    <svg className={className} {...base} aria-hidden>
      <circle cx="12" cy="12" r="3" />
      <path d="M12 2v2M12 20v2M4.9 4.9l1.4 1.4M17.7 17.7l1.4 1.4M2 12h2M20 12h2M4.9 19.1l1.4-1.4M17.7 6.3l1.4-1.4" />
    </svg>
  )
}

export function LiveIcon({ className }: IconProps) {
  return (
    <svg className={className} {...base} aria-hidden>
      <circle cx="12" cy="12" r="2" fill="currentColor" stroke="none" />
      <path d="M7 12a5 5 0 0110 0M5 12a7 7 0 0114 0M3 12a9 9 0 0118 0" />
    </svg>
  )
}
