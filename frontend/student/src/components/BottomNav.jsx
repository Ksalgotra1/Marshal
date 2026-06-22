import { Activity, MessageCircle, Route, Send } from 'lucide-react'

const NAV_ITEMS = [
  { id: 'request', label: 'Request', icon: Send },
  { id: 'groups', label: 'Groups', icon: Route },
  { id: 'status', label: 'Status', icon: Activity },
  { id: 'chat', label: 'Chat', icon: MessageCircle },
]

export function BottomNav({ activePage, onChange }) {
  return (
    <nav className="bottom-nav" aria-label="Student navigation">
      <div className="bottom-rail">
        {NAV_ITEMS.map(item => {
          const Icon = item.icon
          return (
            <button
              className={activePage === item.id ? 'active' : ''}
              type="button"
              aria-label={item.label}
              aria-current={activePage === item.id ? 'page' : undefined}
              key={item.id}
              onClick={() => onChange(item.id)}
            >
              <Icon size={21} />
              <span>{item.label}</span>
            </button>
          )
        })}
      </div>
    </nav>
  )
}
