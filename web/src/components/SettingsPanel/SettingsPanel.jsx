import { useEffect } from 'react'
import { motion, AnimatePresence } from 'motion/react'
import { X, Moon, Sun, Scroll } from 'lucide-react'
import { useConfigStore } from '../../stores/useConfigStore.js'
import styles from './SettingsPanel.module.css'

const THEMES = [
  { id: 'dark', icon: Moon, name: '深渊', sub: '深夜的书房' },
  { id: 'light', icon: Sun, name: '晨曦', sub: '窗边初醒' },
  { id: 'sepia', icon: Scroll, name: '羊皮纸', sub: '旧图书馆' },
]

const PRESETS = {
  compact: { fontSize: 1.0, lineHeight: 1.65, pageWidth: 680 },
  standard: { fontSize: 1.15, lineHeight: 1.85, pageWidth: 760 },
  relaxed: { fontSize: 1.28, lineHeight: 2.0, pageWidth: 820 },
}

export function SettingsPanel({ open, onClose }) {
  const config = useConfigStore()

  useEffect(() => {
    if (!open) return
    const onKey = (e) => {
      if (e.key === 'Escape') onClose()
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
          <motion.aside
            className={`glass-strong ${styles.panel}`}
            role="dialog"
            aria-label="阅读设置"
            initial={{ x: '105%' }}
            animate={{ x: 0 }}
            exit={{ x: '105%' }}
            transition={{ type: 'spring', stiffness: 320, damping: 34 }}
            drag="x"
            dragConstraints={{ left: 0, right: 420 }}
            dragElastic={0.08}
            dragMomentum={false}
            onDragEnd={(_, info) => {
              if (info.offset.x > 100 || info.velocity.x > 500) onClose()
            }}
          >
            <div className={styles.dragHandle} aria-hidden="true" />
            <header className={styles.header}>
              <h2 className={styles.heading}>阅读设置</h2>
              <button className={styles.close} onClick={onClose} title="关闭">
                <X size={18} />
              </button>
            </header>

            <div className={styles.preview}>
              <p
                style={{
                  fontSize: `${config.fontSize}rem`,
                  lineHeight: config.lineHeight,
                  fontFamily: 'var(--font-reading)',
                  textAlign: config.textAlign,
                  textIndent: config.indent ? '2em' : 0,
                }}
              >
                山间小径通向松柏深处，风过林梢如潮水般起伏，他停下脚步，听见自己的心跳与远处溪水相应和。
              </p>
            </div>

            <section className={styles.section}>
              <h3 className={styles.sectionTitle}>主题氛围</h3>
              <div className={styles.themeGrid}>
                {THEMES.map((t) => {
                  const Icon = t.icon
                  const active = config.theme === t.id
                  return (
                    <button
                      key={t.id}
                      className={`${styles.themeCard} ${active ? styles.themeActive : ''}`}
                      data-theme-preview={t.id}
                      onClick={() => config.patch({ theme: t.id })}
                    >
                      <Icon size={16} className={styles.themeIcon} />
                      <span className={styles.themeName}>{t.name}</span>
                      <span className={styles.themeSub}>{t.sub}</span>
                    </button>
                  )
                })}
              </div>
            </section>

            <section className={styles.section}>
              <h3 className={styles.sectionTitle}>排版预设</h3>
              <div className={styles.presetRow}>
                {Object.entries(PRESETS).map(([id, preset]) => {
                  const active =
                    config.fontSize === preset.fontSize &&
                    config.lineHeight === preset.lineHeight &&
                    config.pageWidth === preset.pageWidth
                  return (
                    <button
                      key={id}
                      className={`${styles.preset} ${active ? styles.presetActive : ''}`}
                      onClick={() => config.patch(preset)}
                    >
                      {id === 'compact' ? '紧凑' : id === 'standard' ? '标准' : '宽松'}
                    </button>
                  )
                })}
              </div>
            </section>

            <section className={styles.section}>
              <h3 className={styles.sectionTitle}>精细调节</h3>
              <Slider
                label="字号"
                value={config.fontSize}
                min={0.9}
                max={1.5}
                step={0.01}
                display={`${config.fontSize.toFixed(2)} rem`}
                onChange={(v) => config.patch({ fontSize: v })}
              />
              <Slider
                label="行距"
                value={config.lineHeight}
                min={1.4}
                max={2.2}
                step={0.05}
                display={config.lineHeight.toFixed(2)}
                onChange={(v) => config.patch({ lineHeight: v })}
              />
              <Slider
                label="宽度"
                value={config.pageWidth}
                min={560}
                max={960}
                step={10}
                display={`${config.pageWidth}px`}
                onChange={(v) => config.patch({ pageWidth: v })}
              />
            </section>

            <section className={styles.section}>
              <h3 className={styles.sectionTitle}>段落样式</h3>
              <div className={styles.toggleRow}>
                <ToggleChip
                  active={config.textAlign === 'left'}
                  onClick={() => config.patch({ textAlign: 'left' })}
                >
                  左对齐
                </ToggleChip>
                <ToggleChip
                  active={config.textAlign === 'justify'}
                  onClick={() => config.patch({ textAlign: 'justify' })}
                >
                  两端对齐
                </ToggleChip>
                <ToggleChip active={config.indent} onClick={() => config.patch({ indent: !config.indent })}>
                  首行缩进
                </ToggleChip>
              </div>
            </section>

            <section className={styles.section}>
              <h3 className={styles.sectionTitle}>自动滑动</h3>
              <Slider
                label="速度"
                value={config.autoScrollSpeed}
                min={10}
                max={240}
                step={5}
                display={`${config.autoScrollSpeed} px/s`}
                onChange={(v) => config.patch({ autoScrollSpeed: v })}
              />
              <div className={styles.toggleRow}>
                <ToggleChip
                  active={config.autoAdvance}
                  onClick={() => config.patch({ autoAdvance: !config.autoAdvance })}
                >
                  章末自动翻页
                </ToggleChip>
              </div>
              <p className={styles.hint}>
                阅读时按 <kbd className={styles.kbd}>P</kbd> 开关，
                <kbd className={styles.kbd}>[</kbd> / <kbd className={styles.kbd}>]</kbd> 微调速度
              </p>
            </section>
          </motion.aside>
        </>
      )}
    </AnimatePresence>
  )
}

function Slider({ label, value, min, max, step, display, onChange }) {
  return (
    <label className={styles.slider}>
      <div className={styles.sliderHeader}>
        <span className={styles.sliderLabel}>{label}</span>
        <span className={styles.sliderValue}>{display}</span>
      </div>
      <input
        type="range"
        min={min}
        max={max}
        step={step}
        value={value}
        onChange={(e) => onChange(Number(e.target.value))}
        className={styles.sliderInput}
      />
    </label>
  )
}

function ToggleChip({ active, onClick, children }) {
  return (
    <button className={`${styles.chip} ${active ? styles.chipActive : ''}`} onClick={onClick}>
      {children}
    </button>
  )
}
