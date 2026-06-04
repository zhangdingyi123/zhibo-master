type BadgeVariant = 'product' | 'session' | 'order' | 'neutral'

const VARIANT_CLASS: Record<BadgeVariant, string> = {
  product: 'badge--product',
  session: 'badge--session',
  order: 'badge--order',
  neutral: 'badge--neutral',
}

interface StatusBadgeProps {
  label: string
  variant?: BadgeVariant
  tone?: string
}

export function StatusBadge({
  label,
  variant = 'neutral',
  tone,
}: StatusBadgeProps) {
  const className = [
    'badge',
    VARIANT_CLASS[variant],
    tone ? `badge--${tone}` : '',
  ]
    .filter(Boolean)
    .join(' ')
  return <span className={className}>{label}</span>
}
