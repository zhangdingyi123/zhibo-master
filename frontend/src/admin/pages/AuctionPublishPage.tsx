import { Link, useNavigate, useParams } from 'react-router-dom'
import { useEffect, useState } from 'react'
import { getProduct, publishAuction } from '../../api/admin'
import type { ProductView } from '../../api/types'
import { AuctionRulesForm } from '../components/AuctionRulesForm'

export function AuctionPublishPage() {
  const { id } = useParams()
  const navigate = useNavigate()
  const [product, setProduct] = useState<ProductView | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const pid = Number(id)
    if (!pid) return
    getProduct(pid)
      .then((p) => {
        if (p.auction) {
          throw new Error('该商品已有进行中的竞拍场次')
        }
        setProduct(p)
      })
      .catch((e) => setError(e instanceof Error ? e.message : '加载失败'))
  }, [id])

  if (error) {
    return (
      <div className="admin-page">
        <p className="form-error">{error}</p>
        <Link to={`/admin/products/${id}`}>返回商品详情</Link>
      </div>
    )
  }

  if (!product) return <p className="muted">加载中…</p>

  return (
    <div className="admin-page">
      <div className="admin-page__head">
        <div>
          <h2>发布竞拍</h2>
          <p className="page-desc">{product.name}</p>
          <Link to={`/admin/products/${product.id}`} className="breadcrumb">
            ← 返回详情
          </Link>
        </div>
      </div>
      <div className="admin-card">
        <AuctionRulesForm
          submitLabel="发布竞拍场次"
          onSubmit={async (values) => {
            const body = {
              startingPrice: values.startingPrice,
              bidIncrement: values.bidIncrement,
              capPrice: values.capPrice,
              durationSec: values.durationSec,
              extendThresholdSec: values.extendThresholdSec,
              extendSec: values.extendSec,
              scheduledStartAt: values.scheduledStartAt || undefined,
            }
            await publishAuction(product.id, body)
            navigate(`/admin/products/${product.id}`)
          }}
        />
      </div>
    </div>
  )
}
