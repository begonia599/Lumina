import { api } from './client.js'

export const getProgress = (bookId) => api.get(`/books/${bookId}/progress`)

export const updateProgress = (bookId, { chapterIdx, charOffset, anchor, scrollPct, percentage }) =>
  api.put(`/books/${bookId}/progress`, { chapterIdx, charOffset, anchor, scrollPct, percentage })
