import { useEffect, useMemo, useState } from 'react'
import { motion, AnimatePresence } from 'motion/react'
import { useBookStore } from '../../stores/useBookStore.js'
import { TopBar } from '../../components/TopBar/TopBar.jsx'
import { BookCard } from '../../components/BookCard/BookCard.jsx'
import { BookCover } from '../../components/BookCover/BookCover.jsx'
import { UploadZone } from '../../components/UploadZone/UploadZone.jsx'
import { SettingsPanel } from '../../components/SettingsPanel/SettingsPanel.jsx'
import { BookEditPanel } from '../../components/BookEditPanel/BookEditPanel.jsx'
import { Link } from 'react-router-dom'
import { ChevronRight, Search, SlidersHorizontal, Trash2 } from 'lucide-react'
import { estimateMinutes, formatMinutes } from '../../utils/reading-time.js'
import styles from './Bookshelf.module.css'

const SORTS = [
  { id: 'recent', label: '最近阅读' },
  { id: 'uploaded', label: '上传时间' },
  { id: 'title', label: '书名' },
  { id: 'progress', label: '阅读进度' },
]

export function Bookshelf() {
  const { books, status, load, upload, remove } = useBookStore()
  const [settingsOpen, setSettingsOpen] = useState(false)
  const [editing, setEditing] = useState(null)
  const [query, setQuery] = useState('')
  const [sort, setSort] = useState('recent')
  const [activeTag, setActiveTag] = useState(null)

  useEffect(() => {
    if (status === 'idle') load()
  }, [status, load])

  // All unique tags — for the quick-filter row.
  const allTags = useMemo(() => {
    const set = new Set()
    books.forEach((b) => (b.tags || []).forEach((t) => set.add(t)))
    return [...set].sort()
  }, [books])

  const filteredBooks = useMemo(() => {
    let list = books
    if (query.trim()) {
      const q = query.trim().toLowerCase()
      list = list.filter(
        (b) =>
          b.title?.toLowerCase().includes(q) ||
          b.author?.toLowerCase().includes(q) ||
          (b.tags || []).some((t) => t.toLowerCase().includes(q))
      )
    }
    if (activeTag) {
      list = list.filter((b) => (b.tags || []).includes(activeTag))
    }
    return [...list].sort((a, b) => sortBooks(a, b, sort))
  }, [books, query, activeTag, sort])

  const recents = useMemo(
    () =>
      [...books]
        .filter((b) => b.progress?.updatedAt)
        .sort(
          (a, b) =>
            new Date(b.progress.updatedAt) - new Date(a.progress.updatedAt)
        )
        .slice(0, 3),
    [books]
  )

  const isEmpty = books.length === 0 && status === 'ready'

  return (
    <>
      <TopBar variant="shelf" onSettings={() => setSettingsOpen(true)} />

      <main className={styles.page}>
        {isEmpty ? (
          <EmptyShelf onUpload={upload} />
        ) : (
          <div className={styles.shelfLayout}>
            <aside className={styles.rail}>
              <div className={styles.railHeader}>
                <span className={styles.eyebrow}>近期阅读</span>
              </div>
              {recents.length > 0 ? (
                <ul className={styles.railList}>
                  {recents.map((b) => (
                    <RecentCard key={b.id} book={b} />
                  ))}
                </ul>
              ) : (
                <p className={styles.railEmpty}>翻开一本书，这里会留下足迹。</p>
              )}
            </aside>

            <section className={styles.main}>
              <header className={styles.mainHeader}>
                <div>
                  <h1 className={`${styles.pageTitle} ${styles.display}`}>
                    我的书架
                  </h1>
                  <p className={styles.pageSub}>
                    共 <span className={styles.num}>{books.length}</span> 本
                    {(query || activeTag) && (
                      <>
                        {' · '}
                        <span className={styles.num}>{filteredBooks.length}</span> 本命中
                      </>
                    )}
                  </p>
                </div>
                <div className={styles.uploadSlot}>
                  <UploadZone onUpload={upload} variant="compact" />
                </div>
              </header>

              <div className={styles.controls}>
                <div className={styles.searchWrap}>
                  <Search size={14} className={styles.searchIcon} />
                  <input
                    type="text"
                    className={styles.search}
                    placeholder="搜索书名 / 作者 / 标签"
                    value={query}
                    onChange={(e) => setQuery(e.target.value)}
                  />
                  {query && (
                    <button
                      className={styles.searchClear}
                      onClick={() => setQuery('')}
                      title="清除"
                    >
                      ×
                    </button>
                  )}
                </div>
                <div className={styles.sortWrap}>
                  <SlidersHorizontal size={14} className={styles.sortIcon} />
                  <select
                    className={styles.sort}
                    value={sort}
                    onChange={(e) => setSort(e.target.value)}
                  >
                    {SORTS.map((s) => (
                      <option key={s.id} value={s.id}>
                        {s.label}
                      </option>
                    ))}
                  </select>
                </div>
              </div>

              {allTags.length > 0 && (
                <div className={styles.tagBar}>
                  <button
                    className={`${styles.tagPill} ${!activeTag ? styles.tagPillActive : ''}`}
                    onClick={() => setActiveTag(null)}
                  >
                    全部
                  </button>
                  {allTags.map((t) => (
                    <button
                      key={t}
                      className={`${styles.tagPill} ${activeTag === t ? styles.tagPillActive : ''}`}
                      onClick={() => setActiveTag(activeTag === t ? null : t)}
                    >
                      {t}
                    </button>
                  ))}
                </div>
              )}

              {filteredBooks.length === 0 ? (
                <div className={styles.noMatch}>
                  <p>没有找到匹配的书籍。</p>
                  <button
                    className={styles.noMatchReset}
                    onClick={() => {
                      setQuery('')
                      setActiveTag(null)
                    }}
                  >
                    清除筛选
                  </button>
                </div>
              ) : (
                <motion.ul
                  className={styles.grid}
                  initial="hidden"
                  animate="show"
                  variants={{
                    hidden: { transition: { staggerChildren: 0 } },
                    show: {
                      transition: {
                        staggerChildren: 0.04,
                        delayChildren: 0.05,
                      },
                    },
                  }}
                >
                  <AnimatePresence mode="popLayout">
                    {filteredBooks.map((b) => (
                      <motion.li
                        key={b.id}
                        layout
                        className={styles.gridItem}
                        variants={{
                          hidden: { opacity: 0, y: 14 },
                          show: {
                            opacity: 1,
                            y: 0,
                            transition: { type: 'spring', stiffness: 320, damping: 28 },
                          },
                        }}
                        exit={{ opacity: 0, scale: 0.9, transition: { duration: 0.2 } }}
                      >
                        <BookCard book={b} onEdit={setEditing} />
                      </motion.li>
                    ))}
                  </AnimatePresence>
                </motion.ul>
              )}
            </section>
          </div>
        )}
      </main>

      <SettingsPanel open={settingsOpen} onClose={() => setSettingsOpen(false)} />

      {/* Find the live book from store so cover/tag changes flow in without remount. */}
      <BookEditPanel
        book={editing ? books.find((b) => b.id === editing.id) || editing : null}
        onClose={() => setEditing(null)}
      />

      {/* Delete affordance lives inside the edit panel via a separate button below */}
      {editing && (
        <DeleteButton
          book={editing}
          onDelete={async () => {
            if (!window.confirm(`确定删除《${editing.title}》？此操作不可撤销。`)) return
            await remove(editing.id)
            setEditing(null)
          }}
        />
      )}
    </>
  )
}

function DeleteButton({ book, onDelete }) {
  return (
    <button className={styles.deleteFloat} onClick={onDelete} title="删除该书">
      <Trash2 size={14} />
      <span>删除《{book.title.length > 10 ? book.title.slice(0, 10) + '…' : book.title}》</span>
    </button>
  )
}

function EmptyShelf({ onUpload }) {
  return (
    <section className={styles.empty}>
      <div className={styles.emptyArt}>
        <span className={styles.emptyL}>L</span>
      </div>
      <h1 className={styles.emptyTitle}>
        <span className={styles.display}>你的书架静待开张</span>
      </h1>
      <p className={styles.emptyHint}>
        拖入一份 TXT 文件，让一本书在 Lumina 里安家。
      </p>
      <div className={styles.emptyUpload}>
        <UploadZone onUpload={onUpload} variant="full" />
      </div>
    </section>
  )
}

function RecentCard({ book }) {
  const percent = Math.round((book.progress?.percentage || 0) * 100)
  const remainingChars = Math.max(
    0,
    Math.round((book.fileSize || 0) * (1 - (book.progress?.percentage || 0)))
  )
  const remainingMin = estimateMinutes(remainingChars, 600)

  return (
    <li className={styles.recent}>
      <Link to={`/read/${book.id}`} className={styles.recentLink}>
        <div className={styles.recentCover}>
          <BookCover book={book} size="md" />
        </div>
        <div className={styles.recentMeta}>
          <span className={styles.eyebrow}>继续阅读</span>
          <h3 className={styles.recentTitle}>{book.title}</h3>
          <p className={styles.recentChapter}>
            第 {(book.progress?.chapterIdx ?? 0) + 1} 章 · 剩余约{' '}
            {formatMinutes(remainingMin)}
          </p>
          <div className={styles.recentBar}>
            <div className={styles.recentFill} style={{ width: `${percent}%` }} />
          </div>
          <div className={styles.recentFoot}>
            <span>{percent}%</span>
            <ChevronRight size={16} className={styles.recentArrow} />
          </div>
        </div>
      </Link>
    </li>
  )
}

function sortBooks(a, b, mode) {
  switch (mode) {
    case 'title':
      return (a.title || '').localeCompare(b.title || '', 'zh')
    case 'uploaded':
      return new Date(b.createdAt) - new Date(a.createdAt)
    case 'progress':
      return (b.progress?.percentage || 0) - (a.progress?.percentage || 0)
    case 'recent':
    default: {
      const ta = a.progress?.updatedAt || a.updatedAt
      const tb = b.progress?.updatedAt || b.updatedAt
      return new Date(tb) - new Date(ta)
    }
  }
}
