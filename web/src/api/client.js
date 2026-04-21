// Thin fetch wrapper shared by all API modules.
// - Always sends the session cookie (credentials: 'include').
// - Unwraps the unified error envelope  { error: { code, message } }.
// - On 401 clears the auth cache and redirects to /auth.

const BASE = '/api'

export class ApiError extends Error {
  constructor(status, code, message) {
    super(message || code || `HTTP ${status}`)
    this.status = status
    this.code = code
  }
}

let onUnauthorized = null
export function setUnauthorizedHandler(fn) {
  onUnauthorized = fn
}

async function request(path, { method = 'GET', body, headers, signal, raw } = {}) {
  const init = {
    method,
    credentials: 'include',
    headers: {
      Accept: 'application/json',
      ...(body && !(body instanceof FormData) ? { 'Content-Type': 'application/json' } : {}),
      ...headers,
    },
    signal,
  }
  if (body !== undefined) {
    init.body = body instanceof FormData ? body : JSON.stringify(body)
  }

  let res
  try {
    res = await fetch(`${BASE}${path}`, init)
  } catch (err) {
    throw new ApiError(0, 'NETWORK', err.message || '网络连接失败')
  }

  if (res.status === 204) return null

  if (res.status === 401) {
    if (onUnauthorized) onUnauthorized()
  }

  if (!res.ok) {
    let code = 'INTERNAL'
    let message = `请求失败 (${res.status})`
    try {
      const body = await res.json()
      if (body?.error?.code) code = body.error.code
      if (body?.error?.message) message = body.error.message
    } catch {
      /* not JSON */
    }
    throw new ApiError(res.status, code, message)
  }

  if (raw) return res
  const ct = res.headers.get('content-type') || ''
  if (!ct.includes('application/json')) return null
  return res.json()
}

export const api = {
  get: (p, opts) => request(p, { ...opts, method: 'GET' }),
  post: (p, body, opts) => request(p, { ...opts, method: 'POST', body }),
  put: (p, body, opts) => request(p, { ...opts, method: 'PUT', body }),
  patch: (p, body, opts) => request(p, { ...opts, method: 'PATCH', body }),
  del: (p, opts) => request(p, { ...opts, method: 'DELETE' }),
}
