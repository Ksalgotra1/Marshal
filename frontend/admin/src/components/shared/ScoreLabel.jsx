export function ScoreLabel({ score }) {
  const n = Number(score)
  return (
    <div style={{ display: 'flex', alignItems: 'baseline', gap: '6px' }}>
      <span style={{ fontSize: '11px', color: 'var(--dark-gray)', fontWeight: 500, letterSpacing: '0.04em', textTransform: 'uppercase' }}>
        Route Score:
      </span>
      <span style={{
        fontFamily: "'JetBrains Mono', 'Courier New', monospace",
        fontSize: '22px',
        fontWeight: 400,
        color: 'var(--green)',
        lineHeight: 1,
        letterSpacing: '-0.02em',
      }}>
        {isNaN(n) ? '—' : n.toFixed(1)}
      </span>
    </div>
  )
}
