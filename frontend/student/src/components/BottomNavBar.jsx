import { NavLink } from 'react-router-dom'

const NAV_ITEMS = [
  { to: '/',        end: true,  icon: 'home',               label: 'Home'   },
  { to: '/explore', end: false, icon: 'explore',            label: 'Explore' },
  { to: '/groups',  end: false, icon: 'group',              label: 'Groups' },
  { to: '/status',  end: false, icon: 'sync_saved_locally', label: 'Status' },
  { to: '/chat',    end: false, icon: 'chat_bubble',        label: 'Chat'   },
]

export function BottomNavBar() {
  return (
    <nav
      className="md:hidden fixed bottom-0 left-0 w-full z-50 flex justify-around items-center px-1 pb-5 pt-3 bg-surface-container-low rounded-t-xl"
      style={{ boxShadow: '0 -4px 20px rgba(46,50,48,0.06)' }}
      aria-label="Student navigation"
    >
      {NAV_ITEMS.map(({ to, end, icon, label }) => (
        <NavLink
          key={to}
          to={to}
          end={end}
          className={({ isActive }) =>
            isActive
              ? 'flex min-w-0 flex-col items-center justify-center bg-primary-container text-on-primary-container rounded-xl px-3 py-2 active:scale-90 transition-transform duration-200'
              : 'flex min-w-0 flex-col items-center justify-center text-on-surface-variant px-3 py-2 hover:text-primary transition-all active:scale-90 duration-200'
          }
          aria-label={label}
        >
          {({ isActive }) => (
            <>
              <span className={`material-symbols-outlined icon-sm${isActive ? ' fill-icon' : ''}`}>
                {icon}
              </span>
              <span className={`font-label text-xs tracking-wide mt-1${isActive ? ' font-bold' : ''}`}>
                {label}
              </span>
            </>
          )}
        </NavLink>
      ))}
    </nav>
  )
}
