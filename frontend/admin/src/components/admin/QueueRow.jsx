function fmtTime(ts) {
  try { return new Date(ts).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' }) }
  catch { return '--:--:--' }
}

export function QueueRow({ request }) {
  return (
    <div style={{
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'space-between',
      padding: '9px 18px',
      borderBottom: '1px solid var(--frost-alt)',
      transition: 'background 0.1s',
    }}
    onMouseEnter={e => e.currentTarget.style.background = 'rgba(255,255,255,0.025)'}
    onMouseLeave={e => e.currentTarget.style.background = 'transparent'}
    >
      <div style={{ display: 'flex', alignItems: 'center', gap: '9px' }}>
        <span style={{
          width: '5px',
          height: '5px',
          borderRadius: '9999px',
          background: 'rgba(255,196,61,0.7)',
          flexShrink: 0,
        }} />
        <span style={{ fontSize: '13px', color: 'var(--near-white)', fontWeight: 400 }}>
          {request.requester_name}
        </span>
      </div>
      <span style={{
        fontFamily: "'JetBrains Mono', 'Courier New', monospace",
        fontSize: '10px',
        color: 'var(--dark-gray)',
        letterSpacing: '0.04em',
      }}>
        {fmtTime(request.created_at)}
      </span>
    </div>
  )
}
