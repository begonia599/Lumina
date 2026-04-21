import { create } from 'zustand'
import * as authApi from '../api/auth.js'

export const useAuthStore = create((set) => ({
  user: null,
  status: 'idle', // 'idle' | 'loading' | 'authed' | 'guest'
  error: null,

  async bootstrap() {
    set({ status: 'loading', error: null })
    try {
      const data = await authApi.me()
      set({ user: data.user, status: 'authed' })
    } catch {
      set({ user: null, status: 'guest' })
    }
  },

  async login(username, password) {
    set({ error: null })
    const data = await authApi.login(username, password)
    set({ user: data.user, status: 'authed' })
    return data.user
  },

  async register(username, password) {
    set({ error: null })
    const data = await authApi.register(username, password)
    set({ user: data.user, status: 'authed' })
    return data.user
  },

  async logout() {
    try {
      await authApi.logout()
    } catch {
      /* ignore — cookie was likely gone */
    }
    set({ user: null, status: 'guest' })
  },
}))
