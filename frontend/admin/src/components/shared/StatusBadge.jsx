const STATUSES = {
  pending:     { bg: 'rgba(255,196,61,0.14)',  text: '#ffc53d', label: 'Pending' },
  grouped:     { bg: 'var(--blue-dim)',         text: 'var(--blue)', label: 'Grouped' },
  dispatching: { bg: 'var(--blue-dim)',         text: 'var(--blue)', label: 'Dispatching' },
  assigned:    { bg: 'var(--blue-dim)',         text: 'var(--blue)', label: 'Assigned' },
  confirmed:   { bg: 'var(--green-mid)',        text: 'var(--green)', label: 'Confirmed' },
  arriving:    { bg: 'var(--green-mid)',        text: 'var(--green)', label: 'Arriving' },
  cancelled:   { bg: 'var(--red-dim)',          text: 'var(--red)', label: 'Cancelled' },
}

export function StatusBadge({ status }) {
  const s = STATUSES[status] ?? { bg: 'rgba(161,164,165,0.12)', text: 'var(--silver)', label: status }
  return (
    <span style={{
      display: 'inline-flex',
      alignItems: 'center',
      gap: '5px',
      background: s.bg,
      color: s.text,
      fontSize: '11px',
      fontWeight: 500,
      padding: '3px 10px',
      borderRadius: '9999px',
      letterSpacing: '0.01em',
      whiteSpace: 'nowrap',
    }}>
      <span style={{ width: '4px', height: '4px', borderRadius: '9999px', background: s.text, flexShrink: 0 }} />
      {s.label}
    </span>
  )
}
