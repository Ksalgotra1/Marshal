import { RouteScoreBar } from './RouteScoreBar'
import { ScoreLabel } from './ScoreLabel'
import { StatusBadge } from './StatusBadge'

function fmtTime(ts) {
  try { return new Date(ts).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }) }
  catch { return '--:--' }
}

function initials(name) {
  return name.split(' ').map(p => (p[0] || '').toUpperCase()).join('').slice(0, 2) || '??'
}

export function GroupCard({ group, memberNames = [], driverName }) {
  return (
    <article
      style={{
        border: '1px solid var(--frost)',
        borderRadius: '16px',
        padding: '16px 18px',
        boxShadow: 'var(--ring)',
        background: 'transparent',
        transition: 'border-color 0.15s',
      }}
      onMouseEnter={e => e.currentTarget.style.borderColor = 'rgba(214,235,253,0.35)'}
      onMouseLeave={e => e.currentTarget.style.borderColor = 'var(--frost)'}
    >
      {/* Header */}
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '14px' }}>
        <span style={{
          fontFamily: "'JetBrains Mono', 'Courier New', monospace",
          fontSize: '11px',
          color: 'var(--dark-gray)',
          letterSpacing: '0.05em',
        }}>
          #{group.id.slice(0, 8)}
        </span>
        <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
          {group.priority === 'high' && (
            <span style={{
              display: 'inline-flex',
              alignItems: 'center',
              background: 'var(--orange-dim)',
              color: 'var(--orange)',
              fontSize: '11px',
              fontWeight: 500,
              padding: '3px 10px',
              borderRadius: '9999px',
              letterSpacing: '0.01em',
              whiteSpace: 'nowrap',
            }}>
              <span style={{ width: '4px', height: '4px', borderRadius: '9999px', background: 'var(--orange)', flexShrink: 0, marginRight: '5px' }} />
              Fast Track
            </span>
          )}
          <StatusBadge status={group.status} />
        </div>
      </div>

      {/* Score row */}
      <div style={{ display: 'flex', alignItems: 'flex-end', justifyContent: 'space-between', marginBottom: '10px' }}>
        <ScoreLabel score={group.route_score} />
        <div style={{ textAlign: 'right' }}>
          <div style={{ fontSize: '11px', color: 'var(--dark-gray)', marginBottom: '2px' }}>Arrive by</div>
          <div style={{
            fontFamily: "'JetBrains Mono', 'Courier New', monospace",
            fontSize: '13px',
            color: 'var(--silver)',
            fontWeight: 400,
          }}>
            {fmtTime(group.arrive_by)}
          </div>
        </div>
      </div>

      <RouteScoreBar score={group.route_score} />

      {/* Footer */}
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginTop: '14px' }}>
        {memberNames.length === 0 ? (
          <span style={{ fontSize: '11px', color: 'var(--dark-gray)' }}>No members yet</span>
        ) : (
          <div style={{ display: 'flex' }}>
            {memberNames.slice(0, 5).map((name, i) => (
              <div key={`${group.id}-${name}`} title={name} style={{
                width: '24px',
                height: '24px',
                borderRadius: '9999px',
                background: 'rgba(161,164,165,0.14)',
                border: '1.5px solid var(--void)',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                fontFamily: "'JetBrains Mono', 'Courier New', monospace",
                fontSize: '8px',
                fontWeight: 500,
                color: 'var(--silver)',
                marginLeft: i > 0 ? '-7px' : 0,
                letterSpacing: 0,
                zIndex: 5 - i,
                position: 'relative',
              }}>
                {initials(name)}
              </div>
            ))}
            {memberNames.length > 5 && (
              <div style={{
                width: '24px',
                height: '24px',
                borderRadius: '9999px',
                background: 'rgba(161,164,165,0.10)',
                border: '1.5px solid var(--void)',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                fontSize: '9px',
                color: 'var(--dark-gray)',
                marginLeft: '-7px',
              }}>
                +{memberNames.length - 5}
              </div>
            )}
          </div>
        )}

        {driverName ? (
          <div style={{ display: 'flex', alignItems: 'center', gap: '5px' }}>
            <span style={{ width: '5px', height: '5px', borderRadius: '9999px', background: 'var(--green)', flexShrink: 0 }} />
            <span style={{ fontSize: '12px', color: 'var(--silver)' }}>{driverName}</span>
          </div>
        ) : (
          <span style={{ fontSize: '11px', color: 'var(--dark-gray)' }}>Unassigned</span>
        )}
      </div>
    </article>
  )
}
