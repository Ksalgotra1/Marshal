import { CarFront, Fingerprint, UsersRound } from 'lucide-react'

function shortId(id) {
  return id ? id.slice(0, 8) : 'Not set'
}

function formatDate(value) {
  if (!value) return 'Awaiting match'
  return new Date(value).toLocaleString([], {
    hour: '2-digit',
    minute: '2-digit',
    month: 'short',
    day: 'numeric',
  })
}

export function RideSummary({ activeRequest, groupDetail }) {
  const group = groupDetail?.group
  const members = groupDetail?.members || []

  return (
    <section className="liquid-panel ride-summary" id="summary">
      <div className="panel-heading">
        <span className="accent-badge yellow">Trip</span>
        <h2>Current ride</h2>
      </div>

      <div className="summary-grid">
        <div>
          <Fingerprint size={17} />
          <span>Request</span>
          <strong>{shortId(activeRequest?.id)}</strong>
        </div>
        <div>
          <UsersRound size={17} />
          <span>Group</span>
          <strong>{shortId(group?.id || activeRequest?.group_id)}</strong>
        </div>
        <div>
          <CarFront size={17} />
          <span>Depart</span>
          <strong>{formatDate(group?.expected_departure)}</strong>
        </div>
      </div>

      <div className="members">
        <p className="section-label">Riders</p>
        {members.length === 0 && <span className="muted">Riders appear after grouping.</span>}
        {members.map(member => (
          <span className="member-pill" key={member.request_id}>
            {member.requester_name || shortId(member.request_id)}
          </span>
        ))}
      </div>
    </section>
  )
}
