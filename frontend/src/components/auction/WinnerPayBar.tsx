import { Link } from 'react-router-dom'
import { formatCents } from '../../utils/money'

type Props = {
  amount: number
  orderId: number
  productName?: string
  onDismiss: () => void
}

/** 连播中标后的轻量支付条，不跳转、不打断观看 */
export function WinnerPayBar({ amount, orderId, productName, onDismiss }: Props) {
  return (
    <div className="winner-pay-bar" role="status">
      <div className="winner-pay-bar__text">
        <strong>恭喜中标</strong>
        {productName && <span> · {productName}</span>}
        <span className="muted"> · 可稍后支付</span>
      </div>
      <div className="winner-pay-bar__actions">
        <Link to={`/app/orders/${orderId}`} className="btn-primary btn-sm">
          支付 {formatCents(amount)}
        </Link>
        <button type="button" className="btn-ghost btn-sm" onClick={onDismiss}>
          继续观看
        </button>
      </div>
    </div>
  )
}
