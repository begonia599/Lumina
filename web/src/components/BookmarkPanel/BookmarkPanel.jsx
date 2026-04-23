import { useEffect, useState } from 'react'
import { motion, AnimatePresence } from 'motion/react'
import { X, Plus, Trash2, Pencil, Check } from 'lucide-react'
import * as bookmarksApi from '../../api/bookmarks.js'
import styles from './BookmarkPanel.module.css'

/**
 * BookmarkPanel — reader-side drawer for managing bookmarks.
 *
 * Props:
 *   open: boolean
 *   bookId
 *   chapters: full chapter list (for showing chapter titles on each mark)
 *   currentChapterIdx, currentCharOffset: for "add at current position"
 *   onJump(bookmarkOrChapterIdx, charOffset?)
 *   onAddCurrent() -> Promise<bookmark | null>
 *   onClose()
 */
export function BookmarkPanel({
  open,
  bookId,
  chapters,
  currentChapterIdx,
  currentCharOffset,
  onJump,
  onAddCurrent,
  onClose,
}) {
  const [bookmarks, setBookmarks] = useState([])
  const [status, setStatus] = useState('idle')
  const [error, setError] = useState(null)
  const [editingId, setEditingId] = useState(null)
  const [editDraft, setEditDraft] = useState('')

  // Load on open; clear on close.
  useEffect(() => {
    if (!open || bookId == null) return
    let cancelled = false
    setStatus('loading')
    setError(null)
    bookmarksApi
      .listBookmarks(bookId)
      .then((list) => {
        if (cancelled) return
        setBookmarks(list || [])
        setStatus('ready')
      })
      .catch((err) => {
        if (cancelled) return
        setError(err.message || '加载失败')
        setStatus('error')
      })
    return () => {
      cancelled = true
    }
  }, [open, bookId])

  useEffect(() => {
    if (!open) return
    const onKey = (e) => {
      if (e.key === 'Escape') onClose()
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [open, onClose])

  async function handleAdd() {
    try {
      if (onAddCurrent) {
        const bm = await onAddCurrent()
        if (bm) {
          setBookmarks((list) => [bm, ...list.filter((item) => item.id !== bm.id)])
        }
        return
      }
      const bm = await bookmarksApi.createBookmark(bookId, {
        chapterIdx: currentChapterIdx,
        charOffset: currentCharOffset,
        note: '',
      })
      setBookmarks((list) => [bm, ...list])
    } catch (err) {
      setError(err.message || '添加失败')
    }
  }

  async function handleDelete(id) {
    try {
      await bookmarksApi.deleteBookmark(id)
      setBookmarks(bookmarks.filter((b) => b.id !== id))
    } catch (err) {
      setError(err.message || '删除失败')
    }
  }

  function beginEdit(bm) {
    setEditingId(bm.id)
    setEditDraft(bm.note || '')
  }

  function cancelEdit() {
    setEditingId(null)
    setEditDraft('')
  }

  async function saveEdit(id) {
    try {
      const updated = await bookmarksApi.updateBookmark(id, { note: editDraft })
      setBookmarks((list) => list.map((b) => (b.id === id ? updated : b)))
      cancelEdit()
    } catch (err) {
      setError(err.message || '保存失败')
    }
  }

  function chapterTitle(idx) {
    return chapters.find((c) => c.chapterIdx === idx)?.title || `第 ${idx + 1} 章`
  }

  return (
    <AnimatePresence>
      {open && (
        <>
          <motion.div
            className={styles.backdrop}
            onClick={onClose}
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            transition={{ duration: 0.2 }}
          />
          <motion.aside
            className={`glass-strong ${styles.panel}`}
            role="dialog"
            aria-label="书签"
            initial={{ x: '105%' }}
            animate={{ x: 0 }}
            exit={{ x: '105%' }}
            transition={{ type: 'spring', stiffness: 320, damping: 34 }}
            drag="x"
            dragConstraints={{ left: 0, right: 400 }}
            dragElastic={0.08}
            dragMomentum={false}
            onDragEnd={(_, info) => {
              if (info.offset.x > 100 || info.velocity.x > 500) onClose()
            }}
          >
            <div className={styles.dragHandle} aria-hidden="true" />
            <header className={styles.header}>
              <div>
                <h2 className={styles.heading}>书签</h2>
                <span className={styles.count}>{bookmarks.length} 条</span>
              </div>
              <button className={styles.close} onClick={onClose} title="关闭 (Esc)">
                <X size={18} />
              </button>
            </header>

            <button className={styles.addBtn} onClick={handleAdd}>
              <Plus size={14} />
              在当前位置加书签
            </button>

            {error && <div className={styles.error}>{error}</div>}

            {status === 'loading' && <div className={styles.hint}>加载中…</div>}

            {status === 'ready' && bookmarks.length === 0 && (
              <div className={styles.empty}>
                <p>还没有书签</p>
                <p className={styles.emptySub}>读到有感觉的地方，按 B 即可标记。</p>
              </div>
            )}

            <ul className={styles.list}>
              {bookmarks.map((bm) => (
                <li key={bm.id} className={styles.item}>
                  <button
                    className={styles.itemJump}
                    onClick={() => onJump(bm)}
                  >
                    <span className={styles.itemChapter}>
                      {chapterTitle(bm.chapterIdx)}
                    </span>
                    <span className={styles.itemMeta}>
                      <span className={styles.itemTime}>
                        {formatRelative(bm.createdAt)}
                      </span>
                      {bm.charOffset > 0 && (
                        <>
                          <span className={styles.itemDot} />
                          <span className={styles.itemOffset}>
                            偏移 {bm.charOffset} 字
                          </span>
                        </>
                      )}
                    </span>
                  </button>

                  {editingId === bm.id ? (
                    <div className={styles.editRow}>
                      <input
                        className={styles.editInput}
                        value={editDraft}
                        onChange={(e) => setEditDraft(e.target.value)}
                        placeholder="添加备注…"
                        maxLength={200}
                        autoFocus
                        onKeyDown={(e) => {
                          if (e.key === 'Enter') saveEdit(bm.id)
                          if (e.key === 'Escape') cancelEdit()
                        }}
                      />
                      <button
                        className={styles.iconBtn}
                        onClick={() => saveEdit(bm.id)}
                        title="保存"
                      >
                        <Check size={14} />
                      </button>
                    </div>
                  ) : (
                    bm.note && <p className={styles.itemNote}>{bm.note}</p>
                  )}

                  <div className={styles.itemActions}>
                    {editingId !== bm.id && (
                      <button
                        className={styles.iconBtn}
                        onClick={() => beginEdit(bm)}
                        title={bm.note ? '编辑备注' : '添加备注'}
                      >
                        <Pencil size={13} />
                      </button>
                    )}
                    <button
                      className={`${styles.iconBtn} ${styles.iconBtnDanger}`}
                      onClick={() => handleDelete(bm.id)}
                      title="删除"
                    >
                      <Trash2 size={13} />
                    </button>
                  </div>
                </li>
              ))}
            </ul>
          </motion.aside>
        </>
      )}
    </AnimatePresence>
  )
}

function formatRelative(iso) {
  try {
    const d = new Date(iso)
    const diff = Date.now() - d.getTime()
    const m = Math.floor(diff / 60000)
    if (m < 1) return '刚才'
    if (m < 60) return `${m} 分钟前`
    const h = Math.floor(m / 60)
    if (h < 24) return `${h} 小时前`
    const day = Math.floor(h / 24)
    if (day < 30) return `${day} 天前`
    return d.toLocaleDateString()
  } catch {
    return ''
  }
}
