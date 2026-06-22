import { Bot, MessageCircle, SendHorizontal, UserRound } from 'lucide-react'

const PLACEHOLDER_MESSAGES = [
  {
    id: 'system-1',
    from: 'system',
    body: 'Driver chat will appear here after a group is assigned.',
    at: 'Pending',
  },
  {
    id: 'driver-1',
    from: 'driver',
    body: 'Placeholder: I am near the pickup gate. Share the exact spot when ready.',
    at: 'Telegram relay',
  },
  {
    id: 'student-1',
    from: 'student',
    body: 'Placeholder: Waiting at the marked pickup point.',
    at: 'Draft',
  },
]

export function DriverChatPanel({ activeRequest, groupDetail }) {
  const assigned = groupDetail?.group?.status === 'assigned'

  return (
    <section className="liquid-panel driver-chat-panel" id="chat">
      <div className="panel-heading split">
        <div>
          <span className="accent-badge blue">Driver</span>
          <h2>Chat</h2>
        </div>
        <span className={assigned ? 'chat-state online' : 'chat-state'}>{assigned ? 'Live' : 'Soon'}</span>
      </div>

      <div className="chat-thread">
        {PLACEHOLDER_MESSAGES.map(message => {
          const Icon = message.from === 'driver' ? UserRound : message.from === 'system' ? Bot : MessageCircle
          return (
            <article className={`chat-message ${message.from}`} key={message.id}>
              <span className="chat-avatar"><Icon size={15} /></span>
              <div>
                <p>{message.body}</p>
                <time>{message.at}</time>
              </div>
            </article>
          )
        })}
      </div>

      <div className="chat-compose">
        <input
          value={activeRequest ? 'Telegram bot integration pending' : 'Create a ride request first'}
          readOnly
          aria-label="Driver chat message"
        />
        <button type="button" disabled aria-label="Send driver message">
          <SendHorizontal size={17} />
        </button>
      </div>
    </section>
  )
}
