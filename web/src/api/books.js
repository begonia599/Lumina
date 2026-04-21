import { api } from './client.js'

export const listBooks = () => api.get('/books')

export const getBook = (id) => api.get(`/books/${id}`)

export const deleteBook = (id) => api.del(`/books/${id}`)

export const updateBook = (id, partial) => api.patch(`/books/${id}`, partial)

export function uploadBook(file) {
  const fd = new FormData()
  fd.append('file', file)
  return api.post('/books/upload', fd)
}

export function uploadCover(id, file) {
  const fd = new FormData()
  fd.append('file', file)
  return api.post(`/books/${id}/cover`, fd)
}

export const deleteCover = (id) => api.del(`/books/${id}/cover`)

/** URL for a book's uploaded cover image. Includes a version buster so
 *  CSS `<img>` doesn't keep a stale cached version across edits. */
export const coverUrl = (id, bust) =>
  `/api/books/${id}/cover${bust ? `?v=${bust}` : ''}`
