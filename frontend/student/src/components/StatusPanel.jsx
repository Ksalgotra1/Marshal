import { CheckCircle2, Clock3, RadioTower, Route } from 'lucide-react'

const STEPS = [
  { key: 'pending', label: 'Requested', icon: Clock3 },
  { key: 'grouped', label: 'Grouped', icon: Route },
  { key: 'dispatching', label: 'Dispatching', icon: RadioTower },
  { key: 'assigned', label: 'Assigned', icon: CheckCircle2 },
]

function stepIndex(request, groupDetail) {
  if (groupDetail?.group?.status === 'assigned' || groupDetail?.group?.status === 'completed') return 3
  if (groupDetail?.group?.status === 'dispatching') return 2
  if (request?.group_id || groupDetail?.group) return 1
  if (request) return 0
  return -1
}

export function StatusPanel({ activeRequest, events, groupDetail }) {
  const activeStep = stepIndex(activeRequest, groupDetail)

  return (
    <section className="liquid-panel status-panel" id="status">
      <div className="panel-heading">
        <span className="accent-badge green">Live</span>
        <h2>Ride status</h2>
      </div>

      <div className="steps">
        {STEPS.map((step, index) => {
          const Icon = step.icon
          return (
            <div className={index <= activeStep ? 'step active' : 'step'} key={step.key}>
              <span><Icon size={16} /></span>
              <p>{step.label}</p>
            </div>
          )
        })}
      </div>

      <div className="event-feed">
        <p className="section-label">Recent signal</p>
        {events.length === 0 && <span className="muted">Waiting for live updates.</span>}
        {events.map(event => (
          <div className="event-row" key={event.id}>
            <span>{event.label}</span>
            <time>{new Date(event.at).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}</time>
          </div>
        ))}
      </div>
    </section>
  )
}
