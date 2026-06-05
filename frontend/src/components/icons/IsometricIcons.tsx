import type { ReactNode } from 'react'

type IconProps = { className?: string }

/** 等距风格图标 — 用于消息中心 Bento 卡片 */

function IsoBase({ className, children }: IconProps & { children: ReactNode }) {
  return (
    <svg
      className={className}
      viewBox="0 0 48 48"
      fill="none"
      aria-hidden
      xmlns="http://www.w3.org/2000/svg"
    >
      {children}
    </svg>
  )
}

export function IsoBellIcon({ className }: IconProps) {
  return (
    <IsoBase className={className}>
      <path d="M24 8L36 14v10l-12 7L12 24V14l12-6z" fill="currentColor" opacity="0.25" />
      <path d="M24 8v19M12 14l12 7 12-7" stroke="currentColor" strokeWidth="1.5" strokeLinejoin="round" />
      <path d="M18 32c0 3.3 2.7 6 6 6s6-2.7 6-6" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
      <circle cx="24" cy="8" r="2" fill="currentColor" />
    </IsoBase>
  )
}

export function IsoTrophyIcon({ className }: IconProps) {
  return (
    <IsoBase className={className}>
      <path d="M16 12h16l4 8-10 6-10-6 4-8z" fill="currentColor" opacity="0.3" />
      <path d="M16 12h16M20 12V8h8v4" stroke="currentColor" strokeWidth="1.5" strokeLinejoin="round" />
      <path d="M12 20h24M18 32h12v4H18v-4z" stroke="currentColor" strokeWidth="1.5" strokeLinejoin="round" />
      <path d="M20 26l4 4 8-8" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
    </IsoBase>
  )
}

export function IsoClockIcon({ className }: IconProps) {
  return (
    <IsoBase className={className}>
      <path d="M24 10l14 8v12L24 38 10 30V18l14-8z" fill="currentColor" opacity="0.22" />
      <path d="M24 10l14 8-14 8-14-8 14-8z" stroke="currentColor" strokeWidth="1.5" strokeLinejoin="round" />
      <path d="M24 22v6l4 3" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
    </IsoBase>
  )
}

export function IsoAlertIcon({ className }: IconProps) {
  return (
    <IsoBase className={className}>
      <path d="M24 6L40 34H8L24 6z" fill="currentColor" opacity="0.28" />
      <path d="M24 6L40 34H8L24 6z" stroke="currentColor" strokeWidth="1.5" strokeLinejoin="round" />
      <path d="M24 18v10" stroke="currentColor" strokeWidth="2" strokeLinecap="round" />
      <circle cx="24" cy="32" r="1.5" fill="currentColor" />
    </IsoBase>
  )
}

export function IsoGavelIcon({ className }: IconProps) {
  return (
    <IsoBase className={className}>
      <path d="M10 30l8-14 14 8-8 14-14-8z" fill="currentColor" opacity="0.25" />
      <path d="M10 30l8-14 14 8-8 14-14-8z" stroke="currentColor" strokeWidth="1.5" strokeLinejoin="round" />
      <path d="M30 16l6-6M14 36l-4 4" stroke="currentColor" strokeWidth="2" strokeLinecap="round" />
    </IsoBase>
  )
}

export function IsoInboxIcon({ className }: IconProps) {
  return (
    <IsoBase className={className}>
      <path d="M8 18h32l-4 16H12L8 18z" fill="currentColor" opacity="0.22" />
      <path d="M8 18h32l-4 16H12L8 18z" stroke="currentColor" strokeWidth="1.5" strokeLinejoin="round" />
      <path d="M8 18l8-8h16l8 8" stroke="currentColor" strokeWidth="1.5" strokeLinejoin="round" />
      <path d="M20 26h8l4 4H16l4-4z" stroke="currentColor" strokeWidth="1.5" strokeLinejoin="round" />
    </IsoBase>
  )
}

export function messageEventIcon(eventType: string) {
  switch (eventType) {
    case 'outbid':
      return IsoGavelIcon
    case 'extended':
      return IsoClockIcon
    case 'settled_win':
      return IsoTrophyIcon
    case 'settled':
      return IsoBellIcon
    case 'cancelled':
      return IsoAlertIcon
    default:
      return IsoInboxIcon
  }
}
