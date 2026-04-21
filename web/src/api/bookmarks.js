import { api } from './client.js'

export const listBookmarks = (bookId) => api.get(`/books/${bookId}/bookmarks`)

export const createBookmark = (bookId, { chapterIdx, charOffset, note }) =>
  api.post(`/books/${bookId}/bookmarks`, { chapterIdx, charOffset, note })

export const updateBookmark = (id, { note }) =>
  api.patch(`/bookmarks/${id}`, { note })

export const deleteBookmark = (id) => api.del(`/bookmarks/${id}`)
