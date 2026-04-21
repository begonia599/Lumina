import { api } from './client.js'

export const listChapters = (bookId) => api.get(`/books/${bookId}/chapters`)

export const getChapter = (bookId, idx) =>
  api.get(`/books/${bookId}/chapters/${idx}`)

export const searchBook = (bookId, query) =>
  api.get(`/books/${bookId}/search?q=${encodeURIComponent(query)}`)
