import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useApi } from '../../hooks/useApi'
import { useEventSource } from '../../hooks/useEventSource'
import { TopNav } from '../shared/TopNav'
import { StatsBar } from './StatsBar'
import { QueueColumn } from './QueueColumn'
import { GroupsColumn } from './GroupsColumn'
import { DriversColumn } from './DriversColumn'

async function fetchGroupDetails(request, groupIds) {
  const details = await Promise.all(
    groupIds.map(async id => {
      try { return await request(`/api/groups/${id}`) }
      catch { return null }
    }),
  )
  const out = {}
  details.forEach(d => {
    if (!d?.group) return
    out[d.group.id] = (d.members || []).map(m => m.requester_name)
  })
  return out
}

function avg(groups) {
  if (!groups.length) return 0
  return groups.reduce((s, g) => s + Number(g.route_score || 0), 0) / groups.length
}

function useDebouncedCallback(callback, delay) {
  const callbackRef = useRef(callback)
  const timerRef = useRef(null)

  useEffect(() => {
    callbackRef.current = callback
  }, [callback])

  useEffect(() => () => {
    if (timerRef.current) clearTimeout(timerRef.current)
  }, [])

  return useCallback(() => {
    if (timerRef.current) clearTimeout(timerRef.current)
    timerRef.current = setTimeout(() => {
      callbackRef.current()
    }, delay)
  }, [delay])
}

const PAGE_SIZE = 20

export function AdminDashboard() {
  const { request, error, setError } = useApi()
  const [requests, setRequests] = useState([])
  const [groups, setGroups]     = useState([])
  const [drivers, setDrivers]   = useState([])
  const [connections, setCx]    = useState(0)
  const [groupMembers, setGM]   = useState({})
  const [lastSyncAt, setLastSyncAt] = useState(null)
  const [isRefreshing, setIsRefreshing] = useState(false)
  const [requestLimit, setRequestLimit] = useState(PAGE_SIZE)
  const [groupLimit, setGroupLimit] = useState(PAGE_SIZE)
  const [driverLimit, setDriverLimit] = useState(PAGE_SIZE)

  const refreshAll = useCallback(async () => {
    setIsRefreshing(true)
    try {
      const [q, g, d, h] = await Promise.all([
        request(`/api/requests?status=pending&limit=${requestLimit}`),
        request(`/api/groups?limit=${groupLimit}`),
        request(`/api/drivers?limit=${driverLimit}`),
        request('/healthz'),
      ])
      setRequests(Array.isArray(q) ? q : [])
      setGroups(Array.isArray(g) ? g : [])
      setDrivers(Array.isArray(d) ? d : [])
      setCx(Number(h?.connections || 0))
      const ids = (Array.isArray(g) ? g : []).map(x => x.id)
      setGM(await fetchGroupDetails(request, ids))
      setLastSyncAt(new Date().toISOString())
    } finally {
      setIsRefreshing(false)
    }
  }, [request, requestLimit, groupLimit, driverLimit])

  const scheduleRefresh = useDebouncedCallback(() => {
    refreshAll().catch(() => {})
  }, 400)

  useEffect(() => {
    let dead = false
    const run = async () => { if (!dead) await refreshAll().catch(() => {}) }
    run()
    const t = setInterval(run, 60000)
    return () => { dead = true; clearInterval(t) }
  }, [refreshAll])

  useEventSource('/events?room=global', {
    enabled: true,
    onMessage: ev => {
      if (ev?.type === 'system:connections') {
        setCx(Number(ev?.data?.connections || 0))
        setLastSyncAt(new Date().toISOString())
        return
      }

      const evts = ['request:created','group:formed','group:dispatching','group:assigned','group:cancelled','member:joined','driver:registered']
      if (evts.includes(ev?.type)) {
        scheduleRefresh()
      }
    },
  })

  const groupDriverNames = useMemo(() => {
    const byId = Object.fromEntries(drivers.map(d => [d.id, d.name]))
    return Object.fromEntries(groups.filter(g => g.driver_id).map(g => [g.id, byId[g.driver_id] || 'Unknown']))
  }, [drivers, groups])

  const stats = useMemo(() => ({
    groups: groups.length,
    onlineDrivers: drivers.filter(d => d.status === 'online').length,
    avgScore: avg(groups),
    connections,
  }), [groups, drivers, connections])

  return (
    <div style={{ minHeight: '100vh', background: 'var(--void)' }}>
      <TopNav />

      <main style={{ maxWidth: '1400px', margin: '0 auto', padding: '24px', display: 'flex', flexDirection: 'column', gap: '16px' }}>
        <StatsBar stats={stats} />

        {error && (
          <div style={{
            border: '1px solid rgba(255,32,71,0.3)',
            borderRadius: '16px',
            padding: '14px 18px',
            background: 'var(--red-dim)',
            display: 'flex',
            alignItems: 'flex-start',
            justifyContent: 'space-between',
            gap: '12px',
          }}>
            <div>
              <p style={{ fontSize: '12px', fontWeight: 500, color: 'var(--red)', marginBottom: '4px', textTransform: 'uppercase', letterSpacing: '0.05em' }}>
                Request error
              </p>
              <p style={{ fontSize: '13px', color: 'var(--silver)' }}>{error}</p>
            </div>
            <button
              type="button"
              onClick={() => setError(null)}
              style={{
                border: '1px solid rgba(255,32,71,0.3)',
                borderRadius: '9999px',
                background: 'transparent',
                color: 'var(--red)',
                fontSize: '11px',
                fontWeight: 500,
                padding: '4px 12px',
                cursor: 'pointer',
                flexShrink: 0,
                letterSpacing: '0.02em',
              }}
            >
              Dismiss
            </button>
          </div>
        )}

        <div className="admin-dashboard-grid">
          <QueueColumn
            requests={requests}
            hasMore={requests.length >= requestLimit}
            onLoadMore={() => setRequestLimit(limit => limit + PAGE_SIZE)}
          />
          <GroupsColumn
            groups={groups}
            groupMembers={groupMembers}
            groupDriverNames={groupDriverNames}
            hasMore={groups.length >= groupLimit}
            onLoadMore={() => setGroupLimit(limit => limit + PAGE_SIZE)}
          />
          <DriversColumn
            drivers={drivers}
            hasMore={drivers.length >= driverLimit}
            onLoadMore={() => setDriverLimit(limit => limit + PAGE_SIZE)}
          />
        </div>

        {isRefreshing && (
          <p style={{ fontSize: '11px', color: 'var(--dark-gray)', letterSpacing: '0.04em' }}>
            Syncing…
          </p>
        )}

        {lastSyncAt && (
          <p style={{ fontSize: '11px', color: 'var(--dark-gray)', letterSpacing: '0.04em' }}>
            Live updated {new Date(lastSyncAt).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' })}
          </p>
        )}
      </main>
    </div>
  )
}
