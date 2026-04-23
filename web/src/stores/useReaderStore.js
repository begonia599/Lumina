import { create } from 'zustand'
import * as chaptersApi from '../api/chapters.js'
import * as progressApi from '../api/progress.js'

export const useReaderStore = create((set, get) => ({
  bookId: null,
  book: null,
  chapters: [],
  chapter: null, // { chapterIdx, title, paragraphs, charCount, prevIdx, nextIdx }
  progress: null, // { chapterIdx, charOffset, anchor?, scrollPct?, percentage }
  status: 'idle',
  error: null,

  async open(book, initialChapterIdx = null) {
    set({
      bookId: book.id,
      book,
      chapter: null,
      // Seed from book.progress if the book was fetched from /books (the
      // bookshelf listing includes progress via LEFT JOIN). Single-book fetch
      // (/books/:id) does NOT include it, so we always re-fetch below.
      progress: book.progress || null,
      status: 'loading',
      error: null,
    })
    try {
      // Parallel fetch: chapter list + authoritative progress.
      const [chapters, progress] = await Promise.all([
        chaptersApi.listChapters(book.id),
        // GET /progress returns 404 when no row exists yet — treat as null.
        progressApi.getProgress(book.id).catch(() => null),
      ])
      set({
        chapters: chapters || [],
        progress: progress || null,
      })

      // Pick initial chapter using the freshly-fetched progress, not the
      // (potentially stale / undefined) book.progress.
      const startIdx =
        initialChapterIdx ??
        progress?.chapterIdx ??
        chapters?.[0]?.chapterIdx ??
        0
      await get().loadChapter(startIdx)
      set({ status: 'ready' })
    } catch (err) {
      set({ status: 'error', error: err.message })
    }
  },

  async loadChapter(chapterIdx) {
    const bookId = get().bookId
    if (bookId == null) return
    const chapter = await chaptersApi.getChapter(bookId, chapterIdx)
    set({ chapter })
    return chapter
  },

  async saveProgress({ chapterIdx, charOffset, anchor, scrollPct, percentage }) {
    const bookId = get().bookId
    if (bookId == null) return
    set({ progress: { chapterIdx, charOffset, anchor, scrollPct, percentage } })
    try {
      await progressApi.updateProgress(bookId, { chapterIdx, charOffset, anchor, scrollPct, percentage })
    } catch {
      /* silent — retried on next save */
    }
  },

  close() {
    set({
      bookId: null,
      book: null,
      chapters: [],
      chapter: null,
      progress: null,
      status: 'idle',
      error: null,
    })
  },
}))
