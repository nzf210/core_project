<template>
  <div class="view-container">
    <header class="view-header">
      <div>
        <h1>Plan Numeric Limits</h1>
        <p class="subtitle">Atur batas kuantitas (max_users, max_transactions, dll.) per plan. Untuk toggle fitur on/off gunakan <a href="/feature-matrix">Feature Matrix</a>.</p>
      </div>
    </header>

    <div v-if="loading" class="loading">Memuat plans...</div>

    <div v-else class="plans-grid">
      <div class="plan-card" v-for="plan in plans" :key="plan.id">
        <div class="plan-card-header">
          <h2 :class="['plan-name', plan.name.toLowerCase()]">{{ plan.name.toUpperCase() }}</h2>
          <span class="plan-id"><code>{{ plan.id }}</code></span>
        </div>
        <form @submit.prevent="savePlan(plan.id)">
          <div class="form-group" v-for="key in featureKeys" :key="key">
            <label>{{ formatKey(key) }}</label>
            <input type="number" v-model.number="formStates[plan.id][key]" min="0" />
          </div>
          <button type="submit" class="btn-save" :disabled="saving[plan.id]">
            {{ saving[plan.id] ? '⏳ Menyimpan...' : '💾 Simpan' }}
          </button>
        </form>
        <p v-if="saveMsg[plan.id]" :class="['save-msg', saveMsgType[plan.id]]">{{ saveMsg[plan.id] }}</p>
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
const saving = ref<Record<string, boolean>>({})
const saveMsg = ref<Record<string, string>>({})
const saveMsgType = ref<Record<string, string>>({})

const featureKeys = [
  'max_users', 'max_transactions', 'max_ai_text',
  'max_ai_vision', 'max_ai_audio_minutes', 'max_image_gen',
  'max_products', 'max_customers', 'max_storage_mb',
  'api_rate_limit_per_min', 'data_retention_months'
]

function formatKey(key: string): string {
  return key.replace(/_/g, ' ').replace(/\b\w/g, c => c.toUpperCase())
}

async function loadData() {
  try {
    if (!isAuthed()) return
    const res = await api.listPlans()
    plans.value = res.data || []

    await Promise.all(plans.value.map(async (p: any) => {
      try {
        const matrix = await api.fetchPlanFeatureMatrix(p.id)
        formStates.value[p.id] = {}
        featureKeys.forEach(k => {
          formStates.value[p.id][k] = (matrix.data as any)?.[k] ?? 0
        })
      } catch {
        formStates.value[p.id] = {}
        featureKeys.forEach(k => {
          formStates.value[p.id][k] = (p as any)[k] ?? 0
        })
      }
    }))
  } catch (err) {
    alert('Gagal memuat data plans')
  } finally {
    loading.value = false
  }
}

async function savePlan(planId: string) {
  saving.value[planId] = true
  saveMsg.value[planId] = ''
  try {
    const payload = formStates.value[planId]
    await api.updatePlanFeatureNumeric(planId, payload)
    saveMsg.value[planId] = '✅ Berhasil disimpan'
    saveMsgType.value[planId] = 'success'
    setTimeout(() => { saveMsg.value[planId] = '' }, 3000)
  } catch (err) {
    saveMsg.value[planId] = '❌ Gagal menyimpan'
    saveMsgType.value[planId] = 'error'
  } finally {
    saving.value[planId] = false
  }
}

onMounted(() => {
  loadData()
})
</script>

<style scoped>
.view-container { }

.view-header { margin-bottom: 24px; }
h1 { font-size: 22px; margin-bottom: 6px; }
.subtitle { font-size: 13px; color: var(--muted); }
.subtitle a { color: var(--accent); }

.loading { padding: 40px; text-align: center; color: var(--muted); }

.plans-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
  gap: 20px;
}

.plan-card {
  background: var(--card);
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 20px;
}

.plan-card-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 16px;
  padding-bottom: 12px;
  border-bottom: 1px solid var(--border);
}

.plan-name {
  font-size: 16px;
  font-weight: 700;
  margin: 0;
}
.plan-name.lite { color: var(--muted); }
.plan-name.pro { color: var(--accent); }
.plan-name.ultimate { color: var(--warning); }

.plan-id code { font-size: 10px; color: var(--muted); }

.form-group {
  margin-bottom: 10px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 8px;
}

.form-group label {
  font-size: 12px;
  color: var(--muted);
  flex: 1;
}

.form-group input {
  width: 100px;
  padding: 5px 8px;
  text-align: right;
  border: 1px solid var(--border);
  border-radius: 5px;
  background: var(--bg);
  color: var(--text);
  font-size: 13px;
}
.form-group input:focus { outline: none; border-color: var(--accent); }

.btn-save {
  width: 100%;
  margin-top: 14px;
  padding: 9px;
  background: var(--accent);
  color: white;
  border: none;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 600;
}
.btn-save:hover:not(:disabled) { background: #2563eb; }

.save-msg { margin-top: 8px; font-size: 13px; text-align: center; }
.save-msg.success { color: var(--success); }
.save-msg.error { color: var(--danger); }
</style>
