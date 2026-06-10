type Props = {
  liveTitle?: string
  status?: 'idle' | 'live' | 'ended'
}

export function LiveRoomEmptyBanner({ liveTitle, status = 'idle' }: Props) {
  const hint =
    status === 'ended'
      ? '本场直播已结束'
      : status === 'live'
        ? '主播正在准备商品，请稍候…'
        : '直播尚未开始，请等待主播开播'

  return (
    <div className="live-room-empty" role="status">
      <p className="live-room-empty__title">{liveTitle ?? '直播间'}</p>
      <p className="live-room-empty__hint">{hint}</p>
      <p className="live-room-empty__sub muted">商品上架后将自动展示画面与出价面板</p>
    </div>
  )
}
