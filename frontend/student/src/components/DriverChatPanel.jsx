import { useState } from 'react'
import { Bot, MessageCircle, SendHorizontal, UserRound } from 'lucide-react'

export function DriverChatPanel({ activeRequest, groupDetail, messages = [], isChatSending, sendChatMessage }) {
  const assigned = groupDetail?.group?.status === 'assigned'
  const [inputText, setInputText] = useState('')

  const handleSend = async (e) => {
    e.preventDefault()
    if (!inputText.trim() || !assigned || isChatSending) return
    const success = await sendChatMessage(inputText)
    if (success) {
      setInputText('')
    }
  }

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
        {(messages.length === 0 ? [{
          id: 'system-1',
          sender_type: 'system',
          content: 'Driver chat will appear here after a group is assigned.',
        }] : messages).map(message => {
          const fromType = message.sender_type || message.from // fallback
          const Icon = fromType === 'driver' ? UserRound : fromType === 'system' ? Bot : MessageCircle
          const timeLabel = message.created_at ? new Date(message.created_at).toLocaleTimeString([], {hour: '2-digit', minute:'2-digit'}) : ''
          
          return (
            <article className={`chat-message ${fromType}`} key={message.id}>
              <span className="chat-avatar"><Icon size={15} /></span>
              <div>
                <p>{message.content || message.body}</p>
                {timeLabel && <time>{timeLabel}</time>}
              </div>
            </article>
          )
        })}
      </div>

      <div className="chat-compose">
        <input
          value={assigned ? inputText : ''}
          placeholder={!assigned ? (activeRequest ? 'Waiting for assignment...' : 'Create a ride request first') : "Message driver..."}
          onChange={e => setInputText(e.target.value)}
          onKeyDown={e => {
            if (e.key === 'Enter') handleSend(e)
          }}
          readOnly={!assigned || isChatSending}
          aria-label="Driver chat message"
        />
        <button type="button" disabled={!assigned || isChatSending || !inputText.trim()} onClick={handleSend} aria-label="Send driver message">
          <SendHorizontal size={17} />
        </button>
      </div>
    </section>
  )
}
