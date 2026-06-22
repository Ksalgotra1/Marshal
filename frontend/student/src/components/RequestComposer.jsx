import { useMemo, useState } from 'react'
import { LocateFixed, MapPin, Send, Timer } from 'lucide-react'
import { LocationSearchField } from './LocationSearchField.jsx'

const LOCATIONS = [
  { name: 'North Gate', lat: 28.5457, lng: 77.1928 },
  { name: 'Library Steps', lat: 28.5443, lng: 77.1912 },
  { name: 'Hostel Block C', lat: 28.5481, lng: 77.1899 },
  { name: 'Metro Stop', lat: 28.5431, lng: 77.1961 },
]

function getArrivalDefault() {
  const date = new Date(Date.now() + 45 * 60 * 1000)
  date.setSeconds(0, 0)
  return date.toISOString().slice(0, 16)
}

function isValidCoord(lat, lng) {
  return lat >= -90 && lat <= 90 && lng >= -180 && lng <= 180
}

export function RequestComposer({ activeRequest, isBusy, onSubmit }) {
  const [name, setName] = useState(activeRequest?.requester_name || '')
  const [mode, setMode] = useState('search')
  const [pickupSearch, setPickupSearch] = useState(null)
  const [dropoffSearch, setDropoffSearch] = useState(null)
  const [pickup, setPickup] = useState(LOCATIONS[0].name)
  const [dropoff, setDropoff] = useState(LOCATIONS[3].name)
  const [pickupCoords, setPickupCoords] = useState({ lat: '28.5457', lng: '77.1928' })
  const [dropoffCoords, setDropoffCoords] = useState({ lat: '28.5431', lng: '77.1961' })
  const [arriveBy, setArriveBy] = useState(getArrivalDefault)

  const canSubmit = useMemo(() => {
    const pickupLat = Number(pickupCoords.lat)
    const pickupLng = Number(pickupCoords.lng)
    const dropoffLat = Number(dropoffCoords.lat)
    const dropoffLng = Number(dropoffCoords.lng)
    const hasValidCoords = isValidCoord(pickupLat, pickupLng)
      && isValidCoord(dropoffLat, dropoffLng)
      && (pickupLat !== dropoffLat || pickupLng !== dropoffLng)
    const hasValidSavedRoute = pickup !== dropoff
    const hasValidSearchRoute = pickupSearch && dropoffSearch
      && (pickupSearch.lat !== dropoffSearch.lat || pickupSearch.lng !== dropoffSearch.lng)
    return name.trim()
      && !activeRequest
      && (mode === 'search' ? hasValidSearchRoute : mode === 'saved' ? hasValidSavedRoute : hasValidCoords)
  }, [activeRequest, dropoff, dropoffCoords, dropoffSearch, mode, name, pickup, pickupCoords, pickupSearch])

  function handleSubmit(event) {
    event.preventDefault()
    const from = mode === 'search'
      ? pickupSearch
      : mode === 'saved'
      ? LOCATIONS.find(item => item.name === pickup)
      : { lat: Number(pickupCoords.lat), lng: Number(pickupCoords.lng) }
    const to = mode === 'search'
      ? dropoffSearch
      : mode === 'saved'
      ? LOCATIONS.find(item => item.name === dropoff)
      : { lat: Number(dropoffCoords.lat), lng: Number(dropoffCoords.lng) }

    onSubmit({
      requester_name: name.trim(),
      pickup_lat: from.lat,
      pickup_lng: from.lng,
      dropoff_lat: to.lat,
      dropoff_lng: to.lng,
      arrive_by: new Date(arriveBy).toISOString(),
    })
  }

  return (
    <section className="liquid-panel request-panel" id="request">
      <div className="panel-heading">
        <span className="accent-badge orange">Request</span>
        <h2>Start a ride</h2>
      </div>

      <form className="request-form" onSubmit={handleSubmit}>
        <label>
          <span>Name</span>
          <input
            value={name}
            onChange={event => setName(event.target.value)}
            placeholder="Student name"
            disabled={Boolean(activeRequest)}
          />
        </label>

        <div className="segmented-control" aria-label="Location input mode">
          <button className={mode === 'search' ? 'active' : ''} type="button" onClick={() => setMode('search')} disabled={Boolean(activeRequest)}>
            <MapPin size={15} />
            Search
          </button>
          <button className={mode === 'saved' ? 'active' : ''} type="button" onClick={() => setMode('saved')} disabled={Boolean(activeRequest)}>
            <MapPin size={15} />
            Saved
          </button>
          <button className={mode === 'coords' ? 'active' : ''} type="button" onClick={() => setMode('coords')} disabled={Boolean(activeRequest)}>
            <LocateFixed size={15} />
            Coordinates
          </button>
        </div>

        {mode === 'search' ? (
          <div className="coord-stack">
            <LocationSearchField
              disabled={Boolean(activeRequest)}
              label="Pickup"
              value={pickupSearch}
              onSelect={setPickupSearch}
            />
            <LocationSearchField
              disabled={Boolean(activeRequest)}
              label="Dropoff"
              value={dropoffSearch}
              onSelect={setDropoffSearch}
            />
            <p className="attribution">Search by OpenStreetMap Nominatim</p>
          </div>
        ) : mode === 'saved' ? (
          <div className="form-grid">
            <label>
              <span><MapPin size={14} /> Pickup</span>
              <select value={pickup} onChange={event => setPickup(event.target.value)} disabled={Boolean(activeRequest)}>
                {LOCATIONS.map(location => (
                  <option key={location.name} value={location.name}>{location.name}</option>
                ))}
              </select>
            </label>
            <label>
              <span><MapPin size={14} /> Dropoff</span>
              <select value={dropoff} onChange={event => setDropoff(event.target.value)} disabled={Boolean(activeRequest)}>
                {LOCATIONS.map(location => (
                  <option key={location.name} value={location.name}>{location.name}</option>
                ))}
              </select>
            </label>
          </div>
        ) : (
          <div className="coord-stack">
            <div className="form-grid">
              <label>
                <span><LocateFixed size={14} /> Pickup lat</span>
                <input type="number" step="any" value={pickupCoords.lat} onChange={event => setPickupCoords(value => ({ ...value, lat: event.target.value }))} disabled={Boolean(activeRequest)} />
              </label>
              <label>
                <span>Pickup lng</span>
                <input type="number" step="any" value={pickupCoords.lng} onChange={event => setPickupCoords(value => ({ ...value, lng: event.target.value }))} disabled={Boolean(activeRequest)} />
              </label>
            </div>
            <div className="form-grid">
              <label>
                <span><LocateFixed size={14} /> Dropoff lat</span>
                <input type="number" step="any" value={dropoffCoords.lat} onChange={event => setDropoffCoords(value => ({ ...value, lat: event.target.value }))} disabled={Boolean(activeRequest)} />
              </label>
              <label>
                <span>Dropoff lng</span>
                <input type="number" step="any" value={dropoffCoords.lng} onChange={event => setDropoffCoords(value => ({ ...value, lng: event.target.value }))} disabled={Boolean(activeRequest)} />
              </label>
            </div>
          </div>
        )}

        <label>
          <span><Timer size={14} /> Arrive by</span>
          <input
            type="datetime-local"
            value={arriveBy}
            onChange={event => setArriveBy(event.target.value)}
            disabled={Boolean(activeRequest)}
          />
        </label>

        <button className="pill solid wide" type="submit" disabled={!canSubmit || isBusy}>
          <Send size={15} />
          {activeRequest ? 'Request active' : 'Create request'}
        </button>
      </form>
    </section>
  )
}
