import { api } from './client.js'

export const getProgress = (bookId) => api.get(`/books/${bookId}/progress`)

export const updateProgress = (bookId, { chapterIdx, charOffset, percentage }) =>
  api.put(`/books/${bookId}/progress`, { chapterIdx, charOffset, percentage })
