export function ConfidenceBar({ score }) {
  const pct = Math.max(0, Math.min(100, (Number(score) / 40) * 100))
  const color = pct > 70 ? 'var(--green)' : pct > 40 ? 'rgba(17,255,153,0.55)' : 'rgba(17,255,153,0.25)'
  return (
    <div style={{ height: '2px', background: 'var(--frost)', borderRadius: '9999px', overflow: 'hidden' }}>
      <div style={{
        height: '100%',
        width: `${pct}%`,
        background: color,
        borderRadius: '9999px',
        transition: 'width 0.5s ease',
      }} />
    </div>
  )
}
