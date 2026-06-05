# 直播画面说明

当前版本使用 **Mock 占位画面**，不依赖 SRS / RTMP / HLS 推流。

用户端直播间（`/app/live/:roomId`）展示：

- 商品封面 + 缓慢缩放动画
- 扫描线 / 光晕叠加
- **LIVE** 角标与动态观看人数

竞拍实时数据（出价、排名、倒计时）仍通过 WebSocket 同步，与视频画面无关。

> 历史推拉流方案（SRS）相关文件仍保留在 `deploy/srs.conf`、`docker-compose.yml`，默认不再启用。
