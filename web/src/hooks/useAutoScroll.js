import { useEffect, useRef } from 'react'

/**
 * useAutoScroll — continuous page auto-scroll (teleprompter style).
 *
 * Contract:
 *   - `enabled`        toggles the loop. Set to false to pause.
 *   - `speed`          reading speed in px/sec (recommend 20–120).
 *   - `onReachEnd()`   fired when the bottom is reached. Caller decides
 *                      whether to load next chapter or just stop.
 *   - `onInterrupt()`  fired when the user performs a manual scroll gesture
 *                      (wheel / touchmove / PageUp / Home / End / arrow keys).
 *                      Caller should set enabled=false.
 *
 * Implementation notes:
 *   - We distinguish *our* programmatic scroll from *user* scroll by listening
 *     to wheel / touchmove / keydown (input sources) instead of the aggregated
 *     `scroll` event. `window.scrollBy` does not fire wheel events, so we
 *     don't trigger our own interrupt.
 *   - Subpixel accumulator avoids speed quantization at low px/sec values.
 */
export function useAutoScroll({ enabled, speed, onReachEnd, onInterrupt }) {
  // Keep callback identity out of the effect deps.
  const endRef = useRef(onReachEnd)
  const interruptRef = useRef(onInterrupt)
  useEffect(() => {
    endRef.current = onReachEnd
    interruptRef.current = onInterrupt
  }, [onReachEnd, onInterrupt])

  useEffect(() => {
    if (!enabled) return

    let rafId = 0
    let lastTs = 0
    let fractional = 0 // accumulate sub-px movement
    let stalledFrames = 0 // consecutive frames with zero progress

    const tick = (ts) => {
      if (lastTs === 0) lastTs = ts
      const dt = Math.min(0.1, (ts - lastTs) / 1000) // clamp, in case tab was backgrounded
      lastTs = ts

      fractional += speed * dt
      const stepInt = Math.floor(fractional)
      fractional -= stepInt

      if (stepInt > 0) {
        const before = window.scrollY
        window.scrollBy({ top: stepInt, left: 0, behavior: 'instant' })
        if (window.scrollY === before) {
          stalledFrames++
          // Two consecutive stalls → we're at the bottom.
          if (stalledFrames >= 2) {
            endRef.current?.()
            return
          }
        } else {
          stalledFrames = 0
        }
      }

      rafId = requestAnimationFrame(tick)
    }
    rafId = requestAnimationFrame(tick)

    const interruptKeys = new Set([
      'PageDown',
      'PageUp',
      'ArrowUp',
      'ArrowDown',
      'Home',
      'End',
      ' ',
      'Spacebar',
    ])
    const interrupt = () => interruptRef.current?.()
    const onKey = (e) => {
      if (interruptKeys.has(e.key)) interrupt()
    }
    window.addEventListener('wheel', interrupt, { passive: true })
    window.addEventListener('touchmove', interrupt, { passive: true })
    window.addEventListener('keydown', onKey)

    return () => {
      cancelAnimationFrame(rafId)
      window.removeEventListener('wheel', interrupt)
      window.removeEventListener('touchmove', interrupt)
      window.removeEventListener('keydown', onKey)
    }
  }, [enabled, speed])
}
