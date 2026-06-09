import { useState } from 'react'
import { generateProductIntro } from '../../api/admin'
import type { ProductBody, ProductView } from '../../api/types'

interface ProductFormProps {
  initial?: ProductView
  onSubmit: (body: ProductBody) => Promise<void>
  submitLabel: string
}

function parseImages(raw: string): string[] {
  return raw
    .split(/[\n,]/)
    .map((s) => s.trim())
    .filter(Boolean)
}

export function ProductForm({ initial, onSubmit, submitLabel }: ProductFormProps) {
  const [name, setName] = useState(initial?.name ?? '')
  const [description, setDescription] = useState(initial?.description ?? '')
  const [coverUrl, setCoverUrl] = useState(initial?.coverUrl ?? '')
  const [imagesRaw, setImagesRaw] = useState(
    (initial?.images?.length ? initial.images : initial?.coverUrl ? [initial.coverUrl] : []).join('\n'),
  )
  const [keywords, setKeywords] = useState('')
  const [aiHint, setAiHint] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)
  const [aiLoading, setAiLoading] = useState(false)

  async function handleAiGenerate() {
    setError(null)
    setAiHint(null)
    if (!name.trim()) {
      setError('请先填写商品名称')
      return
    }
    setAiLoading(true)
    try {
      const result = await generateProductIntro({
        name: name.trim(),
        keywords: keywords.trim() || undefined,
      })
      setDescription(result.description)
      setAiHint(
        result.source === 'llm'
          ? '已由大模型生成，请确认后保存'
          : '未配置 AI_API_KEY，已使用模板文案，可编辑后保存',
      )
    } catch (err) {
      setError(err instanceof Error ? err.message : 'AI 生成失败')
    } finally {
      setAiLoading(false)
    }
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError(null)
    if (!name.trim()) {
      setError('请填写商品名称')
      return
    }
    const images = parseImages(imagesRaw)
    const cover = coverUrl.trim() || images[0] || ''
    if (!cover) {
      setError('请填写封面图 URL 或图片列表')
      return
    }

    setLoading(true)
    try {
      await onSubmit({
        name: name.trim(),
        description: description.trim(),
        coverUrl: cover,
        images: images.length ? images : [cover],
      })
    } catch (err) {
      setError(err instanceof Error ? err.message : '保存失败')
    } finally {
      setLoading(false)
    }
  }

  return (
    <form className="admin-form" onSubmit={handleSubmit}>
      <label>
        商品名称 *
        <input value={name} onChange={(e) => setName(e.target.value)} required />
      </label>
      <label>
        卖点关键词（AI 生成可选）
        <input
          value={keywords}
          onChange={(e) => setKeywords(e.target.value)}
          placeholder="如：纯棉、透气、直播间专拍"
        />
      </label>
      <div className="admin-form__ai-row">
        <label>
          商品介绍
          <textarea
            rows={5}
            value={description}
            onChange={(e) => {
              setDescription(e.target.value)
              setAiHint(null)
            }}
            placeholder="可点击「AI 生成介绍」自动撰写直播口播稿"
          />
        </label>
        <button
          type="button"
          className="btn-secondary admin-form__ai-btn"
          onClick={handleAiGenerate}
          disabled={aiLoading || loading}
        >
          {aiLoading ? '生成中…' : '✨ AI 生成介绍'}
        </button>
      </div>
      {aiHint && <p className="form-hint">{aiHint}</p>}
      <label>
        封面图 URL *
        <input
          type="url"
          placeholder="https://..."
          value={coverUrl}
          onChange={(e) => setCoverUrl(e.target.value)}
        />
      </label>
      {coverUrl && (
        <div className="image-preview">
          <img src={coverUrl} alt="封面预览" onError={(e) => { (e.target as HTMLImageElement).style.display = 'none' }} />
        </div>
      )}
      <label>
        商品图 URL（每行一个，或用逗号分隔）
        <textarea
          rows={3}
          value={imagesRaw}
          onChange={(e) => setImagesRaw(e.target.value)}
          placeholder="https://example.com/1.jpg"
        />
      </label>
      {error && <p className="form-error">{error}</p>}
      <div className="admin-form__actions">
        <button type="submit" className="btn-primary" disabled={loading}>
          {loading ? '保存中…' : submitLabel}
        </button>
      </div>
    </form>
  )
}
