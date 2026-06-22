const METRIC_ACCENTS = [
  { color: 'var(--blue)',   dimBg: 'var(--blue-dim)'   },
  { color: 'var(--green)',  dimBg: 'var(--green-mid)'  },
  { color: 'var(--orange)', dimBg: 'var(--orange-dim)' },
  { color: 'var(--silver)', dimBg: 'rgba(161,164,165,0.10)' },
]

export function StatsBar({ stats }) {
  const metrics = [
    { label: 'Groups today',    value: stats.groups },
    { label: 'Drivers online',  value: stats.onlineDrivers },
    { label: 'Avg confidence',  value: typeof stats.avgScore === 'number' ? stats.avgScore.toFixed(1) : '0.0' },
    { label: 'Live connections', value: stats.connections },
  ]

  return (
    <div style={{
      display: 'grid',
      gridTemplateColumns: 'repeat(4, 1fr)',
      border: '1px solid var(--frost)',
      borderRadius: '16px',
      overflow: 'hidden',
      boxShadow: 'var(--ring)',
    }}>
      {metrics.map((m, i) => {
        const a = METRIC_ACCENTS[i]
        return (
          <div key={m.label} style={{
            padding: '20px 22px',
            borderRight: i < 3 ? '1px solid var(--frost)' : 'none',
            background: 'transparent',
            position: 'relative',
          }}>
            <p style={{
              fontSize: '12px',
              fontWeight: 400,
              color: 'var(--silver)',
              marginBottom: '12px',
              letterSpacing: '0.01em',
            }}>
              {m.label}
            </p>
            <p style={{
              fontFamily: "'JetBrains Mono', 'Courier New', monospace",
              fontSize: '32px',
              fontWeight: 400,
              color: a.color,
              lineHeight: 1,
              letterSpacing: '-0.02em',
            }}>
              {m.value}
            </p>
          </div>
        )
      })}
    </div>
  )
}
