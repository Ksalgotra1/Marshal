import { useRef, useState, useEffect } from 'react'

const AVATAR_COLORS = [
  { bg: 'bg-primary-container',   text: 'text-on-primary-container'   },
  { bg: 'bg-secondary-container', text: 'text-on-secondary-container' },
  { bg: 'bg-surface-container-highest', text: 'text-on-surface-variant' },
]

function avatarStyle(name) {
  let hash = 0
  for (let i = 0; i < (name || '').length; i++) hash = name.charCodeAt(i) + ((hash << 5) - hash)
  return AVATAR_COLORS[Math.abs(hash) % AVATAR_COLORS.length]
}

function initials(name) {
  if (!name) return '?'
  return name.split(' ').map(w => w[0]).join('').slice(0, 2).toUpperCase()
}

function formatTime(iso) {
  if (!iso) return ''
  return new Date(iso).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}

export function GroupChatPanel({ groupDetail, messages, isChatSending, sendChatMessage, currentUserName }) {
  const [input, setInput] = useState('')
  const messagesEndRef    = useRef(null)
  const hasGroup          = Boolean(groupDetail?.group)
  const memberCount       = groupDetail?.members?.length || 0

  // Auto-scroll to bottom on new messages
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages])

  async function handleSend(e) {
    e.preventDefault()
    const text = input.trim()
    if (!text || isChatSending || !hasGroup) return
    setInput('')
    await sendChatMessage(text)
  }

  return (
    <div className="bg-surface-bright rounded-[2rem] border border-outline-variant/20 flex flex-col flex-grow overflow-hidden h-full shadow-[0_4px_20px_rgba(46,50,48,0.06)]">
      {/* Chat Header */}
      <div className="p-6 border-b border-outline-variant/20 bg-surface-container-low flex justify-between items-center flex-shrink-0">
        <div>
          <h2 className="font-headline text-xl text-on-background font-bold">Group Chat</h2>
          <p className="text-sm text-primary flex items-center gap-1.5 mt-1 font-label">
            {hasGroup ? (
              <>
                <span className="w-2 h-2 rounded-full bg-primary animate-pulse inline-block" />
                {memberCount} rider{memberCount !== 1 ? 's' : ''} in group
              </>
            ) : (
              <>
                <span className="w-2 h-2 rounded-full bg-outline-variant inline-block" />
                Waiting for group…
              </>
            )}
          </p>
        </div>
      </div>

      {/* Messages Area */}
      <div className="flex-grow p-6 overflow-y-auto flex flex-col gap-4 bg-background">
        {/* No group yet */}
        {!hasGroup && (
          <div className="flex flex-col items-center justify-center h-full text-center gap-3 py-12">
            <div className="w-14 h-14 rounded-full bg-surface-container-low flex items-center justify-center text-outline">
              <span className="material-symbols-outlined text-3xl">chat_bubble</span>
            </div>
            <p className="text-on-surface-variant text-sm font-body max-w-xs leading-relaxed">
              Group chat becomes available once you&apos;re matched into a group.
            </p>
          </div>
        )}

        {/* Empty chat */}
        {hasGroup && messages.length === 0 && (
          <div className="flex justify-center my-4">
            <span className="bg-surface-container-low border border-outline-variant/30 text-on-surface-variant text-xs px-4 py-1.5 rounded-full font-medium font-label">
              Group chat started
            </span>
          </div>
        )}

        {/* Message list */}
        {messages.map((msg, i) => {
          const isSystem = !msg.sender_name
          if (isSystem) {
            return (
              <div key={msg.id || i} className="flex justify-center my-2">
                <span className="bg-surface-container-high text-on-surface-variant text-xs px-3.5 py-1 rounded-full font-medium font-label">
                  {msg.content}
                </span>
              </div>
            )
          }

          const isDriver = msg.sender_type === 'driver' || msg.is_driver || (msg.sender_name || '').toLowerCase().includes('driver')
          const isSelf   = currentUserName && (msg.sender_name || '').toLowerCase() === currentUserName.toLowerCase()
          const { bg: defaultBg, text: defaultTextColor } = avatarStyle(msg.sender_name)

          // Bubble styling based on sender type
          let bubbleClasses = 'bg-surface-container-high text-on-surface rounded-2xl rounded-tl-xs'
          let avatarBg = defaultBg
          let avatarText = defaultTextColor

          if (isSelf) {
            bubbleClasses = 'bg-primary text-on-primary rounded-2xl rounded-tr-xs'
            avatarBg = 'bg-primary-container'
            avatarText = 'text-on-primary-container'
          } else if (isDriver) {
            bubbleClasses = 'bg-tertiary-container text-on-tertiary-container border border-tertiary/20 rounded-2xl rounded-tl-xs'
            avatarBg = 'bg-tertiary'
            avatarText = 'text-on-tertiary'
          }

          return (
            <div
              key={msg.id || i}
              className={`flex gap-3 max-w-[85%] ${isSelf ? 'ml-auto flex-row-reverse' : ''}`}
            >
              {/* Avatar circle with initials */}
              <div className={`w-8 h-8 rounded-full ${avatarBg} ${avatarText} flex-shrink-0 flex items-center justify-center font-bold text-xs shadow-sm`}>
                {isDriver ? (
                  <span className="material-symbols-outlined icon-sm">local_taxi</span>
                ) : (
                  initials(msg.sender_name)
                )}
              </div>

              {/* Message content & footer metadata */}
              <div className={isSelf ? 'text-right' : ''}>
                <div className={`p-3.5 text-sm font-body break-words whitespace-pre-wrap shadow-sm ${bubbleClasses}`}>
                  {msg.content}
                </div>
                <div className={`text-[11px] text-outline mt-1 font-label flex items-center gap-1.5 ${isSelf ? 'justify-end pr-1' : 'pl-1'}`}>
                  <span className="font-semibold text-on-surface-variant">{msg.sender_name || 'Rider'}</span>
                  <span>•</span>
                  <span>{formatTime(msg.created_at)}</span>
                </div>
              </div>
            </div>
          )
        })}
        <div ref={messagesEndRef} />
      </div>

      {/* Chat Input */}
      <div className="p-4 bg-surface-container-low border-t border-outline-variant/20 flex-shrink-0">
        <form onSubmit={handleSend} className="relative flex items-center">
          <input
            className="w-full bg-surface-bright border border-outline-variant/40 rounded-full py-3.5 pl-5 pr-12 text-on-surface focus:outline-none focus:border-primary focus:ring-1 focus:ring-primary transition-colors text-sm font-body disabled:opacity-50 placeholder:text-on-surface-variant/50"
            placeholder={hasGroup ? 'Message group…' : 'Join a group to chat…'}
            value={input}
            onChange={e => setInput(e.target.value)}
            disabled={!hasGroup || isChatSending}
          />
          <button
            type="submit"
            disabled={!input.trim() || !hasGroup || isChatSending}
            className="absolute right-2 w-9 h-9 rounded-full bg-primary text-on-primary flex items-center justify-center hover:bg-on-primary-fixed-variant transition-all active:scale-95 disabled:opacity-40 disabled:cursor-not-allowed shadow-sm"
          >
            {isChatSending ? (
              <span className="material-symbols-outlined icon-sm animate-spin">progress_activity</span>
            ) : (
              <span className="material-symbols-outlined icon-sm">send</span>
            )}
          </button>
        </form>
      </div>
    </div>
  )
}
