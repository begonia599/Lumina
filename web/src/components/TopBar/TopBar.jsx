import { useNavigate } from 'react-router-dom'
import { LogOut, Settings, ChevronLeft, BookmarkPlus, Search } from 'lucide-react'
import styles from './TopBar.module.css'
import { useAuthStore } from '../../stores/useAuthStore.js'

/**
 * Top navigation bar — shared shell across pages.
 *
 * Props:
 *   variant: 'shelf' | 'reader'
 *   bookTitle, chapterTitle (reader only)
 *   onSettings, onBookmark (callbacks)
 *   immersiveHidden: boolean (reader only — fades the bar)
 */
export function TopBar({
  variant = 'shelf',
  bookTitle,
  chapterTitle,
  onSettings,
  onBookmark,
  onSearch,
  immersiveHidden = false,
}) {
  const navigate = useNavigate()
  const user = useAuthStore((s) => s.user)
  const logout = useAuthStore((s) => s.logout)

  async function handleLogout() {
    await logout()
    navigate('/auth', { replace: true })
  }

  return (
    <nav
      className={`glass ${styles.bar} ${immersiveHidden ? styles.hidden : ''}`}
      data-variant={variant}
    >
      <div className={styles.left}>
        {variant === 'reader' ? (
          <>
            <button
              className={styles.iconBtn}
              onClick={() => navigate('/')}
              title="返回书架"
            >
              <ChevronLeft size={18} />
            </button>
            <div className={styles.bookInfo}>
              <span className={styles.eyebrow}>LUMINA</span>
              <span className={styles.bookTitle}>{bookTitle}</span>
              {chapterTitle && (
                <>
                  <span className={styles.sep}>·</span>
                  <span className={styles.chapterTitle}>{chapterTitle}</span>
                </>
              )}
            </div>
          </>
        ) : (
          <div className={styles.brand}>
            <span className={styles.brandMark}>L</span>
            <span className={styles.brandName}>Lumina</span>
            <span className={styles.brandTag}>沉浸阅读</span>
          </div>
        )}
      </div>

      <div className={styles.right}>
        {variant === 'reader' && (
          <>
            {onSearch && (
              <button className={styles.iconBtn} onClick={onSearch} title="搜索">
                <Search size={18} />
              </button>
            )}
            {onBookmark && (
              <button className={styles.iconBtn} onClick={onBookmark} title="书签">
                <BookmarkPlus size={18} />
              </button>
            )}
          </>
        )}
        {onSettings && (
          <button className={styles.iconBtn} onClick={onSettings} title="设置">
            <Settings size={18} />
          </button>
        )}
        {variant === 'shelf' && (
          <>
            {user && <span className={styles.user}>{user.username}</span>}
            <button className={styles.iconBtn} onClick={handleLogout} title="登出">
              <LogOut size={18} />
            </button>
          </>
        )}
      </div>
    </nav>
  )
}
