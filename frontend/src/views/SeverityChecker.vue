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
  <div class="checker">
    <label class="label" for="symptom-input">Describe your symptoms</label>
    <TextInput id="symptom-input" v-model="inputText" :disabled="loading" />

    <span v-if="validationMessage" class="validation">{{ validationMessage }}</span>

    <button class="submit-btn" :disabled="loading" @click="handleSubmit">
      <span v-if="loading" class="spinner" aria-hidden="true" />
      {{ loading ? 'Analysing…' : 'Check Severity' }}
    </button>

    <ResultDisplay v-if="result" :score="result.score" :advice="result.advice" />
    <BookAppointmentButton :visible="(result?.score ?? 0) >= 8" />

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

.submit-btn:hover:not(:disabled) {
  background: #4338ca;
}

.submit-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.spinner {
  width: 16px;
  height: 16px;
  border: 2px solid rgba(255,255,255,0.4);
  border-top-color: #fff;
  border-radius: 50%;
  animation: spin 0.7s linear infinite;
  flex-shrink: 0;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.validation {
  font-size: 0.85rem;
  color: #dc2626;
}

.error-box {
  padding: 0.875rem 1rem;
  background: #fef2f2;
  border: 1px solid #fecaca;
  border-radius: 8px;
  color: #b91c1c;
  font-size: 0.9rem;
}
</style>
