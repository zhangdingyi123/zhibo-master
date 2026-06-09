type Props = {
  visible: boolean
  productName: string
  coverUrl?: string
}

/** 连播切品时的全屏轻引导，不阻断观看 */
export function NextUpBanner({ visible, productName, coverUrl }: Props) {
  if (!visible) return null

  return (
    <div className="next-up-banner" role="status" aria-live="assertive">
      <div className="next-up-banner__card">
        {coverUrl && (
          <img src={coverUrl} alt="" className="next-up-banner__thumb" />
        )}
        <div className="next-up-banner__text">
          <span className="next-up-banner__label">下一件即将开拍</span>
          <strong className="next-up-banner__name">{productName}</strong>
        </div>
      </div>
    </div>
  )
}
