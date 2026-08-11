import { Routes, Route, Navigate, useLocation } from 'react-router-dom'
import { TopNavBar }      from './components/TopNavBar.jsx'
import { BottomNavBar }   from './components/BottomNavBar.jsx'
import { LandingPage }    from './components/LandingPage.jsx'
import { ExplorePage }    from './components/ExplorePage.jsx'
import { StatusPage }     from './components/StatusPage.jsx'
import { GroupsPage }     from './components/GroupsPage.jsx'
import { GroupChatPanel } from './components/GroupChatPanel.jsx'
import { useStudentDashboard } from './hooks/useStudentDashboard.js'

function ErrorBanner({ error, onDismiss }) {
  if (!error) return null
  return (
    <div className="max-w-7xl mx-auto px-4 md:px-6 mt-4">
      <div className="bg-error-container text-on-error-container rounded-2xl px-5 py-4 flex items-start justify-between gap-4 border border-error/20">
        <div className="flex items-start gap-3">
          <span className="material-symbols-outlined fill-icon text-error mt-0.5 icon-sm">error</span>
          <p className="text-sm font-body">{error}</p>
        </div>
        <button
          type="button"
          onClick={onDismiss}
          className="text-on-error-container hover:text-error transition-colors flex-shrink-0 border border-error/30 rounded-full px-3 py-1 text-xs font-label"
        >
          Dismiss
        </button>
      </div>
    </div>
  )
}

export function App() {
  const location = useLocation()
  const isExplorePage = location.pathname === '/explore'

  const {
    activeRequest,
    events,
    error,
    groupDetail,
    groups,
    isBusy,
    isRefreshing,
    submitRequest,
    joinGroup,
    setError,
    messages,
    isChatSending,
    sendChatMessage,
  } = useStudentDashboard()

  return (
    <div className={`bg-background text-on-background font-body flex flex-col ${isExplorePage ? 'h-screen overflow-hidden' : 'min-h-screen'}`}>
      {/* Desktop top nav */}
      <TopNavBar />

      {/* Global error banner */}
      <ErrorBanner error={error} onDismiss={() => setError(null)} />

      {/* Sync indicator */}
      {isRefreshing && (
        <div className="flex items-center justify-center gap-2 py-2 text-xs text-on-surface-variant font-label">
          <span className="material-symbols-outlined icon-sm animate-spin">progress_activity</span>
          Syncing…
        </div>
      )}

      {/* Page content */}
      <div className="flex-1 flex flex-col min-h-0 h-full w-full overflow-hidden">
        <Routes>
          <Route path="/" element={<LandingPage />} />

          <Route
            path="/explore"
            element={
              <ExplorePage
                activeRequest={activeRequest}
                isBusy={isBusy}
                onSubmit={submitRequest}
                groups={groups}
              />
            }
          />

          <Route
            path="/status"
            element={
              <StatusPage
                activeRequest={activeRequest}
                groupDetail={groupDetail}
                events={events}
                messages={messages}
                isChatSending={isChatSending}
                sendChatMessage={sendChatMessage}
              />
            }
          />

          <Route
            path="/groups"
            element={
              <GroupsPage
                activeRequest={activeRequest}
                groups={groups}
                isBusy={isBusy}
                onJoin={joinGroup}
              />
            }
          />

          {/* Chat: mobile-only standalone tab (desktop sees it inline on /status) */}
          <Route
            path="/chat"
            element={
              <div className="flex-1 flex flex-col p-4 pb-32 h-[calc(100vh-64px)]">
                <GroupChatPanel
                  groupDetail={groupDetail}
                  messages={messages}
                  isChatSending={isChatSending}
                  sendChatMessage={sendChatMessage}
                  currentUserName={activeRequest?.requester_name}
                />
              </div>
            }
          />

          {/* Fallback */}
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </div>

      {/* Mobile bottom nav */}
      <BottomNavBar />
    </div>
  )
}
