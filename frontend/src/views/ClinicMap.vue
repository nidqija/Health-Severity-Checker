<script setup lang="ts">
import { computed } from 'vue'
import { clinicStore } from '../store/clinicStore'

const mapSrc = computed(() => {
  const base = 'https://www.google.com/maps/embed/v1'
  const key = (import.meta as any).env?.VITE_GOOGLE_MAPS_API_KEY ?? ''

  if (clinicStore.confirmedClinic) {
    const { place_id, name } = clinicStore.confirmedClinic
    if (place_id) {
      return `${base}/place?key=${key}&q=place_id:${place_id}&zoom=16`
    }
    return `${base}/search?key=${key}&q=${encodeURIComponent(name)}&zoom=15`
  }

  if (clinicStore.clinics.length > 0) {
    const lat = clinicStore.userLat ?? 3.139
    const lng = clinicStore.userLng ?? 101.6869
    return `${base}/search?key=${key}&q=clinic+hospital&center=${lat},${lng}&zoom=13`
  }

  return `${base}/search?key=${key}&q=clinic+hospital&center=3.139,101.6869&zoom=13`
})

const hasData = computed(
  () => clinicStore.clinics.length > 0 || clinicStore.confirmedClinic !== null
)
</script>

<template>
  <div class="map-view">
    <div v-if="!hasData" class="empty-state">
      <div class="empty-icon">🗺️</div>
      <p class="empty-title">No clinics loaded yet</p>
      <p class="empty-sub">Complete a triage assessment with a high severity score to see nearby clinics here.</p>
    </div>

    <template v-else>
      <div v-if="clinicStore.confirmedClinic" class="confirmed-banner">
        <span class="confirmed-icon">✅</span>
        <div>
          <p class="confirmed-name">{{ clinicStore.confirmedClinic.name }}</p>
          <p class="confirmed-address">{{ clinicStore.confirmedClinic.address }}</p>
        </div>
      </div>

      <div v-if="clinicStore.clinics.length > 0 && !clinicStore.confirmedClinic" class="clinic-pills">
        <span v-for="c in clinicStore.clinics" :key="c.place_id" class="pill">
          {{ c.name }}
        </span>
      </div>

      <div class="map-wrapper">
        <iframe
          :src="mapSrc"
          class="map-frame"
          allowfullscreen
          loading="lazy"
          referrerpolicy="no-referrer-when-downgrade"
          title="Clinic map"
        />
      </div>
    </template>
  </div>
</template>

<style scoped>
.map-view { display: flex; flex-direction: column; gap: 1rem; }

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 3rem 1rem;
  text-align: center;
  gap: 0.5rem;
}
.empty-icon { font-size: 3rem; }
.empty-title { font-size: 1.1rem; font-weight: 600; color: #374151; }
.empty-sub { font-size: 0.9rem; color: #9ca3af; max-width: 320px; }

.confirmed-banner {
  display: flex;
  align-items: flex-start;
  gap: 0.75rem;
  padding: 0.875rem 1rem;
  background: #f0fdf4;
  border: 1px solid #bbf7d0;
  border-radius: 10px;
}
.confirmed-icon { font-size: 1.25rem; }
.confirmed-name { font-weight: 600; color: #111827; font-size: 0.95rem; }
.confirmed-address { font-size: 0.82rem; color: #6b7280; margin-top: 0.1rem; }

.clinic-pills { display: flex; flex-wrap: wrap; gap: 0.4rem; }
.pill {
  padding: 0.3rem 0.75rem;
  background: #ede9fe;
  color: #4f46e5;
  border-radius: 999px;
  font-size: 0.8rem;
  font-weight: 500;
}

.map-wrapper {
  border-radius: 12px;
  overflow: hidden;
  border: 1px solid #e5e7eb;
  box-shadow: 0 1px 4px rgba(0,0,0,0.06);
}
.map-frame { width: 100%; height: 480px; border: none; display: block; }
</style>
