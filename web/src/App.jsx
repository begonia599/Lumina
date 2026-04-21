import { useEffect } from 'react'
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { useAuthStore } from './stores/useAuthStore.js'
import { useConfigStore } from './stores/useConfigStore.js'
import { setUnauthorizedHandler } from './api/client.js'
import {
  RequireAuth,
  RedirectIfAuthed,
  useAuthBootstrap,
} from './components/RequireAuth.jsx'
import { AuthPage } from './pages/Auth/AuthPage.jsx'
import { Bookshelf } from './pages/Bookshelf/Bookshelf.jsx'
import { Reader } from './pages/Reader/Reader.jsx'

function AppShell() {
  useAuthBootstrap()
  const status = useAuthStore((s) => s.status)
  const applyConfig = useConfigStore((s) => s.applyAll)
  const hydrateConfig = useConfigStore((s) => s.hydrateFromServer)

  // Apply local theme snapshot ASAP to avoid FOUC.
  useEffect(() => {
    applyConfig()
  }, [applyConfig])

  // When authentication settles, overlay remote settings on top.
  useEffect(() => {
    if (status === 'authed') hydrateConfig()
  }, [status, hydrateConfig])

  return (
    <Routes>
      <Route
        path="/auth"
        element={
          <RedirectIfAuthed>
            <AuthPage />
          </RedirectIfAuthed>
        }
      />
      <Route element={<RequireAuth />}>
        <Route path="/" element={<Bookshelf />} />
        <Route path="/read/:bookId" element={<Reader />} />
      </Route>
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  )
}

export default function App() {
  // Wire 401 → clear auth store & let RequireAuth redirect.
  // Installed once at module scope; safe to re-install on HMR.
  useEffect(() => {
    setUnauthorizedHandler(() => {
      const { status, logout } = useAuthStore.getState()
      if (status === 'authed') logout()
    })
  }, [])

  return (
    <BrowserRouter>
      <AppShell />
    </BrowserRouter>
  )
}
