import { NavLink } from 'react-router-dom'

export function TopNavBar() {
  return (
    <nav className="hidden md:flex justify-between items-center w-full px-6 lg:px-12 py-4 max-w-[1600px] mx-auto bg-background z-50 relative">
      {/* Logo */}
      <NavLink to="/" className="flex items-center gap-2 no-underline">
        <span className="material-symbols-outlined text-primary text-3xl filled">local_taxi</span>
        <span className="font-headline text-2xl font-bold text-primary">Marshal</span>
      </NavLink>

      {/* Nav links (centered horizontally across the header) */}
      <div className="absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 flex gap-8 items-center">
        <NavLink
          to="/"
          end
          className={({ isActive }) =>
            isActive
              ? 'font-label text-base font-bold text-primary border-b-2 border-primary pb-1 flex items-center gap-2 transition-all'
              : 'font-label text-base font-medium text-on-surface-variant hover:bg-surface-container-low transition-colors rounded-lg px-3 py-1 flex items-center gap-2'
          }
        >
          <span className="material-symbols-outlined icon-sm">home</span>
          Home
        </NavLink>
        <NavLink
          to="/explore"
          className={({ isActive }) =>
            isActive
              ? 'font-label text-base font-bold text-primary border-b-2 border-primary pb-1 flex items-center gap-1.5 transition-all'
              : 'font-label text-base font-medium text-on-surface-variant hover:bg-surface-container-low transition-colors rounded-lg px-3 py-1 flex items-center gap-1.5'
          }
        >
          <span className="material-symbols-outlined icon-sm">explore</span>
          Explore
        </NavLink>
        <NavLink
          to="/groups"
          className={({ isActive }) =>
            isActive
              ? 'font-label text-base font-bold text-primary border-b-2 border-primary pb-1 flex items-center gap-1.5 transition-all'
              : 'font-label text-base font-medium text-on-surface-variant hover:bg-surface-container-low transition-colors rounded-lg px-3 py-1 flex items-center gap-1.5'
          }
        >
          <span className="material-symbols-outlined icon-sm">group</span>
          Groups
        </NavLink>
        <NavLink
          to="/status"
          className={({ isActive }) =>
            isActive
              ? 'font-label text-base font-bold text-primary border-b-2 border-primary pb-1 flex items-center gap-1.5 transition-all'
              : 'font-label text-base font-medium text-on-surface-variant hover:bg-surface-container-low transition-colors rounded-lg px-3 py-1 flex items-center gap-1.5'
          }
        >
          <span className="material-symbols-outlined icon-sm">sync_saved_locally</span>
          Status
        </NavLink>
      </div>
    </nav>
  )
}
