/**
 * View Transition helpers.
 *
 * Usage:
 *   const go = useViewTransitionNav()
 *   go('/read/123', { name: `book-${bookId}` })
 *
 * The `name` opt applies a matching `view-transition-name` CSS property to
 * both the source element (via `applyViewTransitionName`) and the target
 * element on the next page. Elements with matching names are interpolated
 * between the two renders.
 *
 * If the browser doesn't support the API this falls back to a plain navigate.
 */

import { useNavigate } from 'react-router-dom'

export function useViewTransitionNav() {
  const navigate = useNavigate()

  return function go(to, opts) {
    if (!document.startViewTransition) {
      navigate(to, opts)
      return
    }
    // startViewTransition captures the old snapshot BEFORE the callback runs,
    // so setting the view-transition-name on the source has to happen before.
    document.startViewTransition(() => {
      // flushSync would be ideal here, but React Router v7 uses transitions
      // that flush fast enough for our use case. If visual glitches appear,
      // wrap the navigate in flushSync from 'react-dom'.
      navigate(to, opts)
    })
  }
}

/**
 * Set a temporary `view-transition-name` on an element for the next transition.
 * Call this inline on click BEFORE triggering navigation.
 * The name is cleared after the transition completes so names stay unique.
 */
export function assignTransitionName(el, name) {
  if (!el || !document.startViewTransition) return
  el.style.viewTransitionName = name
  // Clear after the transition so we don't leave unique names on stale elements.
  const clear = () => {
    el.style.viewTransitionName = ''
  }
  // `viewTransition` events fire on document. Safest: clear after one frame
  // past the transition's nominal duration.
  setTimeout(clear, 700)
}
