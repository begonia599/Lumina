import { useRef } from 'react'
import { MoreHorizontal } from 'lucide-react'
import { BookCover } from '../BookCover/BookCover.jsx'
import { useViewTransitionNav, assignTransitionName } from '../../utils/viewTransition.js'
import styles from './BookCard.module.css'

export function BookCard({ book, onEdit }) {
  const percent = Math.round((book.progress?.percentage || 0) * 100)
  const coverRef = useRef(null)
  const titleRef = useRef(null)
  const go = useViewTransitionNav()

  function handleNavigate(e) {
    e.preventDefault()
    assignTransitionName(coverRef.current, `book-cover-${book.id}`)
    assignTransitionName(titleRef.current, `book-title-${book.id}`)
    go(`/read/${book.id}`)
  }

  return (
    <article className={styles.card}>
      <a href={`/read/${book.id}`} className={styles.link} onClick={handleNavigate}>
        <div className={styles.coverWrap}>
          <div ref={coverRef} className={styles.coverInner}>
            <BookCover book={book} size="md" />
          </div>
          {book.tags?.length > 0 && (
            <div className={styles.tagOverlay}>
              {book.tags.slice(0, 2).map((t) => (
                <span key={t} className={styles.tagChip}>
                  {t}
                </span>
              ))}
            </div>
          )}
        </div>
        <div className={styles.meta}>
          <h3 ref={titleRef} className={styles.title}>
            {book.title}
          </h3>
          {book.author && <p className={styles.author}>{book.author}</p>}
          <div className={styles.sub}>
            <span>{book.chapterCount} 章</span>
            {book.progress && (
              <>
                <span className={styles.dot} />
                <span>{percent}%</span>
              </>
            )}
          </div>
          {percent > 0 && (
            <div className={styles.progress}>
              <div className={styles.progressFill} style={{ width: `${percent}%` }} />
            </div>
          )}
        </div>
      </a>
      <button
        type="button"
        className={styles.menu}
        title="编辑"
        onClick={(e) => {
          e.preventDefault()
          onEdit?.(book)
        }}
      >
        <MoreHorizontal size={16} />
      </button>
    </article>
  )
}
