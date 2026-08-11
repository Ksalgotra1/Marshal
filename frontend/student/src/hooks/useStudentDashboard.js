import { useCallback, useEffect, useMemo, useRef, useState } from 'react'

// Stable ref wrapper so callbacks can be used inside effects without causing re-subscriptions
function useStableCallback(fn) {
  const ref = useRef(fn)
  ref.current = fn
  return useCallback((...args) => ref.current(...args), [])
}

const STORAGE_KEY = 'marshal.student.requestId'
const EVENT_TYPES = new Set([
  'request:created',
  'group:formed',
  'group:dispatching',
  'group:assigned',
  'group:cancelled',
  'group:completed',
  'member:joined',
])

async function requestJSON(path, options = {}) {
  const baseUrl = import.meta.env.VITE_API_URL || ''
  const url = path.startsWith('http') ? path : baseUrl + path
  let response
  try {
    response = await fetch(url, {
      headers: {
        'Content-Type': 'application/json',
        ...(options.headers || {}),
      },
      ...options,
    })
  } catch {
    throw new Error('Backend unavailable. Start the Marshal API, then try again.')
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
  const [messages, setMessages] = useState([])
  const [isChatSending, setIsChatSending] = useState(false)
  
  const groupId = groupDetail?.group?.id

  const prevGroupIdRef = useRef(groupId)
  useEffect(() => {
    if (groupId && prevGroupIdRef.current && groupId !== prevGroupIdRef.current) {
      setMessages([])
    }
    prevGroupIdRef.current = groupId
  }, [groupId])

  const mergeMessages = useCallback((newMessages) => {
    setMessages(current => {
      const merged = [...current]
      let changed = false
      for (const msg of newMessages) {
        if (!merged.some(m => m.id === msg.id)) {
          merged.push(msg)
          changed = true
        }
      }
      if (!changed) return current
      return merged.sort((a, b) => new Date(a.created_at) - new Date(b.created_at))
    })
  }, [])

  const refreshImpl = useCallback(async () => {
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

  // Stable reference so effects don't re-subscribe on every render
  const refresh = useStableCallback(refreshImpl)

  useEffect(() => {
    refreshImpl()
    const timer = setInterval(refreshImpl, 30000)
    return () => clearInterval(timer)
  }, [refreshImpl])

  // SSE subscription — only reconnects when groupId changes, NOT on refresh changes
  useEffect(() => {
    const room = groupId ? `global,${groupId}` : 'global'
    const baseUrl = import.meta.env.VITE_API_URL || ''
    const path = `/events?room=${room}`
    const url = path.startsWith('http') ? path : baseUrl + path
    const source = new EventSource(url)

    source.onmessage = event => {
      try {
        const payload = JSON.parse(event.data)
        
        if (payload?.type === 'chat:message') {
          if (payload.group_id === groupId && payload.data?.id) {
            mergeMessages([payload.data])
          }
          return
        }

        if (!EVENT_TYPES.has(payload?.type)) return

        const line = toEventLine(payload)
        if (line) setEvents(current => [line, ...current].slice(0, 5))
        refresh()
      } catch {
        // Ignore malformed live events.
      }
    }

    let cancelled = false
    if (groupId) {
      requestJSON(`/api/groups/${groupId}/messages`)
        .then(msgs => {
          if (!cancelled && Array.isArray(msgs)) {
            mergeMessages(msgs)
          }
        })
        .catch(() => null)
    }

    return () => {
      cancelled = true
      source.close()
    }
  }, [groupId, mergeMessages, refresh])

  const sendChatMessage = useCallback(async (content) => {
    if (!groupId) return false
    setIsChatSending(true)
    setError(null)
    try {
      const res = await requestJSON(`/api/groups/${groupId}/messages`, {
        method: 'POST',
        body: JSON.stringify({ content, request_id: activeRequest?.id }),
      })
      if (res?.message) {
        mergeMessages([res.message])
      }
      return true
    } catch (err) {
      setError(err.message || 'Unable to send message')
      return false
    } finally {
      setIsChatSending(false)
    }
  }, [groupId, activeRequest, mergeMessages])

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
    messages,
    isChatSending,
    sendChatMessage,
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
    messages,
    isChatSending,
    sendChatMessage,
  ])
}
