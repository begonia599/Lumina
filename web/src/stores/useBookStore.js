import { create } from 'zustand'
import * as booksApi from '../api/books.js'
import { useReaderStore } from './useReaderStore.js'

/** If the reader currently has this book open, merge the change there too. */
function syncReaderBook(id, partial) {
  const readerState = useReaderStore.getState()
  if (readerState.bookId === id && readerState.book) {
    useReaderStore.setState({
      book: { ...readerState.book, ...partial },
    })
  }
}

export const useBookStore = create((set, get) => ({
  books: [],
  status: 'idle', // 'idle' | 'loading' | 'ready' | 'error'
  error: null,

  async load() {
    set({ status: 'loading', error: null })
    try {
      const books = await booksApi.listBooks()
      set({ books: books || [], status: 'ready' })
    } catch (err) {
      set({ status: 'error', error: err.message })
    }
  },

  async upload(file) {
    const { book } = await booksApi.uploadBook(file)
    set({ books: [book, ...get().books] })
    return book
  },

  async remove(id) {
    await booksApi.deleteBook(id)
    set({ books: get().books.filter((b) => b.id !== id) })
  },

  async patch(id, partial) {
    const { book } = await booksApi.updateBook(id, partial)
    set({
      books: get().books.map((b) => (b.id === id ? { ...b, ...book } : b)),
    })
    syncReaderBook(id, book)
    return book
  },

  async setCover(id, file) {
    const { book } = await booksApi.uploadCover(id, file)
    const bust = Date.now()
    set({
      books: get().books.map((b) =>
        b.id === id ? { ...b, ...book, coverBust: bust } : b
      ),
    })
    syncReaderBook(id, { ...book, coverBust: bust })
    return book
  },

  async removeCover(id) {
    await booksApi.deleteCover(id)
    const bust = Date.now()
    set({
      books: get().books.map((b) =>
        b.id === id ? { ...b, hasCover: false, coverBust: bust } : b
      ),
    })
    syncReaderBook(id, { hasCover: false, coverBust: bust })
  },

  clear() {
    set({ books: [], status: 'idle', error: null })
  },
}))
