import { useEffect, useState } from 'react'
import { getAreaName, getInitials } from '../utils/locationUtils.js'

const STATUS_LABEL = {
  grouped:     { label: 'Open',        bg: 'bg-primary-container/60', text: 'text-on-primary-container' },
  dispatching: { label: 'Dispatching', bg: 'bg-secondary-container', text: 'text-on-secondary-container' },
  assigned:    { label: 'Assigned',    bg: 'bg-tertiary-container',  text: 'text-on-tertiary-container' },
  completed:   { label: 'Completed',   bg: 'bg-surface-variant shadow-none', text: 'text-on-surface-variant' },
}

function DestinationBadge({ lat, lng }) {
  const [areaName, setAreaName] = useState('Campus Zone')

  useEffect(() => {
    if (!lat || !lng) return
    getAreaName(lat, lng).then(name => setAreaName(name))
  }, [lat, lng])

  return (
    <div className="flex items-center gap-2 text-xs text-on-surface-variant font-label bg-surface-bright px-3.5 py-2.5 rounded-xl border border-outline-variant/20">
      <span className="material-symbols-outlined text-tertiary icon-sm">location_on</span>
      <span>Heading to: <strong className="text-on-surface font-semibold">{areaName}</strong></span>
    </div>
  )
}

export function GroupsPage({ activeRequest, groups = [], isBusy, onJoin }) {
  const canJoin = Boolean(activeRequest && !activeRequest.group_id)

  return (
    <div className="max-w-[1600px] mx-auto px-6 lg:px-12 py-10 md:py-14 pb-32 md:pb-14 w-full">

      {/* Header */}
      <div className="mb-10 max-w-2xl">
        <span className="font-label text-xs uppercase tracking-[.12em] text-primary">Ride groups</span>
        <h1 className="font-headline text-4xl md:text-5xl font-semibold text-on-background mt-2 mb-2">
          Open Groups
        </h1>
        <p className="text-on-surface-variant font-body text-base">
          Browse and join groups heading your way.
          {activeRequest && !activeRequest.group_id && (
            <span className="ml-2 inline-flex items-center gap-1 text-primary font-medium text-sm">
              <span className="material-symbols-outlined icon-sm">info</span>
              Your request is active — you can join a group below.
            </span>
          )}
        </p>
      </div>

      {/* Empty state */}
      {groups.length === 0 && (
        <div className="rounded-[2rem] border border-outline-variant/20 bg-surface-container-low px-6 py-20 flex flex-col items-center justify-center gap-4 text-center shadow-soft">
          <span className="material-symbols-outlined fill-icon rounded-full bg-primary-container p-5 text-4xl text-on-primary-container">group_work</span>
          <h2 className="font-headline text-2xl text-on-surface">No open groups yet</h2>
          <p className="text-on-surface-variant font-body max-w-sm">
            Request a ride and the H3 grouper will cluster you with nearby riders automatically.
          </p>
        </div>
      )}

      {/* Groups grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 2xl:grid-cols-4 gap-6">
        {groups.map(item => {
          const group   = item.group || item
          const members = item.members || group.members || []
          const score   = Number(group.route_score || 0)
          const status  = STATUS_LABEL[group.status] || STATUS_LABEL.grouped
          const isJoined = activeRequest?.group_id === group.id
          const firstMember = members[0]

          return (
            <div
              key={group.id}
              className="bg-surface-container-low rounded-3xl p-6 border border-outline-variant/20 hover:shadow-medium hover:-translate-y-0.5 transition-all duration-300 flex min-h-[18rem] flex-col justify-between gap-5"
              style={{ boxShadow: '0 4px 20px rgba(46,50,48,0.06)' }}
            >
              <div className="space-y-5">
                {/* Card header */}
                <div className="flex items-start justify-between gap-3">
                  <div className="flex items-center gap-3">
                    <div className="w-12 h-12 rounded-2xl bg-primary-container text-on-primary-container flex items-center justify-center">
                      <span className="material-symbols-outlined fill-icon">group</span>
                    </div>
                    <div>
                      <p className="font-headline font-semibold text-on-surface text-base leading-tight">
                        Group {group.id.slice(0, 6).toUpperCase()}
                      </p>
                      <div className="flex items-center gap-1.5 mt-1">
                        <span className="material-symbols-outlined text-on-surface-variant icon-sm">group</span>
                        <span className="text-xs text-on-surface-variant font-label">
                          {members.length > 0 ? `${members.length} rider${members.length !== 1 ? 's' : ''}` : 'Forming…'}
                        </span>
                      </div>
                    </div>
                  </div>

                  {/* Status badge */}
                  <span className={`text-xs font-label font-semibold px-3 py-1 rounded-full flex-shrink-0 ${status.bg} ${status.text}`}>
                    {status.label}
                  </span>
                </div>

                {/* Destination info */}
                {firstMember?.dropoff_lat && (
                  <DestinationBadge lat={firstMember.dropoff_lat} lng={firstMember.dropoff_lng} />
                )}

                {/* Route score */}
                <div className="flex items-center gap-3">
                  <div className="flex-1 h-1.5 bg-surface-container-highest rounded-full overflow-hidden">
                    <div
                      className="h-full bg-primary rounded-full transition-all duration-500"
                      style={{ width: `${Math.min((score / 60) * 100, 100).toFixed(0)}%` }}
                    />
                  </div>
                  <span className="text-sm font-label font-bold text-primary tabular-nums">
                    {score.toFixed(2)}
                  </span>
                  <span className="text-xs text-on-surface-variant font-label">score</span>
                </div>
              </div>

              {/* Member list - Initials only (Privacy Protected) */}
              <div className="pt-2 border-t border-outline-variant/15 flex items-center justify-between">
                {members.length > 0 ? (
                  <div className="flex -space-x-2">
                    {members.slice(0, 5).map((m, i) => {
                      const badgeStyles = [
                        'bg-primary-container text-on-primary-container',
                        'bg-secondary-container text-on-secondary-container',
                        'bg-tertiary-container text-on-tertiary-container',
                        'bg-surface-container-highest text-on-surface-variant',
                      ]
                      return (
                        <div
                          key={m.id || i}
                          title="Rider (Initials only for privacy)"
                          className={`w-8 h-8 rounded-full border-2 border-background flex items-center justify-center font-bold text-[11px] shadow-sm ${badgeStyles[i % badgeStyles.length]}`}
                        >
                          {getInitials(m.requester_name)}
                        </div>
                      )
                    })}
                    {members.length > 5 && (
                      <div className="w-8 h-8 rounded-full border-2 border-background bg-surface-container-highest text-on-surface flex items-center justify-center font-bold text-xs">
                        +{members.length - 5}
                      </div>
                    )}
                  </div>
                ) : (
                  <span className="text-xs text-on-surface-variant font-label">Waiting for riders…</span>
                )}

                {/* Action button */}
                {isJoined ? (
                  <span className="text-xs font-label font-semibold text-primary flex items-center gap-1">
                    <span className="material-symbols-outlined icon-sm">check_circle</span> Joined
                  </span>
                ) : (
                  <button
                    type="button"
                    disabled={!canJoin || isBusy}
                    onClick={() => onJoin(group.id)}
                    className="text-xs font-headline font-bold bg-primary hover:bg-primary/90 text-on-primary px-4 py-2 rounded-xl transition-all active:scale-95 disabled:opacity-40 disabled:cursor-not-allowed shadow-sm"
                  >
                    Join Group
                  </button>
                )}
              </div>
            </div>
          )
        })}
      </div>
    </div>
  )
}
