import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
const campusMapImg = '/images/stitch-campus-map.jpg'

const FEATURES = [
  {
    icon: 'group_work',
    iconBg: 'bg-primary-container text-on-primary-container',
    title: 'Smart Grouping',
    body: 'H3 bucketing at resolution 9 (~0.1 km²) clusters your request with peers heading the same direction, reducing wait times and empty seats.',
  },
  {
    icon: 'verified_user',
    iconBg: 'bg-tertiary-container text-on-tertiary-container',
    title: 'Instant Dispatch',
    body: 'The assigner pulls the highest-scored group from a Postgres priority queue and dispatches it to verified drivers via Telegram.',
  },
  {
    icon: 'sensors',
    iconBg: 'bg-secondary-container text-on-secondary-container',
    title: 'Live Status',
    body: 'WebSocket and SSE fan out real-time updates to every connected student. Requested → Grouped → Dispatching → Assigned, streamed live.',
  },
]

const COMPARISON_OLD = [
  'Waiting outside with no ETA',
  'Uncoordinated pickup chaos',
  'No visibility into driver assignment',
]

const COMPARISON_NEW = [
  'H3 spatial matching groups you automatically',
  'Telegram-dispatched verified drivers',
  'Live ride status from request to arrival',
]

export function LandingPage() {
  const navigate = useNavigate()
  const [pickup, setPickup] = useState('Main Library')
  const [dropoff, setDropoff] = useState('')

  function handleFindRides(e) {
    e.preventDefault()
    navigate('/explore')
  }

  return (
    <div className="bg-background text-on-background font-body min-h-screen flex flex-col selection:bg-primary-container selection:text-on-primary-container">
      {/* Hero / Main Content */}
      <main className="flex-grow flex flex-col md:flex-row max-w-7xl mx-auto w-full px-6 pt-8 md:pt-12 pb-6 md:pb-10 gap-12 lg:gap-24 relative overflow-hidden">
        {/* Background organic shapes */}
        <div className="absolute -top-40 -left-40 w-96 h-96 bg-primary-container/30 rounded-full blur-3xl -z-10 mix-blend-multiply"></div>
        <div className="absolute top-40 right-0 w-[500px] h-[500px] bg-secondary-container/50 rounded-full blur-3xl -z-10 mix-blend-multiply"></div>

        {/* Left Column: Hero Text & Interaction */}
        <div className="flex-1 flex flex-col justify-center gap-8 z-10">
          <div className="space-y-4">
            <div className="inline-flex items-center gap-2 px-3 py-1 bg-surface-container-low rounded-full border border-outline-variant/30 w-fit">
              <span className="material-symbols-outlined text-tertiary text-sm">local_taxi</span>
              <span className="text-xs font-label text-on-surface-variant tracking-wide uppercase">Campus Transit</span>
            </div>
            <h1 className="font-headline text-5xl md:text-7xl font-semibold text-on-background leading-tight">
              Ride together,<br />
              <span className="text-primary">grounded</span> in community.
            </h1>
            <p className="text-lg md:text-xl text-on-surface-variant leading-relaxed max-w-md font-body">
              Students request rides. Marshal spatially clusters them, scores the groups, and dispatches a driver via Telegram.
            </p>
          </div>

          {/* Start a Ride Card (Bento Style) */}
          <div className="bg-surface-bright rounded-3xl p-6 shadow-[0_4px_20px_rgba(46,50,48,0.06)] border border-outline-variant/20 max-w-md w-full relative overflow-hidden group hover:shadow-md transition-shadow duration-300">
            <div className="flex items-center justify-between mb-6">
              <h3 className="font-headline text-xl font-medium text-on-surface">Start a Ride</h3>
              <div className="bg-primary-container text-on-primary-container p-2 rounded-full flex items-center justify-center">
                <span className="material-symbols-outlined">directions_car</span>
              </div>
            </div>
            <form onSubmit={handleFindRides} className="space-y-4 relative">
              {/* Journey Line connecting dots */}
              <div className="absolute left-[11px] top-[24px] bottom-[24px] w-[2px] bg-outline-variant/30 z-0"></div>

              {/* Pickup */}
              <div className="flex items-center gap-4 relative z-10 bg-surface-container-lowest p-3 rounded-xl border border-outline-variant/20 focus-within:border-primary focus-within:ring-1 focus-within:ring-primary transition-all">
                <div className="w-6 h-6 rounded-full bg-surface border-2 border-primary flex items-center justify-center shrink-0">
                  <div className="w-2 h-2 rounded-full bg-primary"></div>
                </div>
                <div className="flex-grow">
                  <label className="text-xs text-on-surface-variant font-label">Pickup</label>
                  <input
                    className="w-full bg-transparent border-none p-0 text-on-surface focus:ring-0 text-sm font-medium outline-none"
                    placeholder="Where are you?"
                    type="text"
                    value={pickup}
                    onChange={e => setPickup(e.target.value)}
                  />
                </div>
                <button type="button" className="text-on-surface-variant hover:text-primary transition-colors">
                  <span className="material-symbols-outlined">my_location</span>
                </button>
              </div>

              {/* Dropoff */}
              <div className="flex items-center gap-4 relative z-10 bg-surface-container-lowest p-3 rounded-xl border border-outline-variant/20 focus-within:border-primary focus-within:ring-1 focus-within:ring-primary transition-all">
                <div className="w-6 h-6 rounded-full bg-surface border-2 border-tertiary flex items-center justify-center shrink-0">
                  <span className="material-symbols-outlined text-[14px] text-tertiary filled">location_on</span>
                </div>
                <div className="flex-grow">
                  <label className="text-xs text-on-surface-variant font-label">Dropoff</label>
                  <input
                    className="w-full bg-transparent border-none p-0 text-on-surface focus:ring-0 text-sm font-medium outline-none"
                    placeholder="Where to?"
                    type="text"
                    value={dropoff}
                    onChange={e => setDropoff(e.target.value)}
                  />
                </div>
              </div>

              <button
                type="submit"
                className="w-full mt-6 bg-primary text-on-primary rounded-xl py-4 font-medium text-lg hover:bg-primary/90 active:scale-[0.98] transition-all flex items-center justify-center gap-2 shadow-sm"
              >
                <span>Find Rides</span>
                <span className="material-symbols-outlined">arrow_forward</span>
              </button>
            </form>
          </div>

          <div className="flex items-center gap-4 text-sm text-on-surface-variant mt-2">
            <div className="flex -space-x-2">
              <div className="w-8 h-8 rounded-full border-2 border-background bg-primary-container text-on-primary-container flex items-center justify-center font-bold text-xs">AK</div>
              <div className="w-8 h-8 rounded-full border-2 border-background bg-tertiary-container text-on-tertiary-container flex items-center justify-center font-bold text-xs">RS</div>
              <div className="w-8 h-8 rounded-full border-2 border-background bg-secondary-container text-on-secondary-container flex items-center justify-center font-bold text-xs">MK</div>
            </div>
            <span>Join students riding today</span>
          </div>
        </div>

        {/* Right Column: Visuals */}
        <div className="flex-1 relative min-h-[400px] md:min-h-full flex items-center justify-center z-10">
          {/* Map Visualization Glassmorphism Container */}
          <div className="relative w-full aspect-square md:aspect-auto md:h-[600px] rounded-[2rem] overflow-hidden shadow-[0_8px_30px_rgba(46,50,48,0.08)] border border-white/20">
            {/* Campus Map Image */}
            <div
              className="absolute inset-0 bg-cover bg-center"
              style={{ backgroundImage: `url(${campusMapImg})` }}
            ></div>
            {/* Map Overlay Gradient */}
            <div className="absolute inset-0 bg-gradient-to-t from-background via-transparent to-transparent opacity-60"></div>

            {/* Floating Elements on Map */}
            <div className="absolute top-1/4 left-1/4 bg-surface-bright/90 backdrop-blur-md p-3 rounded-2xl shadow-sm border border-outline-variant/30 flex items-center gap-3 animate-[bounce_4s_infinite]">
              <div className="w-10 h-10 rounded-full bg-primary-container text-on-primary-container flex items-center justify-center">
                <span className="material-symbols-outlined filled">directions_car</span>
              </div>
              <div>
                <p className="text-xs text-on-surface-variant font-label">Heading to</p>
                <p className="text-sm font-medium text-on-surface">Science Center</p>
              </div>
            </div>

            <div className="absolute bottom-1/3 right-1/4 bg-surface-bright/90 backdrop-blur-md px-4 py-2 rounded-full shadow-sm border border-outline-variant/30 flex items-center gap-2">
              <span className="w-2 h-2 rounded-full bg-tertiary animate-pulse"></span>
              <span className="text-sm font-medium text-on-surface">3 drivers nearby</span>
            </div>

            {/* Center Pin */}
            <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 flex flex-col items-center">
              <div className="bg-primary text-on-primary p-3 rounded-full shadow-lg relative z-10">
                <span className="material-symbols-outlined filled text-2xl">person_pin_circle</span>
              </div>
              {/* Pulse effect */}
              <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-16 h-16 bg-primary/20 rounded-full animate-ping -z-10"></div>
            </div>
          </div>
        </div>
      </main>

      {/* Features Section */}
      <section className="max-w-7xl mx-auto px-6 pt-6 md:pt-10 pb-16 md:pb-24">
        <div className="text-center mb-16">
          <h2 className="font-headline text-3xl md:text-4xl font-semibold text-on-background mb-4">Why ride with Marshal?</h2>
          <p className="text-on-surface-variant text-lg max-w-2xl mx-auto font-body">Purpose-built for the campus community, focusing on safety and efficiency.</p>
        </div>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-8">
          {FEATURES.map(({ icon, iconBg, title, body }) => (
            <div key={title} className="bg-surface-container-low p-8 rounded-3xl border border-outline-variant/30 flex flex-col items-start gap-4 hover:shadow-md transition-shadow">
              <div className={`w-12 h-12 rounded-2xl flex items-center justify-center ${iconBg}`}>
                <span className="material-symbols-outlined filled">{icon}</span>
              </div>
              <h3 className="font-headline text-xl font-medium text-on-surface">{title}</h3>
              <p className="text-on-surface-variant font-body">{body}</p>
            </div>
          ))}
        </div>
      </section>

      {/* What's Different Section */}
      <section className="bg-surface-container-low py-16 md:py-24">
        <div className="max-w-7xl mx-auto px-6">
          <div className="text-center mb-16">
            <h2 className="font-headline text-3xl md:text-4xl font-semibold text-on-background mb-4">A Better Way to Ride</h2>
            <p className="text-on-surface-variant text-lg max-w-2xl mx-auto font-body">Leave the uncertainty behind.</p>
          </div>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-8 lg:gap-16 items-center">
            {/* Traditional */}
            <div className="bg-surface p-8 rounded-3xl border border-outline-variant/20 shadow-sm opacity-80">
              <div className="flex items-center gap-3 mb-6 text-error">
                <span className="material-symbols-outlined filled">cancel</span>
                <h3 className="font-headline text-2xl font-medium">Traditional Lobby Chaos</h3>
              </div>
              <ul className="space-y-4">
                {COMPARISON_OLD.map(item => (
                  <li key={item} className="flex items-start gap-3 text-on-surface-variant">
                    <span className="material-symbols-outlined text-outline mt-0.5 text-xl">remove</span>
                    {item}
                  </li>
                ))}
              </ul>
            </div>
            {/* Marshal */}
            <div className="bg-primary text-on-primary p-8 rounded-3xl shadow-lg transform md:-translate-y-4 relative overflow-hidden">
              <div className="absolute -right-10 -top-10 w-40 h-40 bg-white/10 rounded-full blur-2xl"></div>
              <div className="flex items-center gap-3 mb-6">
                <span className="material-symbols-outlined filled text-primary-fixed">check_circle</span>
                <h3 className="font-headline text-2xl font-medium">The Marshal Way</h3>
              </div>
              <ul className="space-y-4 relative z-10">
                {COMPARISON_NEW.map(item => (
                  <li key={item} className="flex items-start gap-3 text-on-primary">
                    <span className="material-symbols-outlined text-primary-fixed mt-0.5 text-xl">add</span>
                    {item}
                  </li>
                ))}
              </ul>
            </div>
          </div>
        </div>
      </section>

      {/* Footer */}
      <footer className="bg-surface-container-high py-12 md:py-16 pt-16">
        <div className="max-w-7xl mx-auto px-6 grid grid-cols-1 md:grid-cols-4 gap-12">
          <div className="col-span-1 md:col-span-1">
            <div className="flex items-center gap-2 mb-4">
              <span className="material-symbols-outlined text-primary text-3xl filled">local_taxi</span>
              <span className="font-headline text-2xl font-bold text-primary">Marshal</span>
            </div>
            <p className="text-on-surface-variant text-sm mb-6 font-body">Ride together, grounded in community. Safe and efficient transit for the modern campus.</p>
          </div>
          <div>
            <h4 className="font-headline font-semibold text-on-surface mb-4">Portals</h4>
            <ul className="space-y-3 text-sm text-on-surface-variant font-body">
              <li><a className="hover:text-primary transition-colors" href="#">Student App</a></li>
              <li><a className="hover:text-primary transition-colors" href="#">Admin Console</a></li>
            </ul>
          </div>
          <div>
            <h4 className="font-headline font-semibold text-on-surface mb-4">Stack</h4>
            <ul className="space-y-3 text-sm text-on-surface-variant font-body">
              <li><a className="hover:text-primary transition-colors" href="#">Go Backend</a></li>
              <li><a className="hover:text-primary transition-colors" href="#">PostgreSQL / H3</a></li>
              <li><a className="hover:text-primary transition-colors" href="#">Telegram Bot</a></li>
            </ul>
          </div>
          <div>
            <h4 className="font-headline font-semibold text-on-surface mb-4">Support</h4>
            <ul className="space-y-3 text-sm text-on-surface-variant font-body">
              <li><a className="hover:text-primary transition-colors" href="#">Documentation</a></li>
              <li><a className="hover:text-primary transition-colors" href="#">GitHub Repo</a></li>
            </ul>
          </div>
        </div>
        <div className="max-w-7xl mx-auto px-6 mt-12 pt-8 border-t border-outline-variant/30 flex flex-col md:flex-row justify-between items-center gap-4 text-xs text-on-surface-variant font-body">
          <p>© 2025 Marshal. Campus ride-grouping engine.</p>
        </div>
      </footer>
    </div>
  )
}
