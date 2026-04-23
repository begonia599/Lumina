function clamp(value, min, max) {
  return Math.max(min, Math.min(max, value))
}

function cumulativeCharOffset(paragraphs, upToIdx) {
  let n = 0
  for (let k = 0; k < upToIdx; k++) {
    n += [...paragraphs[k]].length
  }
  return n
}

function scrollToCharOffset(map, paragraphs, targetOffset) {
  let cum = 0
  for (let i = 0; i < paragraphs.length; i++) {
    const len = [...paragraphs[i]].length
    if (cum + len >= targetOffset) {
      const el = map.get(i)
      if (el) el.scrollIntoView({ block: 'start', behavior: 'instant' })
      return
    }
    cum += len
  }
}

export function capturePosition(chapter, paraRefs, epubContentRef) {
  if (!chapter) return null

  if (chapter.format === 'epub') {
    const container = epubContentRef?.current
    if (!container) {
      return {
        chapterIdx: chapter.chapterIdx,
        charOffset: 0,
      }
    }

    const rect = container.getBoundingClientRect()
    const top = window.scrollY + rect.top
    const range = Math.max(1, container.offsetHeight - window.innerHeight)
    const scrollPct = clamp((window.scrollY - top) / range, 0, 1)

    let anchor = null
    const viewportTop = 84
    const chapterRootId = `ch-${chapter.chapterIdx}`
    const anchors = Array.from(container.querySelectorAll('[id]')).filter(
      (el) => el.id && el.id !== chapterRootId,
    )
    for (const el of anchors) {
      const elRect = el.getBoundingClientRect()
      if (elRect.top <= viewportTop) anchor = el.id
      else break
    }

    return {
      chapterIdx: chapter.chapterIdx,
      charOffset: 0,
      ...(anchor ? { anchor } : {}),
      scrollPct,
    }
  }

  let visibleOffset = 0
  const topPad = 80
  for (let i = 0; i < chapter.paragraphs.length; i++) {
    const el = paraRefs.current.get(i)
    if (!el) continue
    const rect = el.getBoundingClientRect()
    if (rect.bottom < topPad) continue
    visibleOffset = cumulativeCharOffset(chapter.paragraphs, i)
    break
  }

  return {
    chapterIdx: chapter.chapterIdx,
    charOffset: visibleOffset,
  }
}

export function restorePosition(chapter, position, paraRefs, epubContentRef) {
  if (!chapter) return

  if (chapter.format === 'epub') {
    const container = epubContentRef?.current
    if (!container) return

    const chapterRootId = `ch-${chapter.chapterIdx}`
    if (position?.anchor && position.anchor !== chapterRootId) {
      const target = document.getElementById(position.anchor)
      if (target) {
        target.scrollIntoView({ block: 'start', behavior: 'instant' })
        return
      }
    }

    const scrollPct = clamp(position?.scrollPct ?? 0, 0, 1)
    const rect = container.getBoundingClientRect()
    const top = window.scrollY + rect.top
    const range = Math.max(0, container.offsetHeight - window.innerHeight)
    window.scrollTo({ top: top + range * scrollPct, behavior: 'instant' })
    return
  }

  if (position?.charOffset > 0) {
    scrollToCharOffset(paraRefs.current, chapter.paragraphs, position.charOffset)
    return
  }
  window.scrollTo({ top: 0, behavior: 'instant' })
}

export { cumulativeCharOffset, scrollToCharOffset }
