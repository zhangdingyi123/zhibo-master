import { useRef, useState } from 'react'
import { generateProductIntro, uploadImage } from '../../api/admin'
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

function appendImages(raw: string, urls: string[]): string {
  const existing = parseImages(raw)
  const merged = [...existing, ...urls.filter((u) => !existing.includes(u))]
  return merged.join('\n')
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
  const [uploadingCover, setUploadingCover] = useState(false)
  const [uploadingGallery, setUploadingGallery] = useState(false)
  const coverInputRef = useRef<HTMLInputElement>(null)
  const galleryInputRef = useRef<HTMLInputElement>(null)

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

  async function handleCoverFileChange(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0]
    e.target.value = ''
    if (!file) return

    setError(null)
    setUploadingCover(true)
    try {
      const { url } = await uploadImage(file)
      setCoverUrl(url)
      setImagesRaw((prev) => {
        const images = parseImages(prev)
        if (images.includes(url)) return prev
        return images.length ? prev : url
      })
    } catch (err) {
      setError(err instanceof Error ? err.message : '封面图上传失败')
    } finally {
      setUploadingCover(false)
    }
  }

  async function handleGalleryFileChange(e: React.ChangeEvent<HTMLInputElement>) {
    const files = e.target.files
    e.target.value = ''
    if (!files?.length) return

    setError(null)
    setUploadingGallery(true)
    try {
      const urls: string[] = []
      for (const file of Array.from(files)) {
        const { url } = await uploadImage(file)
        urls.push(url)
      }
      setImagesRaw((prev) => appendImages(prev, urls))
      if (!coverUrl.trim() && urls[0]) {
        setCoverUrl(urls[0])
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : '商品图上传失败')
    } finally {
      setUploadingGallery(false)
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
      setError('请上传封面图或填写图片 URL')
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

  const galleryPreview = parseImages(imagesRaw)

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
        封面图 *
        <input
          type="url"
          placeholder="https://... 或点击下方选择本地图片"
          value={coverUrl}
          onChange={(e) => setCoverUrl(e.target.value)}
        />
      </label>
      <div className="image-upload-row">
        <input
          ref={coverInputRef}
          type="file"
          accept="image/jpeg,image/png,image/gif,image/webp"
          className="image-upload-input"
          onChange={handleCoverFileChange}
        />
        <button
          type="button"
          className="btn-secondary"
          onClick={() => coverInputRef.current?.click()}
          disabled={uploadingCover || loading}
        >
          {uploadingCover ? '上传中…' : '选择本地封面图'}
        </button>
        <span className="field-hint">支持 JPG / PNG / GIF / WebP，最大 5MB</span>
      </div>
      {coverUrl && (
        <div className="image-preview">
          <img src={coverUrl} alt="封面预览" onError={(e) => { (e.target as HTMLImageElement).style.display = 'none' }} />
        </div>
      )}
      <label>
        商品图（每行一个 URL，或用逗号分隔）
        <textarea
          rows={3}
          value={imagesRaw}
          onChange={(e) => setImagesRaw(e.target.value)}
          placeholder="https://example.com/1.jpg"
        />
      </label>
      <div className="image-upload-row">
        <input
          ref={galleryInputRef}
          type="file"
          accept="image/jpeg,image/png,image/gif,image/webp"
          multiple
          className="image-upload-input"
          onChange={handleGalleryFileChange}
        />
        <button
          type="button"
          className="btn-secondary"
          onClick={() => galleryInputRef.current?.click()}
          disabled={uploadingGallery || loading}
        >
          {uploadingGallery ? '上传中…' : '选择本地商品图（可多选）'}
        </button>
      </div>
      {galleryPreview.length > 0 && (
        <div className="image-preview-grid">
          {galleryPreview.map((url) => (
            <img key={url} src={url} alt="商品图预览" onError={(e) => { (e.target as HTMLImageElement).style.display = 'none' }} />
          ))}
        </div>
      )}
      {error && <p className="form-error">{error}</p>}
      <div className="admin-form__actions">
        <button type="submit" className="btn-primary" disabled={loading || uploadingCover || uploadingGallery}>
          {loading ? '保存中…' : submitLabel}
        </button>
      </div>
    </form>
  )
}
