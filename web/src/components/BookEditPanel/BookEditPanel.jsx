import { useEffect, useRef, useState } from 'react'
import { motion, AnimatePresence } from 'motion/react'
import { X, Upload, Trash2, Plus } from 'lucide-react'
import { useBookStore } from '../../stores/useBookStore.js'
import { BookCover } from '../BookCover/BookCover.jsx'
import styles from './BookEditPanel.module.css'

/**
 * BookEditPanel — right-side drawer for editing book metadata.
 *
 * Props:
 *   book: the currently-edited book (null → panel closed)
 *   onClose()
 *
 * Behavior:
 *   - Title / author / description / tags saved on "保存" click.
 *   - Cover upload is its own action — uploads immediately, updates cover in-place.
 *   - "恢复默认封面" deletes the uploaded cover.
 *   - Esc / backdrop closes (prompts if unsaved changes).
 */
export function BookEditPanel({ book, onClose }) {
  const patch = useBookStore((s) => s.patch)
  const setCover = useBookStore((s) => s.setCover)
  const removeCover = useBookStore((s) => s.removeCover)

  const [title, setTitle] = useState('')
  const [author, setAuthor] = useState('')
  const [description, setDescription] = useState('')
  const [tags, setTags] = useState([])
  const [tagDraft, setTagDraft] = useState('')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState(null)
  const [coverBusy, setCoverBusy] = useState(false)
  const fileRef = useRef(null)

  // Re-seed form whenever a new book flows in.
  useEffect(() => {
    if (book) {
      setTitle(book.title || '')
      setAuthor(book.author || '')
      setDescription(book.description || '')
      setTags(book.tags || [])
      setTagDraft('')
      setError(null)
    }
  }, [book?.id]) // eslint-disable-line react-hooks/exhaustive-deps

  const open = !!book

  // Esc to close.
  useEffect(() => {
    if (!open) return
    const onKey = (e) => {
      if (e.key === 'Escape') requestClose()
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [open]) // eslint-disable-line react-hooks/exhaustive-deps

  const dirty =
    !!book &&
    (title !== (book.title || '') ||
      author !== (book.author || '') ||
      description !== (book.description || '') ||
      JSON.stringify(tags) !== JSON.stringify(book.tags || []))

  function requestClose() {
    if (dirty && !window.confirm('有未保存的修改，确认关闭？')) return
    onClose()
  }

  function addTag() {
    const t = tagDraft.trim()
    if (!t) return
    if (tags.includes(t)) {
      setTagDraft('')
      return
    }
    if (tags.length >= 20) {
      setError('最多 20 个标签')
      return
    }
    setTags([...tags, t])
    setTagDraft('')
  }

  function removeTag(t) {
    setTags(tags.filter((x) => x !== t))
  }

  async function handleSave() {
    if (!book) return
    setError(null)
    setSaving(true)
    try {
      await patch(book.id, { title, author, description, tags })
      onClose()
    } catch (err) {
      setError(err.message || '保存失败')
    } finally {
      setSaving(false)
    }
  }

  async function handleCoverFile(file) {
    if (!file || !book) return
    if (file.size > 2 * 1024 * 1024) {
      setError('封面不能超过 2MB')
      return
    }
    setError(null)
    setCoverBusy(true)
    try {
      await setCover(book.id, file)
    } catch (err) {
      setError(err.message || '封面上传失败')
    } finally {
      setCoverBusy(false)
    }
  }

  async function handleCoverRemove() {
    if (!book) return
    setCoverBusy(true)
    try {
      await removeCover(book.id)
    } catch (err) {
      setError(err.message || '封面删除失败')
    } finally {
      setCoverBusy(false)
    }
  }

  // Keep panel mounted for exit transition, but only render content when there's a book.
  return (
    <AnimatePresence>
      {open && (
        <>
          <motion.div
            className={styles.backdrop}
            onClick={requestClose}
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            transition={{ duration: 0.2 }}
          />
          <motion.aside
            className={`glass-strong ${styles.panel}`}
            role="dialog"
            aria-label="编辑书籍"
            initial={{ x: '105%' }}
            animate={{ x: 0 }}
            exit={{ x: '105%' }}
            transition={{ type: 'spring', stiffness: 320, damping: 34 }}
            drag="x"
            dragConstraints={{ left: 0, right: 480 }}
            dragElastic={0.08}
            dragMomentum={false}
            onDragEnd={(_, info) => {
              if (info.offset.x > 100 || info.velocity.x > 500) requestClose()
            }}
          >
            <div className={styles.dragHandle} aria-hidden="true" />
            {book && (
          <>
            <header className={styles.header}>
              <h2 className={styles.heading}>编辑书籍</h2>
              <button className={styles.close} onClick={requestClose} title="关闭 (Esc)">
                <X size={18} />
              </button>
            </header>

            <section className={styles.coverSection}>
              <div className={styles.coverPreview}>
                <BookCover book={book} title={title || book.title} size="lg" />
              </div>
              <div className={styles.coverActions}>
                <input
                  ref={fileRef}
                  type="file"
                  accept="image/jpeg,image/png,image/webp"
                  className={styles.hiddenFile}
                  onChange={(e) => handleCoverFile(e.target.files?.[0])}
                />
                <button
                  className={styles.coverBtn}
                  onClick={() => fileRef.current?.click()}
                  disabled={coverBusy}
                >
                  <Upload size={14} />
                  {book.hasCover ? '更换封面' : '上传封面'}
                </button>
                {book.hasCover && (
                  <button
                    className={`${styles.coverBtn} ${styles.coverBtnDanger}`}
                    onClick={handleCoverRemove}
                    disabled={coverBusy}
                  >
                    <Trash2 size={14} />
                    恢复默认
                  </button>
                )}
                <p className={styles.coverHint}>
                  JPEG / PNG / WebP，最大 2MB。推荐比例 2:3。
                </p>
              </div>
            </section>

            <section className={styles.fields}>
              <Field label="书名">
                <input
                  type="text"
                  className={styles.input}
                  value={title}
                  onChange={(e) => setTitle(e.target.value)}
                  maxLength={200}
                  required
                />
              </Field>

              <Field label="作者">
                <input
                  type="text"
                  className={styles.input}
                  value={author}
                  onChange={(e) => setAuthor(e.target.value)}
                  placeholder="选填"
                  maxLength={80}
                />
              </Field>

              <Field label="简介">
                <textarea
                  className={`${styles.input} ${styles.textarea}`}
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  placeholder="可填写一段描述、摘抄或阅读笔记"
                  rows={4}
                  maxLength={2000}
                />
                <div className={styles.counter}>
                  {description.length} / 2000
                </div>
              </Field>

              <Field label="标签">
                <div className={styles.tagRow}>
                  {tags.map((t) => (
                    <span key={t} className={styles.tag}>
                      {t}
                      <button
                        className={styles.tagRemove}
                        onClick={() => removeTag(t)}
                        title="移除"
                      >
                        <X size={12} />
                      </button>
                    </span>
                  ))}
                  <span className={styles.tagInputWrap}>
                    <input
                      className={styles.tagInput}
                      value={tagDraft}
                      onChange={(e) => setTagDraft(e.target.value)}
                      onKeyDown={(e) => {
                        if (e.key === 'Enter' || e.key === ',') {
                          e.preventDefault()
                          addTag()
                        } else if (e.key === 'Backspace' && !tagDraft && tags.length) {
                          setTags(tags.slice(0, -1))
                        }
                      }}
                      placeholder={tags.length === 0 ? '科幻、文学、长篇…' : '+'}
                      maxLength={32}
                    />
                    {tagDraft && (
                      <button className={styles.tagAdd} onClick={addTag}>
                        <Plus size={12} />
                      </button>
                    )}
                  </span>
                </div>
                <p className={styles.coverHint}>
                  回车或逗号添加；退格删除最后一个
                </p>
              </Field>
            </section>

            {error && <div className={styles.error}>{error}</div>}

            <footer className={styles.footer}>
              <button className={styles.cancel} onClick={requestClose}>
                取消
              </button>
              <button
                className={styles.save}
                onClick={handleSave}
                disabled={saving || !dirty}
              >
                {saving ? '保存中…' : '保存'}
              </button>
            </footer>
          </>
        )}
          </motion.aside>
        </>
      )}
    </AnimatePresence>
  )
}

function Field({ label, children }) {
  return (
    <label className={styles.field}>
      <span className={styles.label}>{label}</span>
      {children}
    </label>
  )
}
