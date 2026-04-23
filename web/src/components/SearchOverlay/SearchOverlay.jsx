import { useEffect, useRef, useState } from 'react'
import { Search, X, CornerDownLeft } from 'lucide-react'
import { motion, AnimatePresence } from 'motion/react'
import * as chaptersApi from '../../api/chapters.js'
import { debounce } from '../../utils/throttle.js'
import styles from './SearchOverlay.module.css'

/**
 * Full-text search overlay for a single book.
 *
 * Props:
 *   open
 *   bookId
 *   onJump(hit, query)
 *   onClose()
 */
export function SearchOverlay({ open, bookId, onJump, onClose }) {
  const [query, setQuery] = useState('')
  const [hits, setHits] = useState([])
  const [status, setStatus] = useState('idle') // 'idle' | 'loading' | 'ready' | 'empty' | 'error'
  const [total, setTotal] = useState(0)
  const [active, setActive] = useState(0)
  const inputRef = useRef(null)
  const listRef = useRef(null)

  // Autofocus on open.
  useEffect(() => {
    if (open) {
      setTimeout(() => inputRef.current?.focus(), 10)
    } else {
      setQuery('')
      setHits([])
      setStatus('idle')
      setActive(0)
    }
  }, [open])

  useEffect(() => {
    if (!open) return undefined

    const { body, documentElement } = document
    const prevBodyOverflow = body.style.overflow
    const prevBodyTouchAction = body.style.touchAction
    const prevHtmlOverflow = documentElement.style.overflow

    body.style.overflow = 'hidden'
    body.style.touchAction = 'none'
    documentElement.style.overflow = 'hidden'

    return () => {
      body.style.overflow = prevBodyOverflow
      body.style.touchAction = prevBodyTouchAction
      documentElement.style.overflow = prevHtmlOverflow
    }
  }, [open])

  // Debounced search. Re-created when bookId changes so a stale closure
  // can't fire after a book swap.
  const debouncedSearchRef = useRef(null)
  useEffect(() => {
    if (!open || bookId == null) return
    debouncedSearchRef.current = debounce(async (q) => {
      if (!q || !q.trim()) {
        setHits([])
        setStatus('idle')
        return
      }
      setStatus('loading')
      try {
        const resp = await chaptersApi.searchBook(bookId, q.trim())
        setHits(resp.hits || [])
        setTotal(resp.total || 0)
        setStatus((resp.hits || []).length === 0 ? 'empty' : 'ready')
        setActive(0)
      } catch (err) {
        setStatus('error')
        setHits([])
        // eslint-disable-next-line no-console
        console.warn(err)
      }
    }, 280)
  }, [open, bookId])

  useEffect(() => {
    if (!open) return
    debouncedSearchRef.current?.(query)
  }, [query, open])

  // Keyboard: Esc close, ↑/↓ navigate, Enter select.
  useEffect(() => {
    if (!open) return
    const onKey = (e) => {
      if (e.key === 'Escape') {
        onClose()
      } else if (e.key === 'ArrowDown') {
        e.preventDefault()
        setActive((i) => Math.min(hits.length - 1, i + 1))
      } else if (e.key === 'ArrowUp') {
        e.preventDefault()
        setActive((i) => Math.max(0, i - 1))
      } else if (e.key === 'Enter') {
        const h = hits[active]
        if (h) {
          onJump(h, query.trim())
          onClose()
        }
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [open, hits, active, query, onJump, onClose])

  // Keep active hit in view.
  useEffect(() => {
    const el = listRef.current?.querySelector(`[data-hit-idx="${active}"]`)
    if (el) el.scrollIntoView({ block: 'nearest' })
  }, [active])

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
          <motion.div
            role="dialog"
            aria-label="全书搜索"
            className={`glass-strong ${styles.panel}`}
            initial={{ opacity: 0, y: -24, scale: 0.98 }}
            animate={{ opacity: 1, y: 0, scale: 1 }}
            exit={{ opacity: 0, y: -12, scale: 0.98 }}
            transition={{ type: 'spring', stiffness: 380, damping: 32 }}
          >
            <header className={styles.header}>
              <Search size={18} className={styles.searchIcon} />
              <input
                ref={inputRef}
                className={styles.input}
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                placeholder="在本书中搜索…"
                maxLength={200}
              />
              <kbd className={styles.kbd}>Esc</kbd>
              <button className={styles.close} onClick={onClose}>
                <X size={16} />
              </button>
            </header>

            <div className={styles.status}>
              {status === 'loading' && <span>搜索中…</span>}
              {status === 'ready' && (
                <span>
                  共 <strong>{total}</strong> 条匹配
                  {total >= 100 && ' (已截断)'}
                  {' · '}
                  <span className={styles.hint}>
                    <kbd className={styles.kbdSmall}>↑</kbd>{' '}
                    <kbd className={styles.kbdSmall}>↓</kbd> 选择 ·{' '}
                    <kbd className={styles.kbdSmall}>Enter</kbd> 跳转
                  </span>
                </span>
              )}
              {status === 'empty' && <span>没有匹配结果</span>}
              {status === 'error' && <span className={styles.err}>搜索失败</span>}
              {status === 'idle' && !query && (
                <span className={styles.hint}>输入关键词以搜索</span>
              )}
            </div>

            <ul ref={listRef} className={styles.list}>
              {hits.map((h, i) => (
                <li key={`${h.chapterIdx}-${h.hitSeq ?? h.charOffset}-${i}`} data-hit-idx={i}>
                  <button
                    className={`${styles.item} ${i === active ? styles.itemActive : ''}`}
                    onMouseEnter={() => setActive(i)}
                    onClick={() => {
                      onJump(h, query.trim())
                      onClose()
                    }}
                  >
                    <span className={styles.itemChapter}>
                      {h.chapterTitle || `第 ${h.chapterIdx + 1} 章`}
                    </span>
                    <HitPreview hit={h} />
                    {i === active && (
                      <CornerDownLeft size={13} className={styles.itemEnter} />
                    )}
                  </button>
                </li>
              ))}
            </ul>
          </motion.div>
        </>
      )}
    </AnimatePresence>
  )
}

/** Render the preview with inline highlight using rune-offset coordinates. */
function HitPreview({ hit }) {
  const { preview, previewHighlightStart: start, previewHighlightLength: len } = hit
  if (!preview) return null
  const runes = [...preview]
  if (start < 0 || len <= 0 || start + len > runes.length) {
    return <span className={styles.itemPreview}>{preview}</span>
  }
  const before = runes.slice(0, start).join('')
  const match = runes.slice(start, start + len).join('')
  const after = runes.slice(start + len).join('')
  return (
    <span className={styles.itemPreview}>
      …{before}
      <mark className={styles.itemMark}>{match}</mark>
      {after}…
    </span>
  )
}
