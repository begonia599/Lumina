export function GlassPanel({ as: Tag = 'div', strong = false, className = '', children, ...rest }) {
  const base = strong ? 'glass-strong' : 'glass'
  return (
    <Tag className={`${base} ${className}`.trim()} {...rest}>
      {children}
    </Tag>
  )
}
