import type { AuctionToast } from '../../hooks/useAuctionNotifications'

type Props = {
  toasts: AuctionToast[]
  onDismiss: (id: string) => void
}

export function ToastStack({ toasts, onDismiss }: Props) {
  if (toasts.length === 0) return null

  return (
    <div className="toast-stack" aria-live="polite">
      {toasts.map((t) => (
        <div
          key={t.id}
          className={`toast toast--${t.kind}`}
          role="status"
          onClick={() => onDismiss(t.id)}
        >
          <strong>{t.title}</strong>
          {t.message && <p>{t.message}</p>}
        </div>
      ))}
    </div>
  )
}
