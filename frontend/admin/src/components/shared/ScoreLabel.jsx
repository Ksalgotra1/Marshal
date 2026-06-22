export function ScoreLabel({ score }) {
  const n = Number(score)
  return (
    <div style={{ display: 'flex', alignItems: 'baseline', gap: '4px' }}>
      <span style={{
        fontFamily: "'JetBrains Mono', 'Courier New', monospace",
        fontSize: '28px',
        fontWeight: 400,
        color: 'var(--green)',
        lineHeight: 1,
        letterSpacing: '-0.03em',
      }}>
        {isNaN(n) ? '—' : n.toFixed(1)}
      </span>
      <span style={{ fontSize: '11px', color: 'var(--dark-gray)', fontWeight: 400 }}>
        score
      </span>
    </div>
  )
}
