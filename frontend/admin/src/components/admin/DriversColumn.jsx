import { DriverListItem } from './DriverListItem'

const loadMoreStyle = {
  width: '100%',
  border: '0',
  borderTop: '1px solid var(--frost-alt)',
  background: 'transparent',
  color: 'var(--silver)',
  fontSize: '12px',
  fontWeight: 500,
  padding: '11px 18px',
  cursor: 'pointer',
}

export function DriversColumn({ drivers, hasMore, onLoadMore }) {
  const online = drivers.filter(d => d.status === 'online').length
  const busy   = drivers.filter(d => d.status === 'busy').length

  return (
    <section style={{
      border: '1px solid var(--frost)',
      borderRadius: '16px',
      overflow: 'hidden',
      boxShadow: 'var(--ring)',
      display: 'flex',
      flexDirection: 'column',
    }}>
      <div style={{
        padding: '14px 18px',
        borderBottom: '1px solid var(--frost)',
      }}>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '2px' }}>
          <h2 style={{ fontSize: '14px', fontWeight: 500, color: 'var(--near-white)' }}>
            Drivers
          </h2>
          <div style={{ display: 'flex', gap: '5px' }}>
            {online > 0 && (
              <span style={{ background: 'var(--green-dim)', color: 'var(--green)', fontSize: '11px', fontWeight: 500, padding: '3px 9px', borderRadius: '9999px' }}>
                {online} online
              </span>
            )}
            {busy > 0 && (
              <span style={{ background: 'var(--orange-dim)', color: 'var(--orange)', fontSize: '11px', fontWeight: 500, padding: '3px 9px', borderRadius: '9999px' }}>
                {busy} busy
              </span>
            )}
          </div>
        </div>
        <p style={{ fontSize: '12px', color: 'var(--dark-gray)', fontWeight: 400 }}>
          Availability panel
        </p>
      </div>

      <div style={{ flex: 1, overflowY: 'auto', maxHeight: '380px' }}>
        {drivers.length === 0 ? (
          <div style={{ padding: '40px 18px', textAlign: 'center' }}>
            <p style={{ fontSize: '13px', color: 'var(--dark-gray)' }}>No drivers registered</p>
          </div>
        ) : (
          drivers.map(d => <DriverListItem key={d.id} driver={d} />)
        )}
      </div>

      {hasMore && (
        <button type="button" onClick={onLoadMore} style={loadMoreStyle}>
          Load more
        </button>
      )}
    </section>
  )
}
