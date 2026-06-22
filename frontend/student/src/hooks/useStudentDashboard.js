import { useCallback, useEffect, useMemo, useState } from 'react'

const STORAGE_KEY = 'marshal.student.requestId'
const EVENT_TYPES = new Set([
  'request:created',
  'group:formed',
  'group:dispatching',
  'group:assigned',
  'group:cancelled',
  'member:joined',
])

async function requestJSON(path, options = {}) {
  let response
  try {
    response = await fetch(path, {
      headers: {
        'Content-Type': 'application/json',
        ...(options.headers || {}),
      },
      ...options,
    })
  } catch {
    throw new Error('Backend unavailable. Start the Marshal API on port 8080, then try again.')
  }

  if (!response.ok) {
    let message = `Request failed with status ${response.status}`
    try {
      const body = await response.json()
      message = body.error || message
    } catch {
      // Keep the HTTP fallback when the server does not return JSON.
    }
    throw new Error(message)
  }

  if (response.status === 204) return null
  return response.json()
}

function toEventLine(event) {
  if (!event?.type) return null
  const id = `${event.type}-${Date.now()}-${Math.random()}`
  return {
    id,
    type: event.type,
    label: event.type.replaceAll(':', ' '),
    at: new Date().toISOString(),
  }
}

export function useStudentDashboard() {
  const [requestId, setRequestId] = useState(() => localStorage.getItem(STORAGE_KEY) || '')
  const [activeRequest, setActiveRequest] = useState(null)
  const [groups, setGroups] = useState([])
  const [groupDetail, setGroupDetail] = useState(null)
  const [events, setEvents] = useState([])
  const [error, setError] = useState(null)
  const [isBusy, setIsBusy] = useState(false)
  const [isRefreshing, setIsRefreshing] = useState(false)
  const [lastSyncAt, setLastSyncAt] = useState(null)

  const refresh = useCallback(async () => {
    setIsRefreshing(true)
    try {
      const [openGroups, request] = await Promise.all([
        requestJSON('/api/groups/open').catch(() => []),
        requestId ? requestJSON(`/api/requests/${requestId}`).catch(() => null) : Promise.resolve(null),
      ])

      setGroups(Array.isArray(openGroups) ? openGroups : [])
      setActiveRequest(request)

      if (request?.group_id) {
        setGroupDetail(await requestJSON(`/api/groups/${request.group_id}`).catch(() => null))
      } else {
        setGroupDetail(null)
      }

      setLastSyncAt(new Date().toISOString())
    } catch (err) {
      setError(err.message || 'Unable to refresh dashboard')
    } finally {
      setIsRefreshing(false)
    }
  }, [requestId])

  useEffect(() => {
    const firstRun = setTimeout(refresh, 0)
    const timer = setInterval(refresh, 30000)
    return () => {
      clearTimeout(firstRun)
      clearInterval(timer)
    }
  }, [refresh])

  useEffect(() => {
    const source = new EventSource('/events?room=global')

    source.onmessage = event => {
      try {
        const payload = JSON.parse(event.data)
        if (!EVENT_TYPES.has(payload?.type)) return

        const line = toEventLine(payload)
        if (line) setEvents(current => [line, ...current].slice(0, 5))
        refresh()
      } catch {
        // Ignore malformed live events.
      }
    }

    return () => source.close()
  }, [refresh])

  const submitRequest = useCallback(async payload => {
    setIsBusy(true)
    setError(null)
    try {
      const created = await requestJSON('/api/requests', {
        method: 'POST',
        body: JSON.stringify(payload),
      })
      localStorage.setItem(STORAGE_KEY, created.id)
      setRequestId(created.id)
      setEvents(current => [{
        id: `request-created-${created.id}`,
        type: 'request:created',
        label: 'request created',
        at: new Date().toISOString(),
      }, ...current].slice(0, 5))
    } catch (err) {
      setError(err.message || 'Unable to create request')
    } finally {
      setIsBusy(false)
    }
  }, [])

  const joinGroup = useCallback(async groupId => {
    if (!activeRequest?.id) return

    setIsBusy(true)
    setError(null)
    try {
      await requestJSON(`/api/groups/${groupId}/join`, {
        method: 'POST',
        body: JSON.stringify({ request_id: activeRequest.id }),
      })
      await refresh()
    } catch (err) {
      setError(err.message || 'Unable to join group')
    } finally {
      setIsBusy(false)
    }
  }, [activeRequest, refresh])

  const resetRequest = useCallback(() => {
    localStorage.removeItem(STORAGE_KEY)
    setRequestId('')
    setActiveRequest(null)
    setGroupDetail(null)
    setEvents([])
  }, [])

  return useMemo(() => ({
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
  }), [
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
  ])
}
