<template>
  <div class="result">
    <div class="score-row">
      <span class="score-label">Severity Score</span>
      <span class="score-badge" :class="scoreSeverity">{{ score }} / 10</span>
    </div>
    <div class="advice-box">
      <p class="advice-label">Advice</p>
      <p class="advice-text">{{ advice }}</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  score: number
  advice: string
}>()

const scoreSeverity = computed(() => {
  if (props.score >= 8) return 'high'
  if (props.score >= 5) return 'medium'
  return 'low'
})
</script>

<style scoped>
.result {
  display: flex;
  flex-direction: column;
  gap: 1rem;
  padding: 1.25rem;
  background: #f9fafb;
  border: 1px solid #e5e7eb;
  border-radius: 10px;
}

.score-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.score-label {
  font-size: 0.875rem;
  font-weight: 500;
  color: #6b7280;
}

.score-badge {
  font-size: 1.1rem;
  font-weight: 700;
  padding: 0.25rem 0.75rem;
  border-radius: 999px;
}

.score-badge.low    { background: #dcfce7; color: #15803d; }
.score-badge.medium { background: #fef9c3; color: #a16207; }
.score-badge.high   { background: #fee2e2; color: #b91c1c; }

.advice-box {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.advice-label {
  font-size: 0.8rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: #9ca3af;
}

.advice-text {
  font-size: 0.95rem;
  color: #374151;
  line-height: 1.6;
}
</style>
