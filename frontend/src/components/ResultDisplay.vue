<template>
  <div>
    <div>Score: {{ score }}</div>
    <div>Advice: {{ advice }}</div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  score: number
  advice: string
}>()
</script>
