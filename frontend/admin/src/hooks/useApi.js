import { useCallback, useState } from 'react'

export function useApi() {
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState(null)

  const request = useCallback(async (path, options = {}) => {
    setLoading(true)
    setError(null)
    try {
      const baseUrl = import.meta.env.VITE_API_URL || ''
      const url = path.startsWith('http') ? path : baseUrl + path
      const response = await fetch(url, {
        headers: {
          'Content-Type': 'application/json',
          ...(options.headers || {}),
        },
        ...options,
      })

      if (!response.ok) {
        let message = `Request failed with status ${response.status}`
        try {
          const body = await response.json()
          message = body.error || message
        } catch {
          // Leave the fallback message when response body isn't JSON.
        }
        throw new Error(message)
      }

      if (response.status === 204) {
        return null
      }

      return await response.json()
    } catch (err) {
      setError(err.message || 'Request failed')
      throw err
    } finally {
      setLoading(false)
    }
  }, [])

  return { request, loading, error, setError }
}
