import { GroupChatPanel } from './GroupChatPanel.jsx'
import { Link } from 'react-router-dom'

// Ride lifecycle steps
const STEPS = [
  { key: 'pending',     label: 'Requested',   timestamp: null },
  { key: 'grouped',     label: 'Grouped',     timestamp: null },
  { key: 'dispatching', label: 'Dispatching', timestamp: null },
  { key: 'assigned',    label: 'Arriving',    timestamp: null },
]

function stepIndex(request, groupDetail) {
  if (groupDetail?.group?.status === 'assigned' || groupDetail?.group?.status === 'completed') return 3
  if (groupDetail?.group?.status === 'dispatching') return 2
  if (request?.group_id || groupDetail?.group) return 1
  if (request) return 0
  return -1
}

function formatTime(iso) {
  if (!iso) return null
  return new Date(iso).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}

function etaMinutes(arriveBy) {
  if (!arriveBy) return null
  const diff = new Date(arriveBy) - Date.now()
  if (diff <= 0) return null
  return Math.round(diff / 60000)
}

// SVG Winding Path — 4 nodes on a cubic Bézier
function WindingPathSVG({ activeStep }) {
  const nodes = [
    { x: 60,  y: 150, labelY: 35,  align: 'middle' },
    { x: 270, y: 95,  labelY: -30, align: 'middle' },
    { x: 480, y: 155, labelY: 35,  align: 'middle' },
    { x: 700, y: 95,  labelY: -30, align: 'middle' },
  ]

  const fullPath   = 'M 60,150 C 150,40 180,190 270,95 C 360,0 390,210 480,155 C 570,100 630,40 700,95'
  const activePaths = [
    '',
    'M 60,150 C 150,40 180,190 270,95',
    'M 60,150 C 150,40 180,190 270,95 C 360,0 390,210 480,155',
    'M 60,150 C 150,40 180,190 270,95 C 360,0 390,210 480,155 C 570,100 630,40 700,95',
  ]

  return (
    <svg
      viewBox="0 0 760 240"
      preserveAspectRatio="xMidYMid meet"
      className="w-full h-full"
    >
      {/* Background track */}
      <path
        d={fullPath}
        fill="none"
        stroke="#c4c8bc"
        strokeWidth="7"
        strokeLinecap="round"
      />

      {/* Animated active progress */}
      {activeStep >= 0 && (
        <path
          className="path-line"
          d={activePaths[Math.min(activeStep + 1, 3)]}
          fill="none"
          stroke="#4a7c59"
          strokeWidth="7"
          strokeLinecap="round"
        />
      )}

      {/* Nodes */}
      {STEPS.map((step, i) => {
        const node     = nodes[i]
        const isPast   = i < activeStep
        const isCurrent = i === activeStep
        const isFuture = i > activeStep

        return (
          <g key={step.key} transform={`translate(${node.x}, ${node.y})`}>
            {/* Pulse ring for current node */}
            {isCurrent && (
              <circle className="node-pulse" cx="0" cy="0" r="26" fill="#78a886" />
            )}

            {/* Main node circle */}
            {isFuture ? (
              <circle cx="0" cy="0" r="14" fill="#faf6f0" stroke="#c4c8bc" strokeWidth="3.5" />
            ) : (
              <circle cx="0" cy="0" r="14" fill="#4a7c59" />
            )}

            {/* Inner dot for current node */}
            {isCurrent && <circle cx="0" cy="0" r="5" fill="#ffffff" />}

            {/* Completed checkmark */}
            {isPast && (
              <text x="0" y="5" textAnchor="middle" fill="#ffffff" fontSize="14" fontFamily="Material Symbols Outlined">
                check
              </text>
            )}

            {/* Step label */}
            <text
              x="0"
              y={node.labelY}
              textAnchor={node.align}
              fill={isCurrent ? '#4a7c59' : isFuture ? '#74796e' : '#4a4e4a'}
              fontSize={isCurrent ? '14' : '12'}
              fontWeight={isCurrent ? '700' : '600'}
              fontFamily="Nunito Sans, sans-serif"
            >
              {step.label}
            </text>

            {/* Timestamp (placeholder) */}
            {(isPast || isCurrent) && (
              <text
                x="0"
                y={node.labelY + (node.labelY < 0 ? -16 : 18)}
                textAnchor={node.align}
                fill="#74796e"
                fontSize="10"
                fontFamily="Nunito Sans, sans-serif"
              >
                {isCurrent ? 'Now' : ''}
              </text>
            )}
          </g>
        )
      })}
    </svg>
  )
}

export function StatusPage({ activeRequest, groupDetail, events, messages, isChatSending, sendChatMessage }) {
  const activeStep  = stepIndex(activeRequest, groupDetail)
  const eta         = etaMinutes(activeRequest?.arrive_by)
  const groupName   = groupDetail?.group?.id ? `Group ${groupDetail.group.id.slice(0, 6).toUpperCase()}` : null
  const driverName  = groupDetail?.group?.driver_id ? `Driver assigned` : null
  const memberCount = groupDetail?.members?.length || 0
  const destination = activeRequest?.dropoff_lat
    ? `${Number(activeRequest.dropoff_lat).toFixed(4)}, ${Number(activeRequest.dropoff_lng).toFixed(4)}`
    : 'Campus'

  return (
    <div className="flex-grow w-full max-w-7xl mx-auto px-4 md:px-6 py-6 md:py-10 grid grid-cols-1 lg:grid-cols-12 gap-8 pb-32 md:pb-10">

      {/* ── Left / Centre: Status Tracker ──────────────────── */}
      <section className="lg:col-span-8 flex flex-col gap-8">

        {/* No active request */}
        {!activeRequest && (
          <div className="relative overflow-hidden rounded-[2rem] border border-outline-variant/20 bg-surface-container-low p-8 shadow-soft md:p-12">
            <div className="pointer-events-none absolute -right-16 -top-16 h-64 w-64 rounded-full bg-primary-container/30 blur-3xl" />
            <div className="relative mx-auto flex max-w-xl flex-col items-center text-center">
              <span className="material-symbols-outlined fill-icon rounded-full bg-primary-container p-5 text-4xl text-on-primary-container">directions_car</span>
              <span className="mt-6 font-label text-xs uppercase tracking-[.14em] text-primary">Ride status</span>
              <h1 className="font-headline mt-3 text-4xl font-semibold text-on-surface md:text-5xl">Your ride timeline will appear here.</h1>
              <p className="mt-4 max-w-md leading-relaxed text-on-surface-variant">Once a request is active, Marshal will show its grouping, dispatch, and arrival status in real time.</p>
              <Link to="/explore" className="mt-7 inline-flex items-center gap-2 rounded-xl bg-primary px-6 py-3 font-label font-semibold text-on-primary transition hover:bg-on-primary-fixed-variant">Request a ride <span className="material-symbols-outlined icon-sm">arrow_forward</span></Link>
            </div>
            <div className="relative mt-12 grid grid-cols-4 gap-2 rounded-2xl border border-outline-variant/20 bg-surface-bright p-5 md:p-7">
              {STEPS.map((step, index) => <div key={step.key} className="flex flex-col items-center text-center"><span className={`flex h-9 w-9 items-center justify-center rounded-full border-2 ${index === 0 ? 'border-primary bg-primary-container text-primary' : 'border-outline-variant bg-surface text-outline'}`}><span className="text-xs font-bold">{index + 1}</span></span><span className="mt-3 text-xs font-semibold text-on-surface-variant">{step.label}</span></div>)}
            </div>
          </div>
        )}

        {/* Hero status area */}
        {activeRequest && (
          <>
            <div className="bg-surface-container-low rounded-[2rem] p-8 md:p-12 relative overflow-hidden"
              style={{ boxShadow: '0 4px 20px rgba(46,50,48,0.06)' }}>
              {/* Radial gradient overlay */}
              <div className="absolute inset-0 pointer-events-none"
                style={{ background: 'radial-gradient(circle at top right, rgba(74,124,89,0.12) 0%, transparent 55%)' }} />

              <h1 className="font-headline text-3xl md:text-5xl text-on-background font-bold mb-2 relative z-10">
                {activeStep >= 3 ? 'Arriving Soon' : activeStep >= 2 ? 'Driver Dispatched' : activeStep >= 1 ? 'Group Formed' : 'Finding Your Group'}
              </h1>

              <p className="text-on-surface-variant text-lg md:text-xl mb-8 flex items-center gap-2 relative z-10">
                <span className="material-symbols-outlined text-primary icon-sm">schedule</span>
                {activeStep >= 3
                  ? <><strong className="text-primary ml-1">Driver en route</strong></>
                  : activeRequest?.arrive_by
                    ? <>Arrive by <strong className="text-primary ml-1">{new Date(activeRequest.arrive_by).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}</strong></>
                    : <>Heading to <strong className="text-primary ml-1">{destination}</strong></>
                }
              </p>

              {/* Winding Path SVG */}
              <div className="relative w-full h-48 md:h-64 bg-surface-bright rounded-2xl p-4 flex items-center justify-center border border-outline-variant/30 relative z-10">
                <WindingPathSVG activeStep={activeStep} />
              </div>
            </div>

            {/* Bento detail cards */}
            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              {/* Vehicle card */}
              <div className="bg-surface-container-highest rounded-3xl p-6 flex items-center gap-6 border border-outline-variant/20 transition-transform hover:-translate-y-1 duration-300"
                style={{ boxShadow: '0 2px 8px rgba(46,50,48,0.05)' }}>
                <div className="w-16 h-16 rounded-full bg-surface-bright flex items-center justify-center text-primary">
                  <span className="material-symbols-outlined fill-icon icon-lg">directions_car</span>
                </div>
                <div>
                  <p className="text-on-surface-variant text-xs font-semibold uppercase tracking-wider mb-1 font-label">Vehicle</p>
                  <p className="text-on-background font-headline text-xl font-bold">
                    {driverName || 'Pending…'}
                  </p>
                  <p className="text-outline text-sm font-body">
                    {activeStep >= 2 ? 'Driver en route' : 'Matching with nearest van'}
                  </p>
                </div>
              </div>

              {/* Group card */}
              <div className="bg-secondary-container rounded-3xl p-6 flex items-center gap-6 border border-outline-variant/20 transition-transform hover:-translate-y-1 duration-300"
                style={{ boxShadow: '0 2px 8px rgba(46,50,48,0.05)' }}>
                <div className="w-16 h-16 rounded-full bg-surface-bright flex items-center justify-center text-secondary">
                  <span className="material-symbols-outlined fill-icon icon-lg">group</span>
                </div>
                <div>
                  <p className="text-on-secondary-container text-xs font-semibold uppercase tracking-wider mb-1 font-label">Your Group</p>
                  <p className="text-on-secondary-container font-headline text-xl font-bold">
                    {groupName || 'Forming…'}
                  </p>
                  <p className="text-secondary text-sm font-body">
                    {memberCount > 0 ? `${memberCount} rider${memberCount !== 1 ? 's' : ''} matched` : 'Looking for riders nearby'}
                  </p>
                </div>
              </div>
            </div>

            {/* Live event feed */}
            {events.length > 0 && (
              <div className="bg-surface-container-low rounded-2xl p-6 border border-outline-variant/20">
                <p className="text-xs font-semibold uppercase tracking-wider text-on-surface-variant mb-4 font-label">
                  Recent signal
                </p>
                <div className="space-y-3">
                  {events.map(event => (
                    <div key={event.id} className="flex items-center justify-between text-sm">
                      <div className="flex items-center gap-2">
                        <span className="w-1.5 h-1.5 rounded-full bg-primary inline-block" />
                        <span className="text-on-surface font-body capitalize">{event.label}</span>
                      </div>
                      <time className="text-outline text-xs font-label">
                        {formatTime(event.at)}
                      </time>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </>
        )}
      </section>

      {/* ── Right Column: Group Chat (desktop sidebar) ──────── */}
      <aside className="lg:col-span-4 flex flex-col h-[600px] lg:h-auto">
        <GroupChatPanel
          groupDetail={groupDetail}
          messages={messages}
          isChatSending={isChatSending}
          sendChatMessage={sendChatMessage}
          currentUserName={activeRequest?.requester_name}
        />
      </aside>
    </div>
  )
}
