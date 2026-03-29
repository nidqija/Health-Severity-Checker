<script setup lang="ts">
defineProps<{
  visible: boolean
}>()
</script>

<template>
  <button
    v-if="visible"
    style="background-color: red; min-width: 200px; min-height: 48px"
  >
    Book Appointment
  </button>
</template>
