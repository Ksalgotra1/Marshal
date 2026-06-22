import { Radio, RefreshCw } from 'lucide-react'

function formatSync(value) {
  if (!value) return 'Waiting'
  return new Date(value).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}

export function DashboardHeader({ activeRequest, isRefreshing, lastSyncAt, onRefresh }) {
  return (
    <header className="topbar">
      <a className="brand" href="/">
        <span className="brand-mark">M</span>
        <span>Marshal</span>
      </a>
      <div className="topbar-status">
        <span className="live-dot" />
        <span>{activeRequest ? activeRequest.status : 'No active request'}</span>
      </div>
      <button className="icon-button" type="button" onClick={onRefresh} aria-label="Refresh dashboard">
        {isRefreshing ? <RefreshCw className="spin" size={17} /> : <Radio size={17} />}
      </button>
      <span className="sync-time">Synced {formatSync(lastSyncAt)}</span>
    </header>
  )
}
