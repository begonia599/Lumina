import { useEffect } from 'react'
import { motion, AnimatePresence } from 'motion/react'
import { X } from 'lucide-react'
import styles from './ShortcutHelp.module.css'

/**
 * Keyboard shortcut reference overlay.
 * Opened with `?` (shift+/) or the help button.
 * Static content — kept in sync with Reader.jsx by hand.
 */
const GROUPS = [
  {
    title: '阅读导航',
    items: [
      ['← / →', '上一章 / 下一章'],
      ['T', '打开目录'],
      ['L', '打开书签'],
      ['/', '全书搜索'],
    ],
  },
  {
    title: '标记',
    items: [
      ['B', '在当前位置加书签'],
    ],
  },
  {
    title: '自动滑动',
    items: [
      ['P', '开始 / 暂停'],
      ['[', '减速 5 px/s'],
      [']', '加速 5 px/s'],
    ],
  },
  {
    title: '界面',
    items: [
      [',', '设置'],
      ['?', '本帮助'],
      ['Esc', '关闭任意浮层'],
    ],
  },
]

export function ShortcutHelp({ open, onClose }) {
  useEffect(() => {
    if (!open) return
    const onKey = (e) => {
      if (e.key === 'Escape' || e.key === '?') onClose()
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [open, onClose])

  return (
    <AnimatePresence>
      {open && (
        <>
          <motion.div
            className={styles.backdrop}
            onClick={onClose}
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            transition={{ duration: 0.2 }}
          />
          <motion.div
            role="dialog"
            aria-label="键盘快捷键"
            className={`glass-strong ${styles.panel}`}
            initial={{ opacity: 0, y: 12, scale: 0.96 }}
            animate={{ opacity: 1, y: 0, scale: 1 }}
            exit={{ opacity: 0, y: 12, scale: 0.96 }}
            transition={{ type: 'spring', stiffness: 380, damping: 32 }}
          >
            <header className={styles.header}>
              <h2 className={styles.heading}>快捷键</h2>
              <button className={styles.close} onClick={onClose} title="关闭">
                <X size={18} />
              </button>
            </header>

            <div className={styles.grid}>
              {GROUPS.map((g) => (
                <section key={g.title} className={styles.group}>
                  <h3 className={styles.groupTitle}>{g.title}</h3>
                  <dl className={styles.list}>
                    {g.items.map(([keys, label]) => (
                      <div key={keys} className={styles.row}>
                        <dt className={styles.keys}>
                          {keys.split(/\s*\/\s*/).map((k, i, arr) => (
                            <span key={i}>
                              {k.split(' ').map((sub, j) => (
                                <kbd key={j} className={styles.kbd}>
                                  {sub}
                                </kbd>
                              ))}
                              {i < arr.length - 1 && (
                                <span className={styles.sep}>/</span>
                              )}
                            </span>
                          ))}
                        </dt>
                        <dd className={styles.label}>{label}</dd>
                      </div>
                    ))}
                  </dl>
                </section>
              ))}
            </div>

            <footer className={styles.footer}>
              按 <kbd className={styles.kbd}>?</kbd> 或 <kbd className={styles.kbd}>Esc</kbd> 关闭
            </footer>
          </motion.div>
        </>
      )}
    </AnimatePresence>
  )
}
