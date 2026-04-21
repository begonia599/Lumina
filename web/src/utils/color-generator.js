// Deterministic color generator.
// Maps a book title to a stable gradient for cover rendering.

function hashString(s) {
  let h = 2166136261 >>> 0
  for (let i = 0; i < s.length; i++) {
    h ^= s.charCodeAt(i)
    h = Math.imul(h, 16777619) >>> 0
  }
  return h
}

/** Generate a two-stop gradient + decorative glyph for a book cover. */
export function coverFromTitle(title) {
  const h = hashString(title || 'Untitled')
  const hue1 = h % 360
  const hue2 = (hue1 + 30 + ((h >> 9) % 80)) % 360
  const sat = 45 + (h >> 3) % 25 // 45–70
  const light1 = 38 + (h >> 5) % 14 // 38–52
  const light2 = 22 + (h >> 11) % 12 // 22–34
  const angle = (h >> 7) % 180
  const shape = ['disc', 'slash', 'dots', 'arc', 'triangle'][h % 5]
  return {
    gradient: `linear-gradient(${angle}deg, hsl(${hue1}, ${sat}%, ${light1}%) 0%, hsl(${hue2}, ${sat}%, ${light2}%) 100%)`,
    shape,
    angle,
    hue1,
    hue2,
  }
}
