import { api } from './client.js'

export const listBookmarks = (bookId) => api.get(`/books/${bookId}/bookmarks`)

export const createBookmark = (bookId, { chapterIdx, charOffset, anchor, scrollPct, note }) =>
  api.post(`/books/${bookId}/bookmarks`, { chapterIdx, charOffset, anchor, scrollPct, note })

export const updateBookmark = (id, { note }) =>
  api.patch(`/bookmarks/${id}`, { note })

export const deleteBookmark = (id) => api.del(`/bookmarks/${id}`)
