function statusStyle(status) {
  if (status === 'online') return { dot: 'var(--green)',  label: 'Online',  text: 'var(--green)' }
  if (status === 'busy')   return { dot: 'var(--orange)', label: 'Busy',    text: 'var(--orange)' }
  return                          { dot: 'var(--dark-gray)', label: 'Offline', text: 'var(--dark-gray)' }
}

export function DriverListItem({ driver }) {
  const s = statusStyle(driver.status)
  const ini = driver.name.split(' ').map(p => (p[0] || '').toUpperCase()).join('').slice(0, 2)
  return (
    <div
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: '12px',
        padding: '10px 18px',
        borderBottom: '1px solid var(--frost-alt)',
        transition: 'background 0.1s',
      }}
      onMouseEnter={e => e.currentTarget.style.background = 'rgba(255,255,255,0.025)'}
      onMouseLeave={e => e.currentTarget.style.background = 'transparent'}
    >
      <div style={{
        width: '30px',
        height: '30px',
        borderRadius: '9999px',
        border: '1px solid var(--frost)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        flexShrink: 0,
        fontFamily: "'JetBrains Mono', monospace",
        fontSize: '9px',
        fontWeight: 500,
        color: 'var(--silver)',
        letterSpacing: 0,
      }}>
        {ini}
      </div>

      <div style={{ flex: 1, minWidth: 0 }}>
        <div style={{ fontSize: '13px', color: 'var(--near-white)', fontWeight: 400, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
          {driver.name}
        </div>
        <div style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: '10px', color: 'var(--dark-gray)', marginTop: '1px', letterSpacing: '0.04em' }}>
          {driver.id.slice(0, 8)}
        </div>
      </div>

      <div style={{ display: 'flex', alignItems: 'center', gap: '5px', flexShrink: 0 }}>
        <span style={{ width: '5px', height: '5px', borderRadius: '9999px', background: s.dot }} />
        <span style={{ fontSize: '11px', color: s.text, fontWeight: 500 }}>{s.label}</span>
      </div>
    </div>
  )
}
