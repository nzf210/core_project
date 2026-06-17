<template>
  <div class="view-container">
    <header class="view-header">
      <h1>Plan Features (Matrix)</h1>
    </header>

    <div v-if="loading" class="loading">Loading plans...</div>

    <div v-else class="plans-grid">
      <div class="plan-card" v-for="plan in plans" :key="plan.id">
        <h2>{{ plan.name.toUpperCase() }}</h2>
        <form @submit.prevent="savePlan(plan.id)">
          <div class="form-group" v-for="key in featureKeys" :key="key">
            <label>{{ key }}</label>
            <input type="number" v-model.number="formStates[plan.id][key]" />
          </div>
          <button type="submit" class="btn primary">Save Changes</button>
        </form>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { isAuthed, api } from '../api/client'

const loading = ref(true)
const plans = ref<any[]>([])
const formStates = ref<Record<string, Record<string, number>>>({})

const featureKeys = [
  'max_users', 'max_transactions', 'max_ai_text',
  'max_ai_vision', 'max_ai_audio_minutes', 'max_image_gen',
  'max_products', 'max_customers', 'max_storage_mb',
  'api_rate_limit_per_min', 'data_retention_months'
]

async function loadData() {
  try {
    if (!isAuthed()) return
    const res = await api.listPlans()
    plans.value = res.data || []

    // Fetch current numeric limits per plan from the matrix endpoint
    await Promise.all(plans.value.map(async (p: any) => {
      try {
        const matrix = await api.fetchPlanFeatureMatrix(p.id)
        formStates.value[p.id] = {}
        featureKeys.forEach(k => {
          // Fallback to 0 if not present
          formStates.value[p.id][k] = (matrix.data as any)?.[k] ?? 0
        })
      } catch {
        // Fallback to plan object's own keys (all zero if not present)
        formStates.value[p.id] = {}
        featureKeys.forEach(k => {
          formStates.value[p.id][k] = (p as any)[k] ?? 0
        })
      }
    }))
  } catch (err) {
    alert("Failed to load plans")
  } finally {
    loading.value = false
  }
}

async function savePlan(planId: string) {
  try {
    const payload = formStates.value[planId]
    await api.updatePlanFeatureNumeric(planId, payload)
    alert(`Plan ${planId} updated successfully!`)
  } catch (err) {
    alert(`Failed to update ${planId}`)
  }
}

onMounted(() => {
  loadData()
})
</script>

<style scoped>
.plans-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
  gap: 1.5rem;
}
.plan-card {
  background: white;
  border-radius: 8px;
  padding: 1.5rem;
  box-shadow: 0 1px 3px rgba(0,0,0,0.1);
}
.form-group {
  margin-bottom: 0.75rem;
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.form-group label {
  font-size: 0.85rem;
  color: #555;
  font-family: monospace;
}
.form-group input {
  width: 100px;
  padding: 0.25rem;
  text-align: right;
  border: 1px solid #ccc;
  border-radius: 4px;
}
.btn {
  width: 100%;
  margin-top: 1rem;
}
</style>
