export type StreamInfo = {
  roomId: string
  pushUrl: string
  hlsUrl: string
  flvUrl: string
}

export async function getStreamInfo(roomId: string): Promise<StreamInfo> {
  const res = await fetch(`/api/v1/streams/${encodeURIComponent(roomId)}`)
  if (!res.ok) {
    throw new Error('获取推流地址失败')
  }
  return res.json() as Promise<StreamInfo>
}

/** 浏览器拉流地址：开发走 Vite 代理，生产走 nginx /live */
export function hlsPlayUrl(roomId: string): string {
  return `/live/${encodeURIComponent(roomId)}.m3u8`
}
