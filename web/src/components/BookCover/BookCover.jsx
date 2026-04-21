import { coverFromTitle } from '../../utils/color-generator.js'
import { coverUrl } from '../../api/books.js'
import styles from './BookCover.module.css'

/**
 * Procedural or uploaded cover art.
 * - If `book.hasCover`, render an <img> from /api/books/:id/cover.
 * - Otherwise generate a deterministic gradient + shape + watermark from
 *   the title via coverFromTitle().
 *
 * `book` is optional — call as <BookCover title="..."> for procedural-only.
 */
export function BookCover({ book, title, size = 'md' }) {
  const effectiveTitle = title ?? book?.title ?? 'Untitled'

  if (book?.hasCover) {
    return (
      <div className={`${styles.cover} ${styles[size]} ${styles.uploaded}`}>
        <img
          className={styles.uploadedImg}
          src={coverUrl(book.id, book.coverBust)}
          alt={`《${effectiveTitle}》封面`}
          draggable="false"
        />
      </div>
    )
  }

  const { gradient, shape, hue1, hue2 } = coverFromTitle(effectiveTitle)
  return (
    <div
      className={`${styles.cover} ${styles[size]}`}
      style={{ background: gradient }}
      role="img"
      aria-label={`《${effectiveTitle}》封面`}
    >
      <Shape shape={shape} hue={hue2} />
      <div className={styles.grain} aria-hidden="true" />
      <div className={styles.vignette} aria-hidden="true" />
      <div className={styles.watermark}>
        <span className={styles.watermarkTitle}>{effectiveTitle}</span>
        <span
          className={styles.watermarkRule}
          style={{ background: `hsl(${hue1}, 80%, 85%)` }}
        />
        <span className={styles.watermarkBrand}>LUMINA</span>
      </div>
    </div>
  )
}

function Shape({ shape, hue }) {
  const stroke = `hsla(${hue}, 80%, 90%, 0.32)`
  const fill = `hsla(${hue}, 80%, 90%, 0.12)`
  const common = { stroke, fill, strokeWidth: 1.2 }
  return (
    <svg className={styles.shape} viewBox="0 0 100 100" aria-hidden="true">
      {shape === 'disc' && <circle cx="72" cy="30" r="22" {...common} />}
      {shape === 'slash' && (
        <line x1="10" y1="90" x2="90" y2="14" stroke={stroke} strokeWidth="1.4" />
      )}
      {shape === 'dots' && (
        <g {...common} fill={stroke}>
          {Array.from({ length: 6 }).map((_, i) =>
            Array.from({ length: 4 }).map((_, j) => (
              <circle key={`${i}-${j}`} cx={20 + i * 12} cy={18 + j * 10} r="1.3" />
            ))
          )}
        </g>
      )}
      {shape === 'arc' && (
        <path d="M 14 86 Q 50 12 90 50" fill="none" stroke={stroke} strokeWidth="1.3" />
      )}
      {shape === 'triangle' && <polygon points="78,16 92,50 64,50" {...common} />}
    </svg>
  )
}
