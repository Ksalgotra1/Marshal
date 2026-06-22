import { useMemo, useState } from 'react'
import { Check, Loader2, Search } from 'lucide-react'

const CACHE_KEY = 'marshal.locationSearchCache'
let lastSearchAt = 0

function readCache() {
  try {
    return JSON.parse(localStorage.getItem(CACHE_KEY) || '{}')
  } catch {
    return {}
  }
}

function writeCache(cache) {
  localStorage.setItem(CACHE_KEY, JSON.stringify(cache))
}

function normalizeResult(item) {
  return {
    id: item.place_id,
    name: item.display_name,
    lat: Number(item.lat),
    lng: Number(item.lon),
  }
}

export function LocationSearchField({ disabled, label, onSelect, value }) {
  const [query, setQuery] = useState(value?.name || '')
  const [results, setResults] = useState([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

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
      const key = term.toLowerCase()
      const cached = cache[key]

      if (cached) {
        setResults(cached)
        return
      }

      const params = new URLSearchParams({
        q: term,
        format: 'jsonv2',
        limit: '5',
        countrycodes: 'in',
      })
      const response = await fetch(`https://nominatim.openstreetmap.org/search?${params}`)
      if (!response.ok) throw new Error('Location search failed')

      const data = await response.json()
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
    <div className="location-search">
      <label>
        <span>{label}</span>
        <div className="search-control">
          <input
            value={query}
            onChange={event => setQuery(event.target.value)}
            onKeyDown={event => {
              if (event.key === 'Enter') {
                event.preventDefault()
                search()
              }
            }}
            placeholder="Search city, sector, campus gate..."
            disabled={disabled}
          />
          <button type="button" aria-label={`Search ${label}`} onClick={search} disabled={disabled || loading}>
            {loading ? <Loader2 className="spin" size={16} /> : <Search size={16} />}
          </button>
        </div>
      </label>

      {value && <p className="coord-preview">{selectedLabel}</p>}
      {error && <p className="field-error">{error}</p>}

      {results.length > 0 && (
        <div className="search-results">
          {results.map(result => (
            <button
              type="button"
              key={result.id}
              onClick={() => {
                onSelect(result)
                setQuery(result.name)
                setResults([])
              }}
              disabled={disabled}
            >
              <span>{result.name}</span>
              {value?.id === result.id && <Check size={15} />}
            </button>
          ))}
        </div>
      )}
    </div>
  )
}
