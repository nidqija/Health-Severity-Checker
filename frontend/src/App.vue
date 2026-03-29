<script setup lang="ts">
import SeverityChecker from './views/SeverityChecker.vue'
</script>

<template>
  <div class="app">
    <header class="header">
      <div class="logo">🏥 MediQuick</div>
      <h1>AI Triage System</h1>
      <p>Describe your symptoms or upload an image for an instant severity assessment.</p>
    </header>
    <main class="main">
      <SeverityChecker />
    </main>
  </div>
</template>

<style scoped>
.app {
  max-width: 720px;
  margin: 0 auto;
  padding: 2rem 1.5rem;
}

.header {
  text-align: center;
  margin-bottom: 2.5rem;
}

.header h1 {
  font-size: 2rem;
  font-weight: 700;
  color: #1a1a2e;
  margin-bottom: 0.5rem;
}

.logo {
  font-size: 1.1rem;
  font-weight: 700;
  color: #4f46e5;
  letter-spacing: 0.02em;
  margin-bottom: 0.25rem;
}

.header p {
  font-size: 1rem;
  color: #6b7280;
}

.main {
  background: #ffffff;
  border-radius: 12px;
  padding: 2rem;
  box-shadow: 0 1px 3px rgba(0,0,0,0.08), 0 4px 16px rgba(0,0,0,0.06);
}
</style>
