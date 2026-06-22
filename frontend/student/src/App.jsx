import { useState } from 'react'
import { AlertCircle, RefreshCw } from 'lucide-react'
import { BottomNav } from './components/BottomNav.jsx'
import { DashboardHeader } from './components/DashboardHeader.jsx'
import { DriverChatPanel } from './components/DriverChatPanel.jsx'
import { GroupBrowser } from './components/GroupBrowser.jsx'
import { RequestComposer } from './components/RequestComposer.jsx'
import { RideSummary } from './components/RideSummary.jsx'
import { StatusPanel } from './components/StatusPanel.jsx'
import { useStudentDashboard } from './hooks/useStudentDashboard.js'

export function App() {
  const [activePage, setActivePage] = useState('request')
  const dashboard = useStudentDashboard()
  const {
    activeRequest,
    events,
    error,
    groupDetail,
    groups,
    isBusy,
    isRefreshing,
    lastSyncAt,
    submitRequest,
    joinGroup,
    refresh,
    resetRequest,
    setError,
  } = dashboard

  return (
    <div className="app-shell" id="top">
      <DashboardHeader
        activeRequest={activeRequest}
        isRefreshing={isRefreshing}
        lastSyncAt={lastSyncAt}
        onRefresh={refresh}
      />

      <main className="dashboard-grid">
        <section className="hero-panel liquid-panel desktop-hero">
          <div>
            <p className="eyebrow">Student dispatch</p>
            <h1>Ride together without the lobby chaos.</h1>
          </div>
          <p>
            Request a ride, join a compatible group, and keep the trip state visible while Marshal matches the route.
          </p>
          <div className="hero-actions">
            <button className="pill solid" type="button" onClick={refresh} disabled={isRefreshing}>
              <RefreshCw size={15} />
              Sync
            </button>
            {activeRequest && (
              <button className="pill ghost" type="button" onClick={resetRequest}>
                New request
              </button>
            )}
          </div>
        </section>

        {error && (
          <div className="error-banner liquid-panel">
            <AlertCircle size={17} />
            <span>{error}</span>
            <button type="button" onClick={() => setError(null)}>Dismiss</button>
          </div>
        )}

        <div className={activePage === 'request' ? 'primary-stack mobile-page active' : 'primary-stack mobile-page'}>
          <RequestComposer
            activeRequest={activeRequest}
            isBusy={isBusy}
            onSubmit={submitRequest}
          />
        </div>

        <div className={activePage === 'groups' ? 'primary-stack mobile-page active' : 'primary-stack mobile-page'}>
          <GroupBrowser
            activeRequest={activeRequest}
            groups={groups}
            isBusy={isBusy}
            onJoin={joinGroup}
          />
        </div>

        <aside className={activePage === 'status' ? 'side-stack mobile-page active' : 'side-stack mobile-page'}>
          <StatusPanel
            activeRequest={activeRequest}
            events={events}
            groupDetail={groupDetail}
          />
          <RideSummary
            activeRequest={activeRequest}
            groupDetail={groupDetail}
          />
        </aside>

        <div className={activePage === 'chat' ? 'mobile-page active chat-page' : 'mobile-page chat-page'}>
          <DriverChatPanel activeRequest={activeRequest} groupDetail={groupDetail} />
        </div>
      </main>
      <BottomNav activePage={activePage} onChange={setActivePage} />
    </div>
  )
}
