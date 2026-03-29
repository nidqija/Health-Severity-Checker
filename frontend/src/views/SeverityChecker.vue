<script setup lang="ts">
import { ref } from 'vue'
import TextInput from '../components/TextInput.vue'
import ResultDisplay from '../components/ResultDisplay.vue'
import BookAppointmentButton from '../components/BookAppointmentButton.vue'

const inputText = ref('')
const loading = ref(false)
const result = ref<{ score: number; advice: string } | null>(null)
const error = ref<string | null>(null)
const validationMessage = ref<string | null>(null)

async function handleSubmit() {
  if (!inputText.value.trim()) {
    validationMessage.value = 'Text is required'
    return
  }

  validationMessage.value = null
  loading.value = true
  result.value = null
  error.value = null

  try {
    const response = await fetch('/analyze', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ text: inputText.value }),
    })

    if (response.ok) {
      result.value = await response.json()
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
</script>

<template>
  <div>
    <TextInput v-model="inputText" :disabled="loading" />
    <button :disabled="loading" @click="handleSubmit">Check Severity</button>
    <span v-if="loading">Loading...</span>
    <span v-if="validationMessage">{{ validationMessage }}</span>
    <ResultDisplay v-if="result" :score="result.score" :advice="result.advice" />
    <BookAppointmentButton :visible="(result?.score ?? 0) >= 8" />
    <span v-if="error">{{ error }}</span>
  </div>
</template>
