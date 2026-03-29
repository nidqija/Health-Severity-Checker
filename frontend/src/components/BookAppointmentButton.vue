<script setup lang="ts">
defineProps<{
  visible: boolean
}>()
</script>

<template>
  <button
    v-if="visible"
    class="book-btn"
    style="min-width: 200px; min-height: 48px; background-color: red"
  >
    📅 Book Appointment
  </button>
</template>

<style scoped>
.book-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  min-width: 200px;
  min-height: 48px;
  padding: 0.75rem 1.5rem;
  background: #dc2626;
  color: #fff;
  font-family: inherit;
  font-size: 1rem;
  font-weight: 600;
  border: none;
  border-radius: 8px;
  cursor: pointer;
  transition: background 0.15s, transform 0.1s;
  align-self: flex-start;
}

.book-btn:hover {
  background: #b91c1c;
  transform: translateY(-1px);
}

.book-btn:active {
  transform: translateY(0);
}
</style>
