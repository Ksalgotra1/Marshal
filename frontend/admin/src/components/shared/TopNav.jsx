export function TopNav() {
  return (
    <header style={{
      position: 'sticky',
      top: 0,
      zIndex: 50,
      background: 'rgba(0,0,0,0.85)',
      backdropFilter: 'blur(16px)',
      WebkitBackdropFilter: 'blur(16px)',
      borderBottom: '1px solid var(--frost)',
      height: '52px',
      display: 'flex',
      alignItems: 'center',
      padding: '0 24px',
      justifyContent: 'space-between',
    }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
        <svg width="18" height="18" viewBox="0 0 18 18" fill="none" style={{ flexShrink: 0 }}>
          <path d="M9 2L2.5 5.75V12.25L9 16L15.5 12.25V5.75L9 2Z" stroke="var(--near-white)" strokeWidth="1.2" strokeLinejoin="round" fill="none"/>
          <circle cx="9" cy="9" r="2" fill="var(--near-white)"/>
        </svg>
        <span style={{ fontSize: '14px', fontWeight: 500, color: 'var(--near-white)', letterSpacing: '-0.1px' }}>
          Marshal
        </span>
        <span style={{ color: 'var(--frost)', fontSize: '14px', opacity: 0.6 }}>/</span>
        <span style={{ fontSize: '14px', color: 'var(--silver)' }}>Dispatch Console</span>
      </div>

      <div style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
        <span style={{
          display: 'inline-block',
          width: '5px',
          height: '5px',
          borderRadius: '9999px',
          background: 'var(--green)',
          animation: 'liveBlip 2.4s ease-in-out infinite',
        }} />
        <span style={{
          fontSize: '12px',
          fontWeight: 500,
          color: 'var(--green)',
          letterSpacing: '0.02em',
        }}>
          Live
        </span>
      </div>

      <style>{`@keyframes liveBlip { 0%,100%{opacity:1} 50%{opacity:0.3} }`}</style>
    </header>
  )
}
