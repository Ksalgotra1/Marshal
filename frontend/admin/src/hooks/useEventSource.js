import { useEffect, useRef } from 'react'

export function useEventSource(path, { enabled = true, onMessage } = {}) {
  const onMessageRef = useRef(onMessage)

  useEffect(() => {
    onMessageRef.current = onMessage
  }, [onMessage])

  useEffect(() => {
    if (!enabled || !path) return undefined

    const baseUrl = import.meta.env.VITE_API_URL || ''
    const url = path.startsWith('http') ? path : baseUrl + path
    const source = new EventSource(url)

    source.onmessage = (event) => {
      try {
        const payload = JSON.parse(event.data)
        onMessageRef.current?.(payload)
      } catch {
        // Ignore malformed payloads.
      }
    }

    return () => {
      source.close()
    }
  }, [enabled, path])
}
