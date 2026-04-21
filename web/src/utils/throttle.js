/** Classic leading-edge + trailing-edge throttle. */
export function throttle(fn, wait = 250) {
  let last = 0
  let timer = null
  let lastArgs = null
  return (...args) => {
    const now = Date.now()
    lastArgs = args
    const remaining = wait - (now - last)
    if (remaining <= 0) {
      if (timer) {
        clearTimeout(timer)
        timer = null
      }
      last = now
      fn(...args)
    } else if (!timer) {
      timer = setTimeout(() => {
        last = Date.now()
        timer = null
        fn(...lastArgs)
      }, remaining)
    }
  }
}

export function debounce(fn, wait = 500) {
  let timer = null
  return (...args) => {
    if (timer) clearTimeout(timer)
    timer = setTimeout(() => fn(...args), wait)
  }
}
