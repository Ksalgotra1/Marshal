import { QueueRow } from './QueueRow'

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

export function QueueColumn({ requests, hasMore, onLoadMore }) {
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
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
      }}>
        <div>
          <h2 style={{ fontSize: '14px', fontWeight: 500, color: 'var(--near-white)', marginBottom: '2px' }}>
            Live Queue
          </h2>
          <p style={{ fontSize: '12px', color: 'var(--dark-gray)', fontWeight: 400 }}>
            Pending requests
          </p>
        </div>
        {requests.length > 0 && (
          <span style={{
            background: 'rgba(255,196,61,0.12)',
            color: 'var(--yellow)',
            fontSize: '11px',
            fontWeight: 500,
            padding: '3px 10px',
            borderRadius: '9999px',
          }}>
            {requests.length}
          </span>
        )}
      </div>

      <div style={{ flex: 1, overflowY: 'auto', maxHeight: '380px' }}>
        {requests.length === 0 ? (
          <div style={{ padding: '40px 18px', textAlign: 'center' }}>
            <p style={{ fontSize: '13px', color: 'var(--dark-gray)' }}>Queue is clear</p>
          </div>
        ) : (
          requests.map(r => <QueueRow key={r.id} request={r} />)
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
