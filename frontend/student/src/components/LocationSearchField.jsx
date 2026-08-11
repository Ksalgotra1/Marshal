import { useEffect, useMemo, useState } from 'react'

const CACHE_KEY = 'marshal.locationSearchCache'
let lastSearchAt = 0

function readCache() {
  try { return JSON.parse(localStorage.getItem(CACHE_KEY) || '{}') }
  catch { return {} }
}

function writeCache(cache) {
  localStorage.setItem(CACHE_KEY, JSON.stringify(cache))
}

function normalizeResult(item) {
  return {
    id:   item.place_id,
    name: item.display_name,
    lat:  Number(item.lat),
    lng:  Number(item.lon),
  }
}

export function LocationSearchField({ disabled, initialQuery = '', label, onSelect, value }) {
  const [query, setQuery]     = useState(value?.name || initialQuery)
  const [results, setResults] = useState([])
  const [loading, setLoading] = useState(false)
  const [error, setError]     = useState('')

  // Sync the text input when value is set externally (e.g. hydrated from activeRequest)
  useEffect(() => {
    if (value?.name && value.name !== query) {
      setQuery(value.name)
    } else if (!value && !initialQuery) {
      setQuery('')
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [value])

  const selectedLabel = useMemo(() => {
    if (!value) return ''
    return `${value.lat.toFixed(5)}, ${value.lng.toFixed(5)}`
  }, [value])

  async function search() {
    const term = query.trim()
    if (term.length < 3 || loading) return

    const now = Date.now()
    if (now - lastSearchAt < 1000) {
      setError('Wait a second before searching again.')
      return
    }

    setLoading(true)
    setError('')
    lastSearchAt = now

    try {
      const cache = readCache()
      const key   = term.toLowerCase()
      if (cache[key]) { setResults(cache[key]); return }

      const params = new URLSearchParams({ q: term, format: 'jsonv2', limit: '5', countrycodes: 'in' })
      const res = await fetch(`https://nominatim.openstreetmap.org/search?${params}`)
      if (!res.ok) throw new Error('Location search failed')

      const data       = await res.json()
      const normalized = data.map(normalizeResult)
      cache[key] = normalized
      writeCache(cache)
      setResults(normalized)
    } catch (err) {
      setError(err.message || 'Location search failed')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="w-full space-y-1">
      {/* Input row */}
      <div className="flex gap-2">
        <input
          className="flex-1 px-4 py-3 rounded-xl border-none shadow-sm focus:ring-2 focus:ring-primary bg-surface-bright text-on-surface placeholder:text-on-surface-variant/50 text-sm"
          placeholder={`Search ${label.toLowerCase()} location…`}
          value={query}
          onChange={e => setQuery(e.target.value)}
          onKeyDown={e => { if (e.key === 'Enter') { e.preventDefault(); search() } }}
          disabled={disabled}
        />
        <button
          type="button"
          aria-label={`Search ${label}`}
          onClick={search}
          disabled={disabled || loading}
          className="w-11 h-11 rounded-xl bg-surface-bright border border-outline-variant/40 flex items-center justify-center text-primary hover:bg-primary hover:text-on-primary transition-all active:scale-95 disabled:opacity-50 shadow-sm"
        >
          {loading
            ? <span className="material-symbols-outlined icon-sm animate-spin">progress_activity</span>
            : <span className="material-symbols-outlined icon-sm">search</span>
          }
        </button>
      </div>

      {/* Selected coord preview */}
      {value && (
        <p className="text-xs text-on-surface-variant/60 font-label ml-1">{selectedLabel}</p>
      )}

      {/* Error */}
      {error && (
        <p className="text-xs text-error font-label ml-1">{error}</p>
      )}

      {/* Results dropdown */}
      {results.length > 0 && (
        <div className="bg-surface-bright rounded-xl shadow-medium border border-outline-variant/20 overflow-hidden mt-1">
          {results.map(result => (
            <button
              type="button"
              key={result.id}
              className="w-full text-left px-4 py-3 text-sm text-on-surface hover:bg-surface-container-low transition-colors flex items-center justify-between gap-2 border-b border-outline-variant/10 last:border-0"
              onClick={() => {
                onSelect(result)
                setQuery(result.name)
                setResults([])
              }}
              disabled={disabled}
            >
              <span className="truncate">{result.name}</span>
              {value?.id === result.id && (
                <span className="material-symbols-outlined icon-sm text-primary flex-shrink-0">check</span>
              )}
            </button>
          ))}
        </div>
      )}
    </div>
  )
}
