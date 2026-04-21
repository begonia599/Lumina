import { useEffect } from 'react'
import { Navigate, Outlet, useLocation } from 'react-router-dom'
import { useAuthStore } from '../stores/useAuthStore.js'

/**
 * Route guard. Renders children only when authenticated.
 * Relies on App.jsx to have called bootstrap() on mount.
 */
export function RequireAuth() {
  const status = useAuthStore((s) => s.status)
  const location = useLocation()

  // While the session probe is inflight, render a neutral placeholder.
  // The guard is intentionally quiet — no spinner UI so the eventual
  // successful hydration feels instant.
  if (status === 'idle' || status === 'loading') {
    return <div style={{ minHeight: '100vh' }} aria-busy="true" />
  }

  if (status !== 'authed') {
    return <Navigate to="/auth" replace state={{ from: location.pathname }} />
  }

  return <Outlet />
}

/** Inverse — redirects authed users away from /auth. */
export function RedirectIfAuthed({ to = '/', children }) {
  const status = useAuthStore((s) => s.status)
  if (status === 'authed') return <Navigate to={to} replace />
  return children
}

/** Utility: bootstrap once on mount. */
export function useAuthBootstrap() {
  const bootstrap = useAuthStore((s) => s.bootstrap)
  const status = useAuthStore((s) => s.status)
  useEffect(() => {
    if (status === 'idle') bootstrap()
  }, [bootstrap, status])
}
