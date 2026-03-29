<script setup lang="ts">
defineProps<{
  name: string
  address: string
  rating: number
  openNow: boolean
  placeId: string
}>()

const emit = defineEmits<{
  (e: 'approve'): void
  (e: 'reject'): void
}>()

function navigate(placeId: string) {
  window.open(
    `https://www.google.com/maps/search/?api=1&query=Google&query_place_id=${placeId}`,
    '_blank'
  )
}
</script>

<template>
  <div class="clinic-card">
    <div class="clinic-info">
      <span class="clinic-name">{{ name }}</span>
      <span class="clinic-status" :class="openNow ? 'open' : 'closed'">
        {{ openNow ? 'Open now' : 'Closed' }}
      </span>
      <span class="clinic-address">{{ address }}</span>
      <span class="clinic-rating">⭐ {{ rating > 0 ? rating.toFixed(1) : 'N/A' }}</span>
    </div>
    <div class="clinic-actions">
      <button class="btn-navigate" @click="navigate(placeId)">🗺 Navigate</button>
      <button class="btn-approve" @click="emit('approve')" aria-label="Approve">✔</button>
      <button class="btn-reject" @click="emit('reject')" aria-label="Reject">✕</button>
    </div>
  </div>
</template>

<style scoped>
.clinic-card {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 1rem;
  padding: 1rem;
  background: #fff;
  border: 1px solid #e5e7eb;
  border-radius: 10px;
}

.clinic-info {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.clinic-name {
  font-weight: 600;
  font-size: 0.95rem;
  color: #111827;
}

.clinic-address {
  font-size: 0.85rem;
  color: #6b7280;
}

.clinic-rating {
  font-size: 0.85rem;
  color: #374151;
}

.clinic-status {
  font-size: 0.75rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.04em;
}
.clinic-status.open  { color: #16a34a; }
.clinic-status.closed { color: #dc2626; }

.clinic-actions {
  display: flex;
  gap: 0.5rem;
  align-items: center;
  flex-shrink: 0;
}

.btn-navigate {
  padding: 0.4rem 0.75rem;
  background: #4f46e5;
  color: #fff;
  border: none;
  border-radius: 6px;
  font-size: 0.85rem;
  cursor: pointer;
}
.btn-navigate:hover { background: #4338ca; }

.btn-approve {
  padding: 0.4rem 0.6rem;
  background: #16a34a;
  color: #fff;
  border: none;
  border-radius: 6px;
  font-size: 0.9rem;
  cursor: pointer;
}
.btn-approve:hover { background: #15803d; }

.btn-reject {
  padding: 0.4rem 0.6rem;
  background: #dc2626;
  color: #fff;
  border: none;
  border-radius: 6px;
  font-size: 0.9rem;
  cursor: pointer;
}
.btn-reject:hover { background: #b91c1c; }
</style>
