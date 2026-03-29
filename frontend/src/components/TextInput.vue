<template>
  <textarea
    class="text-input"
    :value="modelValue"
    :disabled="disabled"
    placeholder="e.g. I have a persistent headache and mild fever for 3 days…"
    @input="$emit('update:modelValue', ($event.target as HTMLTextAreaElement).value)"
  />
</template>

<script setup lang="ts">
defineProps<{
  modelValue: string
  disabled?: boolean
}>()

defineEmits<{
  (e: 'update:modelValue', value: string): void
}>()
</script>

<style scoped>
.text-input {
  width: 100%;
  min-height: 120px;
  padding: 0.75rem 1rem;
  font-family: inherit;
  font-size: 0.95rem;
  color: #1a1a2e;
  background: #f9fafb;
  border: 1px solid #d1d5db;
  border-radius: 8px;
  resize: vertical;
  transition: border-color 0.15s, box-shadow 0.15s;
  outline: none;
}

.text-input:focus {
  border-color: #4f46e5;
  box-shadow: 0 0 0 3px rgba(79, 70, 229, 0.15);
  background: #fff;
}

.text-input:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}
</style>
