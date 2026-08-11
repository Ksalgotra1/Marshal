const REVERSE_CACHE_KEY = 'marshal.reverseGeoCache'

function getCache() {
  try { return JSON.parse(localStorage.getItem(REVERSE_CACHE_KEY) || '{}') }
  catch { return {} }
}

function setCache(cache) {
  try { localStorage.setItem(REVERSE_CACHE_KEY, JSON.stringify(cache)) }
  catch { /* ignore */ }
}

/**
 * Turns lat, lng into a short human-readable location name (e.g. "Hauz Khas", "IIT Delhi Main Gate").
 * Uses local caching to avoid duplicate requests.
 */
export async function getAreaName(lat, lng) {
  if (!lat || !lng) return 'Campus Area'
  
  const key = `${Number(lat).toFixed(3)},${Number(lng).toFixed(3)}`
  const cache = getCache()
  
  if (cache[key]) {
    return cache[key]
  }

  try {
    const res = await fetch(`https://nominatim.openstreetmap.org/reverse?lat=${lat}&lon=${lng}&format=jsonv2&zoom=16`)
    if (!res.ok) throw new Error('Reverse geo failed')
    const data = await res.json()
    
    // Pick the most relevant short place name
    const addr = data.address || {}
    const placeName = addr.amenity || addr.building || addr.suburb || addr.neighbourhood || addr.road || addr.city_district || data.name || 'Campus Zone'
    
    cache[key] = placeName
    setCache(cache)
    return placeName
  } catch {
    return `Zone (${Number(lat).toFixed(2)}°, ${Number(lng).toFixed(2)}°)`
  }
}

/**
 * Returns initials from a student's full name (e.g. "Alex Kim" -> "AK")
 */
export function getInitials(name) {
  if (!name) return 'S'
  const parts = name.trim().split(/\s+/)
  if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase()
  return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase()
}

/**
 * Mask full name to initials only for privacy (e.g. "Alex Kim" -> "A. K.")
 */
export function maskName(name) {
  if (!name) return 'Student'
  const parts = name.trim().split(/\s+/)
  return parts.map(p => `${p[0].toUpperCase()}.`).join(' ')
}
