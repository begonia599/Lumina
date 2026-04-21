import { api } from './client.js'

export const getSettings = () => api.get('/settings')

export const updateSettings = (partial) => api.put('/settings', partial)
