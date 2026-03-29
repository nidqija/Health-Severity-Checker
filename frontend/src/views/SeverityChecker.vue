<script setup lang="ts">
import { ref, computed } from 'vue'
import TextInput from '../components/TextInput.vue'
import ResultDisplay from '../components/ResultDisplay.vue'
import BookAppointmentButton from '../components/BookAppointmentButton.vue'
import ClinicCard from '../components/ClinicCard.vue'
import { findNearbyClinics, type Clinic } from '../places'

const inputText = ref('')
const loading = ref(false)
const result = ref<{ score: number; advice: string } | null>(null)
const error = ref<string | null>(null)
const validationMessage = ref<string | null>(null)

// Clinic state
const clinics = ref<Clinic[]>([])
const rejectedIds = ref<Set<string>>(new Set())
const confirmedClinic = ref<Clinic | null>(null)
const locationStatus = ref<'idle' | 'requesting' | 'denied' | 'ok'>('idle')
const manualLocation = ref('')
const showManualInput = ref(false)
const clinicsLoading = ref(false)
const clinicError = ref<string | null>(null)

const visibleClinics = computed(() =>
  clinics.value.filter((c) => !rejectedIds.value.has(c.place_id))
)

const isHighSeverity = computed(() => (result.value?.score ?? 0) >= 8)

async function handleSubmit() {
  if (!inputText.value.trim()) {
    validationMessage.value = 'Text is required'
    return
  }

  validationMessage.value = null
  loading.value = true
  result.value = null
  error.value = null
  clinics.value = []
  rejectedIds.value = new Set()
  confirmedClinic.value = null
  locationStatus.value = 'idle'
  showManualInput.value = false
  clinicError.value = null

  try {
    const response = await fetch('/analyze', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ text: inputText.value }),
    })

    if (response.ok) {
      result.value = await response.json()
      if ((result.value?.score ?? 0) >= 8) {
        requestGeolocation()
      }
    } else if (response.status === 502) {
      error.value = 'The analysis service is currently unavailable.'
    } else if (response.status === 422) {
      error.value = 'The analysis returned an unexpected result. Please try again.'
    } else {
      error.value = `An unexpected error occurred (HTTP ${response.status}).`
    }
  } catch {
    error.value = 'Could not reach the server. Please try again.'
  } finally {
    loading.value = false
  }
}

function requestGeolocation() {
  if (!navigator.geolocation) {
    showManualInput.value = true
    locationStatus.value = 'denied'
    return
  }
  locationStatus.value = 'requesting'
  navigator.geolocation.getCurrentPosition(
    (pos) => {
      locationStatus.value = 'ok'
      fetchClinics(pos.coords.latitude, pos.coords.longitude)
    },
    () => {
      locationStatus.value = 'denied'
      showManualInput.value = true
    }
  )
}

async function fetchClinics(lat: number, lng: number) {
  clinicsLoading.value = true
  clinicError.value = null
  try {
    clinics.value = await findNearbyClinics(lat, lng)
  } catch (e) {
    clinicError.value = 'Could not load nearby clinics.'
  } finally {
    clinicsLoading.value = false
  }
}

async function searchByManualLocation() {
  if (!manualLocation.value.trim()) return
  // Use KL centre as a proxy for "Klang Valley" when no coords available
  clinicsLoading.value = true
  clinicError.value = null
  try {
    clinics.value = await findNearbyClinics(3.139, 101.6869)
  } catch {
    clinicError.value = 'Could not load nearby clinics.'
  } finally {
    clinicsLoading.value = false
  }
}

function approveClinic(clinic: Clinic) {
  confirmedClinic.value = clinic
}

function rejectClinic(placeId: string) {
  rejectedIds.value = new Set([...rejectedIds.value, placeId])
}
</script>

<template>
  <div class="checker">
    <label class="label" for="symptom-input">Describe your symptoms</label>
    <TextInput id="symptom-input" v-model="inputText" :disabled="loading" />

    <span v-if="validationMessage" class="validation">{{ validationMessage }}</span>

    <button class="submit-btn" :disabled="loading" @click="handleSubmit">
      <span v-if="loading" class="spinner" aria-hidden="true" />
      <span v-if="loading">Loading...</span>
      <span v-else>Check Severity</span>
    </button>

    <ResultDisplay v-if="result" :score="result.score" :advice="result.advice" />
    <BookAppointmentButton :visible="isHighSeverity" />

    <!-- Geolocation status -->
    <div v-if="isHighSeverity && locationStatus === 'requesting'" class="location-status">
      📍 Requesting your location…
    </div>
    <div v-if="isHighSeverity && locationStatus === 'denied'" class="location-status denied">
      📍 Location access denied or unavailable.
    </div>

    <!-- Geolocation / manual fallback -->
    <div v-if="isHighSeverity && showManualInput" class="manual-location">
      <label class="label" for="location-input">Enter your city or postcode to find nearby clinics</label>
      <div class="location-row">
        <input
          id="location-input"
          v-model="manualLocation"
          class="location-input"
          placeholder="e.g. Kuala Lumpur or 50450"
          @keyup.enter="searchByManualLocation"
        />
        <button class="search-btn" @click="searchByManualLocation">Search</button>
      </div>
    </div>

    <!-- Clinic list -->
    <div v-if="isHighSeverity && clinicsLoading" class="clinics-loading">Finding nearby clinics…</div>

    <div v-if="isHighSeverity && clinicError" class="clinic-error">⚠ {{ clinicError }}</div>

    <div v-if="isHighSeverity && !clinicsLoading && !clinicError && clinics.length === 0 && locationStatus === 'ok'" class="clinics-loading">
      No clinics found nearby.
    </div>

    <div v-if="isHighSeverity && !clinicsLoading && visibleClinics.length > 0 && !confirmedClinic" class="clinic-list">
      <p class="clinic-list-title">Nearby clinics</p>
      <ClinicCard
        v-for="clinic in visibleClinics"
        :key="clinic.place_id"
        :name="clinic.name"
        :address="clinic.address"
        :rating="clinic.rating"
        :open-now="clinic.open_now"
        :place-id="clinic.place_id"
        @approve="approveClinic(clinic)"
        @reject="rejectClinic(clinic.place_id)"
      />
    </div>

    <!-- Visit confirmed summary -->
    <div v-if="confirmedClinic" class="confirmed-box">
      <p class="confirmed-title">✅ Visit Confirmed</p>
      <p class="confirmed-name">{{ confirmedClinic.name }}</p>
      <p class="confirmed-address">{{ confirmedClinic.address }}</p>
    </div>

    <div v-if="error" class="error-box" role="alert">{{ error }}</div>
  </div>
</template>

<style scoped>
.checker {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.label {
  font-size: 0.875rem;
  font-weight: 500;
  color: #374151;
}

.submit-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  padding: 0.75rem 1.5rem;
  background: #4f46e5;
  color: #fff;
  font-family: inherit;
  font-size: 0.95rem;
  font-weight: 600;
  border: none;
  border-radius: 8px;
  cursor: pointer;
  transition: background 0.15s;
  align-self: flex-start;
}

.submit-btn:hover:not(:disabled) { background: #4338ca; }
.submit-btn:disabled { opacity: 0.6; cursor: not-allowed; }

.spinner {
  width: 16px;
  height: 16px;
  border: 2px solid rgba(255,255,255,0.4);
  border-top-color: #fff;
  border-radius: 50%;
  animation: spin 0.7s linear infinite;
  flex-shrink: 0;
}

@keyframes spin { to { transform: rotate(360deg); } }

.validation { font-size: 0.85rem; color: #dc2626; }

.error-box {
  padding: 0.875rem 1rem;
  background: #fef2f2;
  border: 1px solid #fecaca;
  border-radius: 8px;
  color: #b91c1c;
  font-size: 0.9rem;
}

.manual-location {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.location-row {
  display: flex;
  gap: 0.5rem;
}

.location-input {
  flex: 1;
  padding: 0.6rem 0.875rem;
  border: 1px solid #d1d5db;
  border-radius: 8px;
  font-family: inherit;
  font-size: 0.9rem;
  outline: none;
}
.location-input:focus { border-color: #4f46e5; }

.search-btn {
  padding: 0.6rem 1rem;
  background: #4f46e5;
  color: #fff;
  border: none;
  border-radius: 8px;
  font-size: 0.9rem;
  cursor: pointer;
}
.search-btn:hover { background: #4338ca; }

.clinics-loading {
  font-size: 0.9rem;
  color: #6b7280;
}

.location-status {
  font-size: 0.85rem;
  color: #6b7280;
}
.location-status.denied {
  color: #b91c1c;
}

.clinic-error {
  font-size: 0.875rem;
  color: #b91c1c;
  padding: 0.75rem 1rem;
  background: #fef2f2;
  border: 1px solid #fecaca;
  border-radius: 8px;
}

.clinic-list {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.clinic-list-title {
  font-size: 0.875rem;
  font-weight: 600;
  color: #374151;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.confirmed-box {
  padding: 1rem;
  background: #f0fdf4;
  border: 1px solid #bbf7d0;
  border-radius: 10px;
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.confirmed-title {
  font-weight: 700;
  color: #15803d;
  font-size: 1rem;
}

.confirmed-name {
  font-weight: 600;
  color: #111827;
}

.confirmed-address {
  font-size: 0.85rem;
  color: #6b7280;
}
</style>
