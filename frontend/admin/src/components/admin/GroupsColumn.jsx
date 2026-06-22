import { GroupCard } from '../shared/GroupCard'

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

export function GroupsColumn({ groups, groupMembers, groupDriverNames, hasMore, onLoadMore }) {
  const sorted = [...groups].sort((a, b) => Number(b.confidence_score) - Number(a.confidence_score))

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
            Groups Board
          </h2>
          <p style={{ fontSize: '12px', color: 'var(--dark-gray)', fontWeight: 400 }}>
            Ordered by confidence score
          </p>
        </div>
        {sorted.length > 0 && (
          <span style={{
            background: 'var(--green-dim)',
            color: 'var(--green)',
            fontSize: '11px',
            fontWeight: 500,
            padding: '3px 10px',
            borderRadius: '9999px',
          }}>
            {sorted.length}
          </span>
        )}
      </div>

      <div style={{ flex: 1, overflowY: 'auto', maxHeight: '380px', padding: '12px' }}>
        {sorted.length === 0 ? (
          <div style={{ padding: '40px 6px', textAlign: 'center' }}>
            <p style={{ fontSize: '13px', color: 'var(--dark-gray)' }}>No groups formed yet</p>
          </div>
        ) : (
          <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
            {sorted.map(g => (
              <GroupCard
                key={g.id}
                group={g}
                memberNames={groupMembers[g.id] || []}
                driverName={groupDriverNames[g.id]}
              />
            ))}
          </div>
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
