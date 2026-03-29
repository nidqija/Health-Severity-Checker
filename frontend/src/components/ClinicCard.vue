<script setup lang="ts">
defineProps<{
  name: string
  address: string
  rating: number
  openNow: boolean
  placeId: string
}>()

const emit = defineEmits<{
  approve: []
  reject: []
}>()

function navigate(placeId: string) {
  window.open(
    `https://www.google.com/maps/search/?api=1&query=Google&query_place_id=${placeId}`,
    '_blank',
    'noopener,noreferrer'
  )
}

function stars(rating: number): string {
  const full = Math.round(rating)
  return '★'.repeat(full) + '☆'.repeat(Math.max(0, 5 - full))
}
</script>

<template>
  <div class="card">
    <div class="card-header">
      <div class="card-info">
        <p class="card-name">{{ name }}</p>
        <p class="card-address">{{ address }}</p>
        <div class="card-meta">
          <span class="stars">{{ stars(rating) }}</span>
          <span class="rating-num">{{ rating > 0 ? rating.toFixed(1) : 'N/A' }}</span>
          <span class="open-badge" :class="openNow ? 'open' : 'closed'">
            {{ openNow ? 'Open' : 'Closed' }}
          </span>
        </div>
      </div>
    </div>
    <div class="card-actions">
      <button class="btn approve" title="Approve" @click="emit('approve')">✓ Approve</button>
      <button class="btn navigate" title="Navigate" @click="navigate(placeId)">🗺 Navigate</button>
      <button class="btn reject" title="Reject" @click="emit('reject')">✕</button>
    </div>
  </div>
</template>

<style scoped>
.card {
  background: #fff;
  border: 1px solid #e5e7eb;
  border-radius: 10px;
  padding: 1rem;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  box-shadow: 0 1px 3px rgba(0,0,0,0.05);
}

.card-name {
  font-weight: 600;
  font-size: 0.95rem;
  color: #111827;
}

.card-address {
  font-size: 0.82rem;
  color: #6b7280;
  margin-top: 0.15rem;
}

.card-meta {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin-top: 0.35rem;
}

.stars { color: #f59e0b; font-size: 0.85rem; letter-spacing: 1px; }
.rating-num { font-size: 0.82rem; color: #374151; }

.open-badge {
  font-size: 0.75rem;
  font-weight: 600;
  padding: 0.15rem 0.5rem;
  border-radius: 999px;
}
.open-badge.open   { background: #dcfce7; color: #15803d; }
.open-badge.closed { background: #fee2e2; color: #b91c1c; }

.card-actions {
  display: flex;
  gap: 0.5rem;
}

.btn {
  padding: 0.45rem 0.875rem;
  border: none;
  border-radius: 6px;
  font-family: inherit;
  font-size: 0.85rem;
  font-weight: 500;
  cursor: pointer;
  transition: opacity 0.15s;
}
.btn:hover { opacity: 0.85; }

.btn.approve  { background: #4f46e5; color: #fff; }
.btn.navigate { background: #0ea5e9; color: #fff; }
.btn.reject   { background: #f3f4f6; color: #374151; margin-left: auto; }
</style>
