import { useEffect, useRef, useMemo, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import 'leaflet/dist/leaflet.css'
import { LocationSearchField } from './LocationSearchField.jsx'

const DEFAULT_CENTER = [28.5457, 77.1928]
const DEFAULT_ZOOM   = 15

function getArrivalDefault() {
  const d = new Date(Date.now() + 45 * 60 * 1000)
  d.setSeconds(0, 0)
  return d.toISOString().slice(0, 16)
}

function isValidCoord(lat, lng) {
  return lat >= -90 && lat <= 90 && lng >= -180 && lng <= 180
}

function LeafletMap({ pickupCoord, dropoffCoord, groups }) {
  const containerRef = useRef(null)
  const mapRef       = useRef(null)
  const markersRef   = useRef([])
  const [mapReady, setMapReady] = useState(false)

  useEffect(() => {
    if (!containerRef.current || mapRef.current) return

    import('leaflet').then(({ default: L }) => {
      delete L.Icon.Default.prototype._getIconUrl
      L.Icon.Default.mergeOptions({
        iconRetinaUrl: 'https://unpkg.com/leaflet@1.9.4/dist/images/marker-icon-2x.png',
        iconUrl:       'https://unpkg.com/leaflet@1.9.4/dist/images/marker-icon.png',
        shadowUrl:     'https://unpkg.com/leaflet@1.9.4/dist/images/marker-shadow.png',
      })

      const map = L.map(containerRef.current, {
        center:      DEFAULT_CENTER,
        zoom:        DEFAULT_ZOOM,
        zoomControl: false,
      })

      L.tileLayer('https://{s}.basemaps.cartocdn.com/rastertiles/voyager/{z}/{x}/{y}{r}.png', {
        attribution: '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> &copy; <a href="https://carto.com/attributions">CARTO</a>',
        maxZoom: 19,
        subdomains: 'abcd',
      }).addTo(map)

      L.control.zoom({ position: 'bottomright' }).addTo(map)

      mapRef.current = { map, L }
      setMapReady(true)  // ← trigger marker effect after map is ready
    })

    return () => {
      if (mapRef.current?.map) {
        mapRef.current.map.remove()
        mapRef.current = null
      }
    }
  }, [])

  useEffect(() => {
    if (!mapRef.current) return
    const { map, L } = mapRef.current

    // Clear all previous layers
    markersRef.current.forEach(m => map.removeLayer(m))
    markersRef.current = []

    const hasPickup  = pickupCoord  && isValidCoord(pickupCoord.lat, pickupCoord.lng)
    const hasDropoff = dropoffCoord && isValidCoord(dropoffCoord.lat, dropoffCoord.lng)

    // ── Pickup marker (green pulsing dot with label) ──
    if (hasPickup) {
      const pickupIcon = L.divIcon({
        className: 'marshal-map-marker',
        html: `<div style="display:flex;flex-direction:column;align-items:center;">
          <div style="
            width:22px;height:22px;background:#4a7c59;border-radius:50%;
            border:4px solid #fffdf9;box-shadow:0 2px 12px rgba(74,124,89,.55);
          "></div>
          <div style="
            margin-top:4px;background:#4a7c59;color:#fff;font-size:11px;
            font-weight:700;padding:2px 8px;border-radius:8px;white-space:nowrap;
            box-shadow:0 2px 8px rgba(0,0,0,.15);font-family:'Nunito Sans',sans-serif;
          ">Pickup</div>
        </div>`,
        iconSize: [60, 50],
        iconAnchor: [30, 11],
      })

      const marker = L.marker([pickupCoord.lat, pickupCoord.lng], { icon: pickupIcon })
        .addTo(map)
        .bindPopup(pickupCoord.name || 'Pickup location')

      const pulse = L.circle([pickupCoord.lat, pickupCoord.lng], {
        radius: 60,
        color: '#4a7c59',
        fillColor: '#4a7c59',
        fillOpacity: 0.08,
        weight: 1,
        dashArray: '4 4',
      }).addTo(map)

      markersRef.current.push(marker, pulse)
    }

    // ── Dropoff marker (amber/tertiary pin with label) ──
    if (hasDropoff) {
      const dropoffIcon = L.divIcon({
        className: 'marshal-map-marker',
        html: `<div style="display:flex;flex-direction:column;align-items:center;">
          <div style="
            width:22px;height:22px;background:#c4a66a;border-radius:50%;
            border:4px solid #fffdf9;box-shadow:0 2px 12px rgba(196,166,106,.55);
          "></div>
          <div style="
            margin-top:4px;background:#705c30;color:#fff;font-size:11px;
            font-weight:700;padding:2px 8px;border-radius:8px;white-space:nowrap;
            box-shadow:0 2px 8px rgba(0,0,0,.15);font-family:'Nunito Sans',sans-serif;
          ">Dropoff</div>
        </div>`,
        iconSize: [60, 50],
        iconAnchor: [30, 11],
      })

      const marker = L.marker([dropoffCoord.lat, dropoffCoord.lng], { icon: dropoffIcon })
        .addTo(map)
        .bindPopup(dropoffCoord.name || 'Dropoff location')

      markersRef.current.push(marker)
    }

    // ── Route line connecting pickup → dropoff ──
    if (hasPickup && hasDropoff) {
      // Fit bounds immediately so the user sees both markers right away
      const bounds = L.latLngBounds(
        [pickupCoord.lat, pickupCoord.lng],
        [dropoffCoord.lat, dropoffCoord.lng]
      )
      map.fitBounds(bounds, { padding: [60, 60], maxZoom: 15, animate: true })

      // Fetch real driving route from OSRM (free, no API key)
      const osrmUrl = `https://router.project-osrm.org/route/v1/driving/${pickupCoord.lng},${pickupCoord.lat};${dropoffCoord.lng},${dropoffCoord.lat}?overview=full&geometries=geojson`

      fetch(osrmUrl)
        .then(res => res.json())
        .then(data => {
          if (!mapRef.current) return
          const route = data?.routes?.[0]
          if (route?.geometry?.coordinates) {
            // OSRM returns [lng, lat] — Leaflet needs [lat, lng]
            const latlngs = route.geometry.coordinates.map(([lng, lat]) => [lat, lng])

            const routeLine = L.polyline(latlngs, {
              color: '#4a7c59',
              weight: 4,
              opacity: 0.7,
              lineCap: 'round',
              lineJoin: 'round',
            }).addTo(map)

            // Add a subtle shadow line underneath for depth
            const shadowLine = L.polyline(latlngs, {
              color: '#2e3230',
              weight: 7,
              opacity: 0.1,
              lineCap: 'round',
              lineJoin: 'round',
            }).addTo(map)
            shadowLine.bringToBack()

            markersRef.current.push(routeLine, shadowLine)

            // Re-fit to the actual route bounds
            map.fitBounds(routeLine.getBounds(), { padding: [60, 60], maxZoom: 15, animate: true })
          }
        })
        .catch(() => {
          // Fallback: straight dashed line if OSRM is unreachable
          if (!mapRef.current) return
          const fallback = L.polyline(
            [[pickupCoord.lat, pickupCoord.lng], [dropoffCoord.lat, dropoffCoord.lng]],
            { color: '#4a7c59', weight: 3, opacity: 0.5, dashArray: '8 6', lineCap: 'round' }
          ).addTo(map)
          markersRef.current.push(fallback)
        })
    } else if (hasPickup) {
      map.setView([pickupCoord.lat, pickupCoord.lng], 15, { animate: true })
    } else if (hasDropoff) {
      map.setView([dropoffCoord.lat, dropoffCoord.lng], 15, { animate: true })
    }

    // ── Group markers ──
    groups.forEach(group => {
      const item = group.group || group
      const members = group.members || item.members || []
      const firstMember = members[0]
      if (!firstMember?.pickup_lat) return

      const groupIcon = L.divIcon({
        className: 'marshal-map-marker',
        html: `<div style="
          width:44px;height:44px;background:#78a886;border-radius:50%;
          border:4px solid #fffdf9;display:flex;align-items:center;
          justify-content:center;box-shadow:0 4px 14px rgba(46,50,48,0.18);
          font-family:'Material Symbols Outlined';font-size:18px;color:#fff;
        ">group</div>`,
        iconSize: [44, 44],
        iconAnchor: [22, 22],
      })

      const m = L.marker([firstMember.pickup_lat, firstMember.pickup_lng], { icon: groupIcon })
        .addTo(map)
        .bindPopup(`${members.length} rider(s) — Score: ${Number(item.route_score || 0).toFixed(1)}`)

      markersRef.current.push(m)
    })
  }, [mapReady, pickupCoord, dropoffCoord, groups])

  return (
    <div className="relative z-0 h-full w-full marshal-map">
      <div className="absolute inset-y-0 left-0 w-8 bg-gradient-to-r from-surface-container-low to-transparent z-10 pointer-events-none" />
      <div ref={containerRef} className="h-full w-full" />
    </div>
  )
}

export function ExplorePage({ activeRequest, isBusy, onSubmit, groups = [] }) {
  const [searchParams] = useSearchParams()
  const [name, setName]           = useState(activeRequest?.requester_name || '')
  const [pickupSearch, setPickup] = useState(null)
  const [dropoff, setDropoff]     = useState(null)
  const [arriveBy, setArriveBy]   = useState(getArrivalDefault)

  // When activeRequest is cleared (cancelled/expired), reset all form fields
  const prevRequestRef = useRef(activeRequest)
  useEffect(() => {
    const hadRequest = Boolean(prevRequestRef.current)
    const hasRequest = Boolean(activeRequest)
    prevRequestRef.current = activeRequest
    if (hadRequest && !hasRequest) {
      setName('')
      setPickup(null)
      setDropoff(null)
      setArriveBy(getArrivalDefault())
    }
  }, [activeRequest])

  // Reverse-geocode lat/lng → real place name when hydrating from activeRequest
  const [pickupName, setPickupName]   = useState('')
  const [dropoffName, setDropoffName] = useState('')

  useEffect(() => {
    if (pickupSearch || !activeRequest?.pickup_lat) return
    const url = `https://nominatim.openstreetmap.org/reverse?lat=${activeRequest.pickup_lat}&lon=${activeRequest.pickup_lng}&format=jsonv2`
    fetch(url)
      .then(r => r.json())
      .then(d => setPickupName(d.display_name || `${Number(activeRequest.pickup_lat).toFixed(5)}, ${Number(activeRequest.pickup_lng).toFixed(5)}`))
      .catch(() => setPickupName(`${Number(activeRequest.pickup_lat).toFixed(5)}, ${Number(activeRequest.pickup_lng).toFixed(5)}`))
  }, [activeRequest?.pickup_lat, activeRequest?.pickup_lng, pickupSearch])

  useEffect(() => {
    if (dropoff || !activeRequest?.dropoff_lat) return
    const url = `https://nominatim.openstreetmap.org/reverse?lat=${activeRequest.dropoff_lat}&lon=${activeRequest.dropoff_lng}&format=jsonv2`
    fetch(url)
      .then(r => r.json())
      .then(d => setDropoffName(d.display_name || `${Number(activeRequest.dropoff_lat).toFixed(5)}, ${Number(activeRequest.dropoff_lng).toFixed(5)}`))
      .catch(() => setDropoffName(`${Number(activeRequest.dropoff_lat).toFixed(5)}, ${Number(activeRequest.dropoff_lng).toFixed(5)}`))
  }, [activeRequest?.dropoff_lat, activeRequest?.dropoff_lng, dropoff])

  // Hydrate from activeRequest so pins persist after navigating away
  const effectivePickup = pickupSearch || (activeRequest?.pickup_lat ? {
    lat: activeRequest.pickup_lat,
    lng: activeRequest.pickup_lng,
    name: pickupName || `${Number(activeRequest.pickup_lat).toFixed(5)}, ${Number(activeRequest.pickup_lng).toFixed(5)}`,
  } : null)

  const effectiveDropoff = dropoff || (activeRequest?.dropoff_lat ? {
    lat: activeRequest.dropoff_lat,
    lng: activeRequest.dropoff_lng,
    name: dropoffName || `${Number(activeRequest.dropoff_lat).toFixed(5)}, ${Number(activeRequest.dropoff_lng).toFixed(5)}`,
  } : null)

  const canSubmit = useMemo(() => {
    const hasRoute = pickupSearch && dropoff &&
      (pickupSearch.lat !== dropoff.lat || pickupSearch.lng !== dropoff.lng)
    return name.trim() && !activeRequest && hasRoute
  }, [activeRequest, dropoff, name, pickupSearch])

  function handleSubmit(e) {
    e?.preventDefault()
    if (!canSubmit) return
    onSubmit({
      requester_name: name.trim(),
      pickup_lat:   pickupSearch.lat,
      pickup_lng:   pickupSearch.lng,
      dropoff_lat:  dropoff.lat,
      dropoff_lng:  dropoff.lng,
      arrive_by:    new Date(arriveBy).toISOString(),
    })
  }

  return (
    <div className="flex-1 h-full min-h-0 w-full flex flex-col md:flex-row relative z-10 overflow-hidden bg-background">
      {/* Left Panel: Request Form */}
      <section className="w-full md:w-[450px] lg:w-[500px] h-full bg-surface-container-low flex flex-col shrink-0 z-20 shadow-[4px_0_24px_rgba(46,50,48,0.08)] relative overflow-y-auto overflow-x-hidden pt-6 md:pt-8 pb-12 px-6 lg:px-10">
        <header className="mb-8">
          <h1 className="font-headline text-3xl md:text-4xl font-semibold text-on-surface mb-2">Request a Ride</h1>
          <p className="text-on-surface-variant font-body text-base">Find peers heading your way.</p>
        </header>

        <form className="flex-1 flex flex-col justify-between space-y-8" onSubmit={handleSubmit}>
          <div className="space-y-6">
            {/* Name */}
            <div>
              <label className="block text-sm font-semibold text-on-surface-variant mb-1.5 ml-1">Your Name</label>
              <input
                className="w-full px-4 py-3 rounded-xl border-none shadow-sm focus:ring-2 focus:ring-primary bg-surface-bright text-on-surface placeholder:text-on-surface-variant/50 text-base"
                placeholder="Student name"
                type="text"
                value={name}
                onChange={e => setName(e.target.value)}
                disabled={Boolean(activeRequest)}
              />
            </div>

            {/* Location Inputs */}
            <div className="space-y-5 relative">
              {/* Connecting Line */}
              <div className="absolute left-[23px] top-[30px] bottom-[30px] w-0.5 bg-outline-variant border-dashed border-l-2 opacity-50 z-0"></div>

              {/* Pickup */}
              <div className="relative z-10 flex items-start gap-4">
                <div className="mt-3 bg-surface-bright p-2 rounded-full shadow-sm border border-outline-variant shrink-0">
                  <span className="material-symbols-outlined text-primary">trip_origin</span>
                </div>
                <div className="flex-1">
                  <label className="block text-sm font-semibold text-on-surface-variant mb-1.5 ml-1">Pickup Location</label>
                  <LocationSearchField initialQuery={searchParams.get('pickup') || ''} label="Pickup" disabled={Boolean(activeRequest)} value={effectivePickup} onSelect={setPickup} />
                </div>
              </div>

              {/* Dropoff */}
              <div className="relative z-10 flex items-start gap-4">
                <div className="mt-3 bg-surface-bright p-2 rounded-full shadow-sm border border-outline-variant shrink-0">
                  <span className="material-symbols-outlined text-tertiary">location_on</span>
                </div>
                <div className="flex-1">
                  <label className="block text-sm font-semibold text-on-surface-variant mb-1.5 ml-1">Dropoff Location</label>
                  <LocationSearchField initialQuery={searchParams.get('dropoff') || ''} label="Dropoff" disabled={Boolean(activeRequest)} value={effectiveDropoff} onSelect={setDropoff} />
                </div>
              </div>
            </div>

            <hr className="border-outline-variant/30 my-6" />

            {/* Arrive By */}
            <div>
              <label className="block text-sm font-semibold text-on-surface-variant mb-1.5 ml-1">Arrive By</label>
              <div className="relative">
                <span className="material-symbols-outlined absolute left-3 top-1/2 -translate-y-1/2 text-on-surface-variant/70 pointer-events-none">schedule</span>
                <input
                  type="datetime-local"
                  className="w-full pl-10 pr-4 py-3 rounded-xl border-none shadow-sm focus:ring-2 focus:ring-primary bg-surface-bright text-on-surface text-base"
                  value={arriveBy}
                  onChange={e => setArriveBy(e.target.value)}
                  disabled={Boolean(activeRequest)}
                />
              </div>
            </div>
          </div>

          {/* CTA Button at bottom */}
          <div className="mt-auto pt-6 mb-2">
            <button
              type="submit"
              disabled={!canSubmit || isBusy}
              className="w-full bg-primary hover:bg-primary/90 text-on-primary font-headline font-bold text-lg py-4 px-6 rounded-2xl shadow-sm hover:shadow-md transition-all active:scale-[0.98] flex items-center justify-center gap-2 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              <span className="material-symbols-outlined">search</span>
              {activeRequest ? 'Request Active' : 'Find Available Rides'}
            </button>
          </div>
        </form>
      </section>

      {/* Right Panel: Interactive Map */}
      <section className="flex-1 relative bg-surface-container h-full hidden md:block">
        <LeafletMap pickupCoord={effectivePickup} dropoffCoord={effectiveDropoff} groups={groups} />
      </section>
    </div>
  )
}
