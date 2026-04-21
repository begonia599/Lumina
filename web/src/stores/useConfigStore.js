import { create } from 'zustand'
import * as settingsApi from '../api/settings.js'

/**
 * Configuration store. Values have three states:
 *   1. defaults (below) — applied instantly on mount to avoid FOUC
 *   2. localStorage snapshot — applied on boot before /auth/me returns
 *   3. backend /settings — authoritative, overrides both on login
 *
 * Writes go to localStorage synchronously and to the backend via debounce.
 */

const STORAGE_KEY = 'lumina.config.v1'

const defaults = {
  theme: 'dark', // 'dark' | 'light' | 'sepia'
  fontSize: 1.15, // rem, reader body
  lineHeight: 1.85,
  pageWidth: 760, // px, reader container max-width
  fontFamily: 'noto-serif-sc',
  textAlign: 'left', // 'left' | 'justify'
  indent: true,
  autoScrollSpeed: 40, // px/sec — ~12 lines/min at 1.15rem/1.85
  autoAdvance: true, // continue to next chapter when auto-scroll reaches end
}

function loadLocal() {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return {}
    return JSON.parse(raw)
  } catch {
    return {}
  }
}

function persistLocal(state) {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(state))
  } catch {
    /* storage disabled */
  }
}

function applyTheme(theme) {
  const root = document.documentElement
  root.setAttribute('data-theme', theme)
}

function applyReaderVars({ fontSize, lineHeight, pageWidth, textAlign, indent }) {
  const root = document.documentElement
  root.style.setProperty('--reader-font-size', `${fontSize}rem`)
  root.style.setProperty('--reader-line-height', String(lineHeight))
  root.style.setProperty('--reader-width', `${pageWidth}px`)
  root.setAttribute('data-text-align', textAlign)
  root.setAttribute('data-indent', indent ? 'on' : 'off')
}

let debounceTimer = null
function scheduleRemoteSave(snapshot) {
  if (debounceTimer) clearTimeout(debounceTimer)
  debounceTimer = setTimeout(() => {
    settingsApi.updateSettings(snapshot).catch(() => {
      /* silent — local state is still correct */
    })
  }, 800)
}

export const useConfigStore = create((set, get) => ({
  ...defaults,
  ...loadLocal(),

  /** Apply current state to the DOM (run once on mount). */
  applyAll() {
    const s = get()
    applyTheme(s.theme)
    applyReaderVars(s)
  },

  /** Merge values, persist locally, schedule remote save. */
  patch(partial) {
    const next = { ...get(), ...partial }
    set(partial)
    if (partial.theme) applyTheme(next.theme)
    // Re-apply reader-scoped vars for any reader-related change.
    if (
      'fontSize' in partial ||
      'lineHeight' in partial ||
      'pageWidth' in partial ||
      'textAlign' in partial ||
      'indent' in partial
    ) {
      applyReaderVars(next)
    }
    const snapshot = {
      theme: next.theme,
      fontSize: next.fontSize,
      lineHeight: next.lineHeight,
      pageWidth: next.pageWidth,
      fontFamily: next.fontFamily,
      textAlign: next.textAlign,
      indent: next.indent,
      autoScrollSpeed: next.autoScrollSpeed,
      autoAdvance: next.autoAdvance,
    }
    persistLocal(snapshot)
    scheduleRemoteSave(snapshot)
  },

  /** Pull authoritative settings from the backend (on login). */
  async hydrateFromServer() {
    try {
      const remote = await settingsApi.getSettings()
      if (!remote || Object.keys(remote).length === 0) return
      const merged = { ...defaults, ...loadLocal(), ...remote }
      set(merged)
      applyTheme(merged.theme)
      applyReaderVars(merged)
      persistLocal(merged)
    } catch {
      /* backend unavailable — keep local */
    }
  },
}))
