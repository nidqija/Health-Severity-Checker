export interface Clinic {
  name: string
  address: string
  rating: number
  open_now: boolean
  place_id: string
}

// Klang Valley / KL city centre fallback coords
const KL_CENTER = { lat: 3.139, lng: 101.6869 }

// Valid place types from the new Places API (Table A)
const PLACE_TYPES = ['hospital', 'medical_clinic', 'pharmacy', 'doctor', 'general_hospital']

function waitForMaps(): Promise<void> {
  return new Promise((resolve) => {
    if ((window as any).__mapsReady) return resolve()
    ;((window as any).__mapsReadyCallbacks as (() => void)[]).push(resolve)
  })
}

async function searchNearbyNew(
  location: google.maps.LatLngLiteral,
  radius: number,
  type: string
): Promise<Clinic[]> {
  const { Place } = (await google.maps.importLibrary('places')) as google.maps.PlacesLibrary

  const request: google.maps.places.SearchNearbyRequest = {
    // JS SDK field names (camelCase)
    fields: ['displayName', 'formattedAddress', 'rating', 'regularOpeningHours', 'id'],
    locationRestriction: { center: location, radius },
    includedPrimaryTypes: [type],
    maxResultCount: 5,
  }

  try {
    const { places } = await Place.searchNearby(request)
    return places.map((p) => ({
      name: p.displayName ?? '',
      address: p.formattedAddress ?? '',
      rating: p.rating ?? 0,
      open_now: (p.regularOpeningHours?.weekdayDescriptions?.length ?? 0) > 0,
      place_id: p.id ?? '',
    }))
  } catch {
    return []
  }
}

export async function findNearbyClinics(lat: number, lng: number): Promise<Clinic[]> {
  await waitForMaps()

  const attempts: { location: google.maps.LatLngLiteral; radius: number }[] = [
    { location: { lat, lng }, radius: 10000 },
    { location: { lat, lng }, radius: 25000 },
    { location: KL_CENTER, radius: 20000 },
  ]

  const seen = new Set<string>()
  const clinics: Clinic[] = []

  for (const attempt of attempts) {
    if (clinics.length >= 5) break

    for (const type of PLACE_TYPES) {
      if (clinics.length >= 5) break
      const results = await searchNearbyNew(attempt.location, attempt.radius, type)
      for (const r of results) {
        if (!r.place_id || seen.has(r.place_id)) continue
        seen.add(r.place_id)
        clinics.push(r)
        if (clinics.length >= 5) break
      }
    }

    if (clinics.length > 0) break
  }

  return clinics
}
