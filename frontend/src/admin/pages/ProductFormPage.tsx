import { Link, useNavigate, useParams } from 'react-router-dom'
import { useEffect, useState } from 'react'
import { createProduct, getProduct, updateProduct } from '../../api/admin'
import type { ProductView } from '../../api/types'
import { ProductForm } from '../components/ProductForm'

export function ProductCreatePage() {
  const navigate = useNavigate()

  return (
    <div className="admin-page">
      <div className="admin-page__head">
        <div>
          <h2>新建商品</h2>
          <Link to="/admin/products" className="breadcrumb">
            ← 返回列表
          </Link>
        </div>
      </div>
      <div className="admin-card">
        <ProductForm
          submitLabel="创建商品"
          onSubmit={async (body) => {
            const p = await createProduct(body)
            navigate(`/admin/products/${p.id}`)
          }}
        />
      </div>
    </div>
  )
}

export function ProductEditPage() {
  const { id } = useParams()
  const navigate = useNavigate()
  const [product, setProduct] = useState<ProductView | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const pid = Number(id)
    if (!pid) return
    getProduct(pid)
      .then(setProduct)
      .catch((e) => setError(e instanceof Error ? e.message : '加载失败'))
  }, [id])

  if (error) return <p className="form-error">{error}</p>
  if (!product) return <p className="muted">加载中…</p>

  return (
    <div className="admin-page">
      <div className="admin-page__head">
        <div>
          <h2>编辑商品</h2>
          <Link to={`/admin/products/${product.id}`} className="breadcrumb">
            ← 返回详情
          </Link>
        </div>
      </div>
      <div className="admin-card">
        <ProductForm
          initial={product}
          submitLabel="保存修改"
          onSubmit={async (body) => {
            await updateProduct(product.id, body)
            navigate(`/admin/products/${product.id}`)
          }}
        />
      </div>
    </div>
  )
}
