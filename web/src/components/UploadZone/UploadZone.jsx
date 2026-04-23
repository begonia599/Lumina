import { useRef, useState } from 'react'
import { UploadCloud } from 'lucide-react'
import styles from './UploadZone.module.css'

export function UploadZone({ onUpload, variant = 'full' }) {
  const inputRef = useRef(null)
  const [dragging, setDragging] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState(null)

  async function handleFiles(files) {
    const file = files?.[0]
    if (!file) return
    const ext = file.name.toLowerCase().split('.').pop()
    if (!['txt', 'epub'].includes(ext)) {
      setError('目前仅支持 .txt 文件')
      return
    }
    setError(null)
    setBusy(true)
    try {
      await onUpload(file)
    } catch (err) {
      setError(err.message || '上传失败')
    } finally {
      setBusy(false)
      if (inputRef.current) inputRef.current.value = ''
    }
  }

  return (
    <label
      className={`${styles.zone} ${styles[variant]} ${dragging ? styles.dragging : ''} ${busy ? styles.busy : ''}`}
      onDragOver={(e) => {
        e.preventDefault()
        setDragging(true)
      }}
      onDragLeave={() => setDragging(false)}
      onDrop={(e) => {
        e.preventDefault()
        setDragging(false)
        handleFiles(e.dataTransfer.files)
      }}
    >
      <input
        ref={inputRef}
        type="file"
        accept=".txt,.epub,text/plain,application/epub+zip"
        className={styles.input}
        onChange={(e) => handleFiles(e.target.files)}
      />
      <UploadCloud className={styles.icon} size={variant === 'compact' ? 18 : 32} />
      {variant === 'full' ? (
        <>
          <span className={styles.title}>{busy ? '上传中…' : '拖入 TXT，或点击选择'}</span>
          <span className={styles.hint}>
            {error || '支持中文编码自动识别（GBK / UTF-8）'}
          </span>
        </>
      ) : (
        <span className={styles.titleCompact}>{busy ? '上传中…' : '新增书籍'}</span>
      )}
    </label>
  )
}
