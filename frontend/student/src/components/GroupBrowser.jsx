import { UsersRound } from 'lucide-react'

function formatTime(value) {
  if (!value) return 'Pending'
  return new Date(value).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}

export function GroupBrowser({ activeRequest, groups, isBusy, onJoin }) {
  return (
    <section className="liquid-panel" id="groups">
      <div className="panel-heading split">
        <div>
          <span className="accent-badge blue">Open groups</span>
          <h2>Compatible rides</h2>
        </div>
        <span className="count-pill">{groups.length}</span>
      </div>

      <div className="group-list">
        {groups.length === 0 && (
          <div className="empty-state">
            <UsersRound size={22} />
            <p>No open groups yet. Marshal will surface one when routes line up.</p>
          </div>
        )}

        {groups.map(group => (
          <article className="group-card" key={group.id}>
            <div>
              <p className="mono-id">{group.id.slice(0, 8)}</p>
              <h3>{group.status}</h3>
              <span>Arrive by {formatTime(group.arrive_by)}</span>
            </div>

            <button
              className="pill ghost"
              type="button"
              disabled={!activeRequest || isBusy || activeRequest.group_id === group.id}
              onClick={() => onJoin(group.id)}
            >
              {activeRequest?.group_id === group.id ? 'Joined' : 'Join'}
            </button>
          </article>
        ))}
      </div>
    </section>
  )
}
