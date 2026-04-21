import { useState, useEffect } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'
import { useAuthStore } from '../../stores/useAuthStore.js'
import { useConfigStore } from '../../stores/useConfigStore.js'
import styles from './AuthPage.module.css'

export function AuthPage() {
  const [mode, setMode] = useState('login') // 'login' | 'register'
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [showPw, setShowPw] = useState(false)
  const [error, setError] = useState(null)
  const [busy, setBusy] = useState(false)

  const { login, register } = useAuthStore.getState()
  const hydrateConfig = useConfigStore((s) => s.hydrateFromServer)

  const navigate = useNavigate()
  const location = useLocation()
  const from = location.state?.from || '/'

  async function onSubmit(e) {
    e.preventDefault()
    if (busy) return
    setError(null)
    setBusy(true)
    try {
      if (mode === 'login') {
        await login(username, password)
      } else {
        await register(username, password)
      }
      // Pull authoritative settings after login before leaving /auth.
      await hydrateConfig()
      navigate(from, { replace: true })
    } catch (err) {
      setError(err.message || '操作失败')
    } finally {
      setBusy(false)
    }
  }

  return (
    <main className={styles.stage}>
      <div className={styles.ambient} aria-hidden="true">
        {/* Radial warm glow behind the card */}
        <div className={styles.glow} />

        {/* Ultra-thin guide grid for spatial tension */}
        <div className={styles.grid} />

        {/* Sweeping light arc — the "ray" motif */}
        <svg className={styles.arc} viewBox="0 0 1600 900" preserveAspectRatio="xMidYMid slice">
          <defs>
            <linearGradient id="arcGrad" x1="0%" y1="100%" x2="100%" y2="0%">
              <stop offset="0%" stopColor="var(--accent)" stopOpacity="0" />
              <stop offset="45%" stopColor="var(--accent)" stopOpacity="0.55" />
              <stop offset="100%" stopColor="var(--accent)" stopOpacity="0" />
            </linearGradient>
          </defs>
          <path
            d="M -100 820 Q 520 260 1700 80"
            fill="none"
            stroke="url(#arcGrad)"
            strokeWidth="1.2"
          />
          <path
            d="M -100 880 Q 620 340 1700 160"
            fill="none"
            stroke="url(#arcGrad)"
            strokeWidth="0.6"
            opacity="0.6"
          />
        </svg>

        {/* The hero glyph — 光 (light) */}
        <div className={styles.glyph}>光</div>

        {/* A second mirrored glyph on the opposite side for balance */}
        <div className={styles.glyphEcho}>读</div>

        {/* Sparse constellation of dust motes */}
        <div className={styles.stars}>
          {Array.from({ length: 42 }).map((_, i) => (
            <span
              key={i}
              className={styles.star}
              style={{
                top: `${(i * 37) % 100}%`,
                left: `${(i * 71) % 100}%`,
                animationDelay: `${(i % 8) * 0.6}s`,
                opacity: 0.15 + ((i * 13) % 40) / 100,
              }}
            />
          ))}
        </div>

        {/* Editorial corner marks — top-left */}
        <div className={styles.markTopLeft}>
          <span className={styles.markDot} />
          <span className={styles.markText}>LVMINA · MMXXVI</span>
        </div>

        {/* Vertical poetic strip — bottom-right */}
        <div className={styles.markBottomRight}>
          <span className={styles.vChar}>灯</span>
          <span className={styles.vChar}>下</span>
          <span className={styles.vChar}>夜</span>
          <span className={styles.vChar}>读</span>
        </div>

        {/* Rotating edition number — top-right */}
        <div className={styles.markTopRight}>
          <span className={styles.editionNum}>№ 001</span>
          <span className={styles.editionDivider} />
          <span className={styles.editionSub}>PRIVATE LIBRARY</span>
        </div>

        {/* Classical rules — top + bottom */}
        <div className={`${styles.rule} ${styles.ruleTop}`} />
        <div className={`${styles.rule} ${styles.ruleBottom}`} />
      </div>

      <div className={`glass-strong ${styles.card}`}>
        <div className={styles.brand}>
          <span className={styles.mark}>L</span>
          <div className={styles.brandText}>
            <span className={styles.brandName}>Lumina</span>
            <span className={styles.brandTag}>一间属于你的书房</span>
          </div>
        </div>

        <div className={styles.tabs} role="tablist">
          <button
            role="tab"
            aria-selected={mode === 'login'}
            className={`${styles.tab} ${mode === 'login' ? styles.tabActive : ''}`}
            onClick={() => setMode('login')}
          >
            登录
          </button>
          <button
            role="tab"
            aria-selected={mode === 'register'}
            className={`${styles.tab} ${mode === 'register' ? styles.tabActive : ''}`}
            onClick={() => setMode('register')}
          >
            注册
          </button>
          <span
            className={styles.tabIndicator}
            style={{ transform: `translateX(${mode === 'register' ? '100%' : '0'})` }}
          />
        </div>

        <form className={styles.form} onSubmit={onSubmit}>
          <label className={styles.field}>
            <span className={styles.label}>用户名</span>
            <input
              type="text"
              className={styles.input}
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              placeholder="任意字符，支持中文 / emoji"
              autoComplete="username"
              autoFocus
              required
            />
          </label>

          <label className={styles.field}>
            <span className={styles.label}>密码</span>
            <div className={styles.pwWrap}>
              <input
                type={showPw ? 'text' : 'password'}
                className={styles.input}
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder="至少 8 位"
                autoComplete={mode === 'login' ? 'current-password' : 'new-password'}
                required
                minLength={mode === 'register' ? 8 : undefined}
              />
              <button
                type="button"
                className={styles.pwToggle}
                onClick={() => setShowPw((v) => !v)}
              >
                {showPw ? '隐藏' : '显示'}
              </button>
            </div>
            {mode === 'register' && <PasswordStrength value={password} />}
          </label>

          {error && <div className={styles.error}>{error}</div>}

          <button type="submit" className={styles.submit} disabled={busy}>
            {busy ? '处理中…' : mode === 'login' ? '进入书房' : '创建书房'}
          </button>
        </form>

        <p className={styles.hint}>
          {mode === 'login' ? (
            <>
              还没有账号？
              <button type="button" className={styles.switchBtn} onClick={() => setMode('register')}>
                注册一个
              </button>
            </>
          ) : (
            <>
              已经有账号？
              <button type="button" className={styles.switchBtn} onClick={() => setMode('login')}>
                去登录
              </button>
            </>
          )}
        </p>
      </div>
    </main>
  )
}

function PasswordStrength({ value }) {
  const score = scorePassword(value)
  return (
    <div className={styles.strength} aria-hidden="true">
      {[0, 1, 2, 3].map((i) => (
        <span
          key={i}
          className={`${styles.strengthBar} ${i < score ? styles.strengthFilled : ''}`}
        />
      ))}
    </div>
  )
}

function scorePassword(pw) {
  if (!pw) return 0
  let s = 0
  if (pw.length >= 8) s++
  if (pw.length >= 12) s++
  if (/[A-Z]/.test(pw) && /[a-z]/.test(pw)) s++
  if (/\d/.test(pw) && /\W/.test(pw)) s++
  return Math.min(s, 4)
}
