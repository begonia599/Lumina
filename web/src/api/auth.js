import { api } from './client.js'

export const register = (username, password) =>
  api.post('/auth/register', { username, password })

export const login = (username, password) =>
  api.post('/auth/login', { username, password })

export const logout = () => api.post('/auth/logout')

export const me = () => api.get('/auth/me')
