import { useEffect, useRef, useState, useCallback } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { motion, AnimatePresence } from 'motion/react'
import { ChevronLeft, ChevronRight, List, Play, Pause } from 'lucide-react'
import { useReaderStore } from '../../stores/useReaderStore.js'
import { useBookStore } from '../../stores/useBookStore.js'
import { useConfigStore } from '../../stores/useConfigStore.js'
import * as booksApi from '../../api/books.js'
import * as bookmarksApi from '../../api/bookmarks.js'
import { TopBar } from '../../components/TopBar/TopBar.jsx'
import { SettingsPanel } from '../../components/SettingsPanel/SettingsPanel.jsx'
import { BookmarkPanel } from '../../components/BookmarkPanel/BookmarkPanel.jsx'
import { SearchOverlay } from '../../components/SearchOverlay/SearchOverlay.jsx'
import { ShortcutHelp } from '../../components/ShortcutHelp/ShortcutHelp.jsx'
import { throttle, debounce } from '../../utils/throttle.js'
import { estimateMinutes, formatMinutes } from '../../utils/reading-time.js'
import { capturePosition, restorePosition } from '../../utils/position.js'
import { useAutoScroll } from '../../hooks/useAutoScroll.js'
import styles from './Reader.module.css'

const IMMERSIVE_DELAY = 3000

export function Reader() {
  const { bookId } = useParams()
  const navigate = useNavigate()
  const { book, chapters, chapter, progress, status, open, loadChapter, saveProgress, close } =
    useReaderStore()

  const [settingsOpen, setSettingsOpen] = useState(false)
  const [tocOpen, setTocOpen] = useState(false)
  const [bookmarksOpen, setBookmarksOpen] = useState(false)
  const [searchOpen, setSearchOpen] = useState(false)
  const [helpOpen, setHelpOpen] = useState(false)
  const [immersive, setImmersive] = useState(false)
  const [hudText, setHudText] = useState(null)
  const [autoScrolling, setAutoScrolling] = useState(false)

  const autoScrollSpeed = useConfigStore((s) => s.autoScrollSpeed)
  const autoAdvance = useConfigStore((s) => s.autoAdvance)

  const articleRef = useRef(null)
  const epubContentRef = useRef(null)
  const paraRefs = useRef(new Map())
  // Carry an override scroll target across a chapter-load + effect cycle.
  const pendingJumpRef = useRef(null)
  const pendingSearchRef = useRef(null)
  const activeSearchMarkRef = useRef(null)
  const activeSearchFallbackRef = useRef(null)
  const searchCleanupTimerRef = useRef(null)

  const clearTemporarySearchHighlight = useCallback(() => {
    if (searchCleanupTimerRef.current) {
      clearTimeout(searchCleanupTimerRef.current)
      searchCleanupTimerRef.current = null
    }

    if (activeSearchMarkRef.current?.isConnected) {
      unwrapTemporaryMark(activeSearchMarkRef.current)
    }
    activeSearchMarkRef.current = null

    if (activeSearchFallbackRef.current?.classList) {
      activeSearchFallbackRef.current.classList.remove('search-hit-fallback')
    }
    activeSearchFallbackRef.current = null
  }, [])

  const scheduleSearchHighlight = useCallback(
    (query, hitSeq = 0) => {
      requestAnimationFrame(() =>
        requestAnimationFrame(() => {
          clearTemporarySearchHighlight()
          const container = epubContentRef.current
          if (!container || !query) return

          const result = highlightEPUBSearchHit(container, query, hitSeq)
          if (!result) {
            setHudText('未定位到搜索结果')
            setTimeout(() => setHudText(null), 1400)
            return
          }

          activeSearchMarkRef.current = result.mark || null
          activeSearchFallbackRef.current = result.fallback || null
          result.target?.scrollIntoView({ block: 'center', behavior: 'smooth' })

          searchCleanupTimerRef.current = setTimeout(() => {
            clearTemporarySearchHighlight()
          }, 3000)
        }),
      )
    },
    [clearTemporarySearchHighlight],
  )

  // --- Auto-scroll ---
  useAutoScroll({
    enabled: autoScrolling,
    speed: autoScrollSpeed,
    onReachEnd: () => {
      if (autoAdvance && chapter?.nextIdx != null) {
        // Load next chapter and keep scrolling. The chapter-change effect
        // will reset scroll to top; the hook will resume automatically on
        // the next render cycle because `enabled` stays true.
        loadChapter(chapter.nextIdx)
      } else {
        setAutoScrolling(false)
        setHudText('已到末尾')
        setTimeout(() => setHudText(null), 1400)
      }
    },
    onInterrupt: () => {
      if (autoScrolling) {
        setAutoScrolling(false)
        setHudText('自动滑动已暂停')
        setTimeout(() => setHudText(null), 1200)
      }
    },
  })

  // Stop auto-scroll when opening overlays (user isn't watching the text).
  useEffect(() => {
    if (settingsOpen || tocOpen || bookmarksOpen || searchOpen) setAutoScrolling(false)
  }, [settingsOpen, tocOpen, bookmarksOpen, searchOpen])

  function toggleAutoScroll() {
    setAutoScrolling((v) => {
      const next = !v
      setHudText(next ? `自动滑动 · ${autoScrollSpeed} px/s` : '自动滑动已暂停')
      setTimeout(() => setHudText(null), 1200)
      return next
    })
  }

  // --- Boot: fetch book + chapters + initial chapter ---
  useEffect(() => {
    const id = Number(bookId)
    if (!id) return
    ;(async () => {
      try {
        const b = await booksApi.getBook(id)
        await open(b)
        // Also refresh shelf so the "recent" list stays in sync.
        useBookStore.getState().load()
      } catch {
        navigate('/', { replace: true })
      }
    })()
    return () => {
      clearTemporarySearchHighlight()
      close()
    }
  }, [bookId]) // eslint-disable-line react-hooks/exhaustive-deps

  // --- Restore scroll offset when chapter changes ---
  // Runs once per chapter load. Reads progress via getState() so progress
  // updates (from the scroll listener) don't re-trigger this effect.
  useEffect(() => {
    if (!chapter) return
    clearTemporarySearchHighlight()

    // Decide the target offset:
    //   1. Pending scroll-pct jump (from progress-bar scrub) wins.
    //   2. Pending bookmark jump (same chapter).
    //   3. Else saved progress for this chapter, if any.
    //   4. Else scroll to top.
    const pending = pendingJumpRef.current
    let targetPosition = { chapterIdx: chapter.chapterIdx, charOffset: 0 }
    if (pending && pending.chapterIdx === chapter.chapterIdx) {
      pendingJumpRef.current = null
      targetPosition = pending
    } else {
      const p = useReaderStore.getState().progress
      if (p && p.chapterIdx === chapter.chapterIdx) {
        targetPosition = p
      }
    }

    const doScroll = () => {
      restorePosition(chapter, targetPosition, paraRefs, epubContentRef)
      const pendingSearch = pendingSearchRef.current
      if (chapter.format === 'epub' && pendingSearch && pendingSearch.chapterIdx === chapter.chapterIdx) {
        pendingSearchRef.current = null
        scheduleSearchHighlight(pendingSearch.query, pendingSearch.hitSeq ?? 0)
      }
    }
    let cancelled = false
    const run = () => {
      if (cancelled) return
      requestAnimationFrame(() => requestAnimationFrame(doScroll))
    }
    if (document.fonts?.ready) {
      document.fonts.ready.then(run)
    } else {
      run()
    }

    // Brief chapter HUD (only on chapter change — not on progress ticks).
    setHudText(chapter.title)
    const t = setTimeout(() => setHudText(null), 1600)
    return () => {
      cancelled = true
      clearTimeout(t)
    }
  }, [chapter, clearTemporarySearchHighlight, scheduleSearchHighlight]) // intentionally omit `progress`

  // --- Progress sync: find topmost visible paragraph, save throttled+debounced ---
  const computeAndSave = useCallback(() => {
    if (!chapter) return
    const position = capturePosition(chapter, paraRefs, epubContentRef)
    const vh = window.innerHeight || 1
    const scrollPct = Math.max(0, Math.min(1, (window.scrollY + vh) / document.documentElement.scrollHeight))
    const percentage =
      chapters.length > 0
        ? (chapter.chapterIdx + scrollPct) / chapters.length
        : scrollPct
    saveProgress({
      chapterIdx: chapter.chapterIdx,
      charOffset: position?.charOffset || 0,
      anchor: position?.anchor,
      scrollPct: position?.scrollPct,
      percentage: Math.max(0, Math.min(1, percentage)),
    })
  }, [chapter, chapters.length, saveProgress])

  useEffect(() => {
    if (!chapter) return
    const throttled = throttle(computeAndSave, 1500)
    const debounced = debounce(computeAndSave, 600)
    const handler = () => {
      throttled()
      debounced()
    }
    window.addEventListener('scroll', handler, { passive: true })
    return () => window.removeEventListener('scroll', handler)
  }, [chapter, computeAndSave])

  useEffect(() => {
    if (chapter?.format !== 'epub' || !epubContentRef.current) return
    const container = epubContentRef.current

    const onClick = (e) => {
      const target = e.target instanceof Element ? e.target : null
      const link = target?.closest('a[href]')
      if (!link || !container.contains(link)) return

      const href = link.getAttribute('href')?.trim()
      if (!href) return

      if (href.startsWith('#/chapter/')) {
        const destination = parseEPUBChapterHref(href)
        if (!destination) return

        e.preventDefault()
        setAutoScrolling(false)
        pendingSearchRef.current = null

        const nextPosition = destination.anchor
          ? { chapterIdx: destination.chapterIdx, anchor: destination.anchor }
          : { chapterIdx: destination.chapterIdx, scrollPct: 0 }
        pendingJumpRef.current = nextPosition

        if (chapter.chapterIdx !== destination.chapterIdx) {
          loadChapter(destination.chapterIdx)
        } else {
          pendingJumpRef.current = null
          restorePosition(chapter, nextPosition, paraRefs, epubContentRef)
        }
        return
      }

      if (href.startsWith('#')) {
        const anchor = href.slice(1)
        if (!anchor) return
        e.preventDefault()
        setAutoScrolling(false)
        pendingSearchRef.current = null
        restorePosition(
          chapter,
          { chapterIdx: chapter.chapterIdx, anchor },
          paraRefs,
          epubContentRef,
        )
      }
    }

    container.addEventListener('click', onClick)
    return () => container.removeEventListener('click', onClick)
  }, [chapter, loadChapter])

  // --- Immersive mode ---
  useEffect(() => {
    let timer = null
    const showUI = () => {
      setImmersive(false)
      if (timer) clearTimeout(timer)
      timer = setTimeout(() => setImmersive(true), IMMERSIVE_DELAY)
    }
    showUI()
    window.addEventListener('mousemove', showUI)
    window.addEventListener('scroll', showUI, { passive: true })
    window.addEventListener('keydown', showUI)
    window.addEventListener('touchstart', showUI, { passive: true })
    return () => {
      if (timer) clearTimeout(timer)
      window.removeEventListener('mousemove', showUI)
      window.removeEventListener('scroll', showUI)
      window.removeEventListener('keydown', showUI)
      window.removeEventListener('touchstart', showUI)
    }
  }, [])

  // --- Keyboard ---
  useEffect(() => {
    const onKey = (e) => {
      if (e.target.matches('input, textarea, [contenteditable]')) return
      if (e.key === 'ArrowLeft') {
        if (chapter?.prevIdx != null) loadChapter(chapter.prevIdx)
      } else if (e.key === 'ArrowRight') {
        if (chapter?.nextIdx != null) loadChapter(chapter.nextIdx)
      } else if (e.key.toLowerCase() === 't') {
        setTocOpen((v) => !v)
      } else if (e.key.toLowerCase() === 'b') {
        handleAddBookmark()
      } else if (e.key.toLowerCase() === 'l') {
        setBookmarksOpen((v) => !v)
      } else if (e.key.toLowerCase() === 'p') {
        toggleAutoScroll()
      } else if (e.key === '[') {
        // Slow down auto-scroll.
        const cfg = useConfigStore.getState()
        cfg.patch({ autoScrollSpeed: Math.max(10, Math.round(cfg.autoScrollSpeed - 5)) })
        if (autoScrolling) {
          setHudText(`速度 ${useConfigStore.getState().autoScrollSpeed} px/s`)
          setTimeout(() => setHudText(null), 900)
        }
      } else if (e.key === ']') {
        const cfg = useConfigStore.getState()
        cfg.patch({ autoScrollSpeed: Math.min(240, Math.round(cfg.autoScrollSpeed + 5)) })
        if (autoScrolling) {
          setHudText(`速度 ${useConfigStore.getState().autoScrollSpeed} px/s`)
          setTimeout(() => setHudText(null), 900)
        }
      } else if (e.key === '/') {
        e.preventDefault()
        setSearchOpen(true)
      } else if (e.key === '?') {
        e.preventDefault()
        setHelpOpen(true)
      } else if (e.key === ',') {
        setSettingsOpen((v) => !v)
      } else if (e.key === 'Escape') {
        setSettingsOpen(false)
        setTocOpen(false)
        setBookmarksOpen(false)
        setSearchOpen(false)
        setHelpOpen(false)
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [chapter, loadChapter]) // eslint-disable-line react-hooks/exhaustive-deps

  async function handleAddBookmark() {
    if (!chapter || !book) return
    try {
      const position = capturePosition(chapter, paraRefs, epubContentRef)
      const bookmark = await bookmarksApi.createBookmark(book.id, {
        chapterIdx: chapter.chapterIdx,
        charOffset: position?.charOffset || 0,
        anchor: position?.anchor,
        scrollPct: position?.scrollPct,
        note: '',
      })
      setHudText('已添加书签')
      setTimeout(() => setHudText(null), 1400)
      return bookmark
    } catch (err) {
      setHudText(err.message || '书签创建失败')
      setTimeout(() => setHudText(null), 2000)
      return null
    }
  }

  function handleJumpBookmark(targetChapterIdx, targetCharOffset) {
    if (typeof targetChapterIdx === 'object') {
      targetCharOffset = {
        chapterIdx: targetChapterIdx.chapterIdx,
        charOffset: targetChapterIdx.charOffset || 0,
        anchor: targetChapterIdx.anchor,
        scrollPct: targetChapterIdx.scrollPct,
      }
    }
    setBookmarksOpen(false)
    // Carry the target offset via a one-shot ref so the next chapter-load
    // effect can snap the scroll there (overrides saved progress).
    const target =
      typeof targetCharOffset === 'object'
        ? targetCharOffset
        : { chapterIdx: targetChapterIdx, charOffset: targetCharOffset }
    pendingJumpRef.current = target
    if (chapter?.chapterIdx !== target.chapterIdx) {
      loadChapter(target.chapterIdx)
    } else {
      // Same chapter — scroll now.
      pendingJumpRef.current = null
      restorePosition(chapter, target, paraRefs, epubContentRef)
    }
  }

  function gotoChapter(idx) {
    loadChapter(idx)
    setTocOpen(false)
  }

  function handleSearchJump(hit, query) {
    if (!hit) return

    setSearchOpen(false)
    setAutoScrolling(false)

    const isEPUBHit = hit.hitSeq != null
    const target = isEPUBHit
      ? { chapterIdx: hit.chapterIdx, scrollPct: 0 }
      : { chapterIdx: hit.chapterIdx, charOffset: hit.charOffset || 0 }

    pendingJumpRef.current = target
    pendingSearchRef.current =
      isEPUBHit && query
        ? { chapterIdx: hit.chapterIdx, hitSeq: hit.hitSeq, query }
        : null

    if (chapter?.chapterIdx !== hit.chapterIdx) {
      loadChapter(hit.chapterIdx)
      return
    }

    pendingJumpRef.current = null
    restorePosition(chapter, target, paraRefs, epubContentRef)

    if (pendingSearchRef.current && chapter?.format === 'epub') {
      const pendingSearch = pendingSearchRef.current
      pendingSearchRef.current = null
      scheduleSearchHighlight(pendingSearch.query, pendingSearch.hitSeq ?? 0)
    }
  }

  if (status === 'loading' || !chapter) {
    return (
      <div className={styles.loading}>
        <span className={styles.loadingDot} />
        <span className={styles.loadingDot} />
        <span className={styles.loadingDot} />
      </div>
    )
  }

  const remainingMin = estimateMinutes(chapter.charCount, 420)
  const percentFull = Math.round((progress?.percentage || 0) * 100)

  return (
    <>
      <TopBar
        variant="reader"
        bookTitle={book?.title}
        chapterTitle={chapter.title}
        onBookmark={() => setBookmarksOpen(true)}
        onSearch={() => setSearchOpen(true)}
        onSettings={() => setSettingsOpen(true)}
        immersiveHidden={immersive}
      />

      {/* Morph target for the bookshelf card title. Visually empty — we only
          use it as a view-transition anchor so the card title flies toward
          the top bar instead of popping. */}
      {book?.id != null && (
        <span
          aria-hidden="true"
          style={{
            viewTransitionName: `book-title-${book.id}`,
            position: 'fixed',
            top: 32,
            left: '50%',
            transform: 'translateX(-50%)',
            width: 1,
            height: 1,
            opacity: 0,
            pointerEvents: 'none',
          }}
        />
      )}

      <button
        className={`${styles.tocTrigger} glass ${immersive ? styles.immersiveHidden : ''}`}
        onClick={() => setTocOpen(true)}
        title="目录 (T)"
      >
        <List size={18} />
      </button>

      <article ref={articleRef} className={styles.article}>
        <header className={styles.chapterHead}>
          <span className={styles.chapterEyebrow}>
            CHAPTER · {String(chapter.chapterIdx + 1).padStart(2, '0')}
          </span>
          <h1 className={styles.chapterTitle}>{chapter.title}</h1>
          <div className={styles.rule} />
        </header>

        <div className={styles.body}>
          {chapter.format === 'epub' ? (
            <>
              {chapter.css && <style data-epub-scope={`ch-${chapter.chapterIdx}`}>{chapter.css}</style>}
              <div
                ref={epubContentRef}
                className={styles.epubContent}
                dangerouslySetInnerHTML={{ __html: chapter.html }}
              />
            </>
          ) : (
            chapter.paragraphs.map((p, i) => (
              <p
                key={i}
                ref={(el) => {
                  if (el) paraRefs.current.set(i, el)
                  else paraRefs.current.delete(i)
                }}
                data-para-idx={i}
                className={styles.paragraph}
              >
                {p}
              </p>
            ))
          )}
        </div>

        <nav className={styles.chapterNav}>
          <button
            disabled={chapter.prevIdx == null}
            onClick={() => chapter.prevIdx != null && loadChapter(chapter.prevIdx)}
            className={styles.navBtn}
          >
            <ChevronLeft size={16} /> 上一章
          </button>
          <button
            disabled={chapter.nextIdx == null}
            onClick={() => chapter.nextIdx != null && loadChapter(chapter.nextIdx)}
            className={styles.navBtn}
          >
            下一章 <ChevronRight size={16} />
          </button>
        </nav>
      </article>

      {/* Right-bottom reading meta + auto-scroll control */}
      <div className={`${styles.readingMeta} ${immersive && !autoScrolling ? styles.immersiveHidden : ''}`}>
        <button
          className={`${styles.autoBtn} ${autoScrolling ? styles.autoBtnActive : ''}`}
          onClick={toggleAutoScroll}
          title={autoScrolling ? '暂停自动滑动 (P)' : '开始自动滑动 (P)'}
        >
          {autoScrolling ? <Pause size={13} /> : <Play size={13} />}
          <span className={styles.autoBtnLabel}>
            {autoScrolling ? `${autoScrollSpeed} px/s` : '自动滑动'}
          </span>
        </button>
        <span className={styles.metaDot} />
        <span>
          第 <span className={styles.metaNum}>{chapter.chapterIdx + 1}</span> /{' '}
          {chapters.length} 章
        </span>
        <span className={styles.metaDot} />
        <span>剩余 {formatMinutes(remainingMin)}</span>
      </div>

      {/* Bottom progress bar — click/tap to jump, hover to preview chapter */}
      <ProgressBar
        percent={percentFull / 100}
        chapters={chapters}
        hidden={immersive}
        onJump={(targetPct) => {
          const n = chapters.length
          if (n === 0) return
          const idx = Math.min(n - 1, Math.floor(targetPct * n))
          const pctInChapter = targetPct * n - idx
          // Use a new marker on the jump ref so the restore effect knows
          // to interpret it as a percentage rather than a char offset.
          pendingJumpRef.current = {
            chapterIdx: idx,
            scrollPct: pctInChapter,
          }
          if (chapter?.chapterIdx !== idx) {
            loadChapter(idx)
          } else {
            pendingJumpRef.current = null
            scrollToPercentInChapter(pctInChapter)
          }
        }}
      />

      {/* Chapter HUD — short transient notice */}
      {hudText && (
        <div className={`glass ${styles.hud}`} role="status">
          {hudText}
        </div>
      )}

      {/* TOC drawer */}
      <AnimatePresence>
        {tocOpen && (
          <TOCDrawer
            chapters={chapters}
            currentIdx={chapter.chapterIdx}
            onPick={gotoChapter}
            onClose={() => setTocOpen(false)}
          />
        )}
      </AnimatePresence>

      <SettingsPanel open={settingsOpen} onClose={() => setSettingsOpen(false)} />

      <BookmarkPanel
        open={bookmarksOpen}
        bookId={book?.id}
        chapters={chapters}
        currentChapterIdx={chapter.chapterIdx}
        currentCharOffset={progress?.charOffset || 0}
        onAddCurrent={handleAddBookmark}
        onJump={handleJumpBookmark}
        onClose={() => setBookmarksOpen(false)}
      />

      <SearchOverlay
        open={searchOpen}
        bookId={book?.id}
        onJump={handleSearchJump}
        onClose={() => setSearchOpen(false)}
      />

      <ShortcutHelp open={helpOpen} onClose={() => setHelpOpen(false)} />
    </>
  )
}

function TOCDrawer({ chapters, currentIdx, onPick, onClose }) {
  useEffect(() => {
    const onKey = (e) => {
      if (e.key === 'Escape') onClose()
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [onClose])

  return (
    <>
      <motion.div
        className={styles.tocBackdrop}
        onClick={onClose}
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        exit={{ opacity: 0 }}
        transition={{ duration: 0.2 }}
      />
      <motion.aside
        className={`glass-strong ${styles.tocDrawer}`}
        initial={{ x: '-105%' }}
        animate={{ x: 0 }}
        exit={{ x: '-105%' }}
        transition={{ type: 'spring', stiffness: 320, damping: 34 }}
        drag="x"
        dragConstraints={{ left: -380, right: 0 }}
        dragElastic={0.08}
        dragMomentum={false}
        onDragEnd={(_, info) => {
          if (info.offset.x < -100 || info.velocity.x < -500) onClose()
        }}
      >
        <div className={styles.tocDragHandle} aria-hidden="true" />
        <header className={styles.tocHeader}>
          <span>目录</span>
          <span className={styles.tocCount}>{chapters.length} 章</span>
        </header>
        <ul className={styles.tocList}>
          {chapters.map((c) => (
            <li key={c.chapterIdx}>
              <button
                className={`${styles.tocItem} ${c.chapterIdx === currentIdx ? styles.tocItemActive : ''}`}
                onClick={() => onPick(c.chapterIdx)}
              >
                <span className={styles.tocNum}>
                  {String(c.chapterIdx + 1).padStart(3, '0')}
                </span>
                <span className={styles.tocTitle}>{c.title}</span>
              </button>
            </li>
          ))}
        </ul>
      </motion.aside>
    </>
  )
}

/** Scroll the viewport to a proportion (0..1) of the document height.
 *  Used by the progress-bar scrub. */
function scrollToPercentInChapter(pct) {
  const max = Math.max(0, document.documentElement.scrollHeight - window.innerHeight)
  window.scrollTo({ top: max * Math.max(0, Math.min(1, pct)), behavior: 'instant' })
}

function parseEPUBChapterHref(href) {
  const match = href.match(/^#\/chapter\/(\d+)(?:#(.+))?$/)
  if (!match) return null

  const chapterIdx = Number(match[1])
  if (!Number.isInteger(chapterIdx)) return null

  let anchor = null
  if (match[2]) {
    try {
      anchor = decodeURIComponent(match[2])
    } catch {
      anchor = match[2]
    }
  }

  return { chapterIdx, anchor }
}

function highlightEPUBSearchHit(container, query, hitSeq = 0) {
  if (!container || !query) return null

  const walker = document.createTreeWalker(container, NodeFilter.SHOW_TEXT, {
    acceptNode(node) {
      return node.textContent?.trim()
        ? NodeFilter.FILTER_ACCEPT
        : NodeFilter.FILTER_REJECT
    },
  })

  const nodes = []
  let text = ''
  let current = walker.nextNode()
  while (current) {
    const value = current.textContent || ''
    nodes.push({ node: current, start: text.length, text: value })
    text += value
    current = walker.nextNode()
  }

  if (!text) return null

  const matchStart = findNthMatchIndex(text, query, hitSeq)
  if (matchStart < 0) return null

  const matchEnd = matchStart + query.length
  const startPos = locateTextOffset(nodes, matchStart, true)
  const endPos = locateTextOffset(nodes, matchEnd, false)
  if (!startPos || !endPos) return null

  if (startPos.node === endPos.node) {
    const matchNode = startPos.node.splitText(startPos.offset)
    matchNode.splitText(matchEnd - matchStart)
    const mark = document.createElement('mark')
    mark.className = 'search-hit-temp'
    matchNode.parentNode?.replaceChild(mark, matchNode)
    mark.appendChild(matchNode)
    return { mark, target: mark }
  }

  const range = document.createRange()
  range.setStart(startPos.node, startPos.offset)
  range.setEnd(endPos.node, endPos.offset)

  try {
    const mark = document.createElement('mark')
    mark.className = 'search-hit-temp'
    range.surroundContents(mark)
    return { mark, target: mark }
  } catch {
    const fallback = startPos.node.parentElement || container
    fallback.classList.add('search-hit-fallback')
    return { fallback, target: fallback }
  }
}

function findNthMatchIndex(text, query, hitSeq) {
  const lowerText = text.toLowerCase()
  const lowerQuery = query.toLowerCase()
  if (!lowerQuery) return -1

  let from = 0
  let count = 0
  while (from <= lowerText.length) {
    const idx = lowerText.indexOf(lowerQuery, from)
    if (idx < 0) return -1
    if (count === hitSeq) return idx
    count += 1
    from = idx + Math.max(1, lowerQuery.length)
  }
  return -1
}

function locateTextOffset(nodes, absoluteOffset, preferNext) {
  for (const entry of nodes) {
    const end = entry.start + entry.text.length
    if (absoluteOffset < end) {
      return { node: entry.node, offset: absoluteOffset - entry.start }
    }
    if (!preferNext && absoluteOffset === end) {
      return { node: entry.node, offset: entry.text.length }
    }
  }

  const last = nodes[nodes.length - 1]
  if (!last) return null
  return { node: last.node, offset: last.text.length }
}

function unwrapTemporaryMark(mark) {
  if (!mark?.parentNode) return
  const parent = mark.parentNode
  while (mark.firstChild) {
    parent.insertBefore(mark.firstChild, mark)
  }
  parent.removeChild(mark)
  parent.normalize()
}

/** Bottom progress bar. Click/tap anywhere to jump to that proportion of the
 *  total reading flow; hover shows the chapter title at that position. */
function ProgressBar({ percent, chapters, hidden, onJump }) {
  const [hoverPct, setHoverPct] = useState(null)
  const [hoverX, setHoverX] = useState(0)

  function handleMove(e) {
    const touch = e.touches?.[0]
    const clientX = touch ? touch.clientX : e.clientX
    const rect = e.currentTarget.getBoundingClientRect()
    const pct = Math.max(0, Math.min(1, (clientX - rect.left) / rect.width))
    setHoverPct(pct)
    setHoverX(clientX - rect.left)
  }

  function handleLeave() {
    setHoverPct(null)
  }

  function handleClick(e) {
    const touch = e.changedTouches?.[0]
    const clientX = touch ? touch.clientX : e.clientX
    const rect = e.currentTarget.getBoundingClientRect()
    const pct = Math.max(0, Math.min(1, (clientX - rect.left) / rect.width))
    onJump(pct)
  }

  // Chapter title at the hovered position, purely for preview.
  const hoverChapter =
    hoverPct != null && chapters.length > 0
      ? chapters[Math.min(chapters.length - 1, Math.floor(hoverPct * chapters.length))]
      : null

  return (
    <div
      className={`${styles.bottomBar} ${hidden ? styles.immersiveHidden : ''}`}
      onPointerMove={handleMove}
      onPointerLeave={handleLeave}
      onClick={handleClick}
    >
      <div className={styles.bottomHit} aria-hidden="true" />
      <div className={styles.bottomFill} style={{ width: `${percent * 100}%` }} />
      {hoverPct != null && (
        <>
          <div className={styles.bottomMarker} style={{ left: hoverX }} />
          {hoverChapter && (
            <div
              className={`glass ${styles.bottomTooltip}`}
              style={{ left: hoverX }}
            >
              <span className={styles.tooltipNum}>
                {String(hoverChapter.chapterIdx + 1).padStart(2, '0')}
              </span>
              <span className={styles.tooltipTitle}>{hoverChapter.title}</span>
              <span className={styles.tooltipPct}>
                {Math.round(hoverPct * 100)}%
              </span>
            </div>
          )}
        </>
      )}
    </div>
  )
}
