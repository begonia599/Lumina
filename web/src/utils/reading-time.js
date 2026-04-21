/** Estimate reading time in minutes from a rune count (default 300 wpm for CJK). */
export function estimateMinutes(charCount, wpm = 300) {
  if (!charCount || charCount <= 0) return 0
  return Math.max(1, Math.round(charCount / wpm))
}

export function formatMinutes(minutes) {
  if (!minutes) return '—'
  if (minutes < 60) return `${minutes} 分钟`
  const h = Math.floor(minutes / 60)
  const m = minutes % 60
  return m ? `${h} 小时 ${m} 分` : `${h} 小时`
}
