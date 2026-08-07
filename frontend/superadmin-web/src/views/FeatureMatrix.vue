<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { request } from '../api/client'

interface Plan {
  id: string
  name: string
}

interface FeatureEntry {
  feature_key: string
  is_enabled: boolean
  feature_value: number | null
}

interface AddonGating {
  feature_key: string
  feature_name: string
  min_tier: string
  default_enabled: boolean
}

const plans = ref<Plan[]>([])
const features = ref<string[]>([])
const matrix = ref<Record<string, Record<string, boolean>>>({}) // [planId][featureKey] = is_enabled
const addonsGating = ref<AddonGating[]>([])
const loading = ref(true)
const savingCell = ref<Record<string, boolean>>({})
const savingGating = ref<Record<string, boolean>>({})
const error = ref('')
const successMsg = ref('')

const TIERS = ['lite', 'pro', 'ultimate']

async function load() {
  loading.value = true
  error.value = ''
  try {
    const [plansRes, matrixRes, gatingRes] = await Promise.all([
      request('/api/superadmin/billing/plans'),
      request('/api/superadmin/billing/plan-features'),
      request('/api/superadmin/billing/addon-gating'),
    ])

    plans.value = plansRes.data || []
    addonsGating.value = gatingRes.data || []

    // Build feature list from matrix data
    const matrixData: FeatureEntry[] = matrixRes.data || []
    const featureSet = new Set<string>()
    const planMatrixMap: Record<string, Record<string, boolean>> = {}

    plans.value.forEach(p => { planMatrixMap[p.id] = {} })

    matrixData.forEach((entry: any) => {
      featureSet.add(entry.feature_key)
      if (entry.plan_id && planMatrixMap[entry.plan_id] !== undefined) {
        planMatrixMap[entry.plan_id][entry.feature_key] = !!entry.is_enabled
      }
    })

    features.value = Array.from(featureSet).sort()
    matrix.value = planMatrixMap
  } catch (e: any) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}

async function toggleFeature(planId: string, featureKey: string) {
  const cellKey = `${planId}:${featureKey}`
  savingCell.value[cellKey] = true
  const current = matrix.value[planId]?.[featureKey] ?? false
  const newVal = !current

  // Optimistic update
  if (!matrix.value[planId]) matrix.value[planId] = {}
  matrix.value[planId][featureKey] = newVal

  try {
    await request('/api/superadmin/billing/feature-matrix', {
      method: 'PATCH',
      body: JSON.stringify({ plan_id: planId, feature_key: featureKey, is_enabled: newVal }),
    })
    showSuccess(`${featureKey} untuk ${planId} ${newVal ? 'diaktifkan' : 'dinonaktifkan'}`)
  } catch (e: any) {
    // Rollback on error
    matrix.value[planId][featureKey] = current
    error.value = 'Gagal update: ' + e.message
  } finally {
    savingCell.value[cellKey] = false
  }
}

async function saveGating(addon: AddonGating) {
  savingGating.value[addon.feature_key] = true
  try {
    await request('/api/superadmin/billing/addon-gating', {
      method: 'PATCH',
      body: JSON.stringify({ feature_key: addon.feature_key, min_tier: addon.min_tier }),
    })
    showSuccess(`Min tier ${addon.feature_key} disimpan: ${addon.min_tier || 'semua'}`)
  } catch (e: any) {
    error.value = 'Gagal simpan gating: ' + e.message
  } finally {
    savingGating.value[addon.feature_key] = false
  }
}

function showSuccess(msg: string) {
  successMsg.value = msg
  setTimeout(() => successMsg.value = '', 3000)
}

onMounted(load)
</script>

<template>
  <div>
    <div class="page-header">
      <div>
        <h1>Feature Matrix & Addon Gating</h1>
        <p class="subtitle">Toggle fitur per tier plan, dan atur tier minimum untuk setiap addon.</p>
      </div>
      <button class="btn-refresh" @click="load">🔄 Refresh</button>
    </div>

    <div v-if="successMsg" class="success-banner">✅ {{ successMsg }}</div>
    <div v-if="error" class="error-banner">⚠️ {{ error }}</div>

    <div v-if="loading" class="loading">Memuat feature matrix...</div>

    <template v-else>
      <!-- Feature Matrix Section -->
      <section class="section">
        <h2 class="section-title">📋 Feature Matrix</h2>
        <p class="section-desc">Klik checkbox untuk toggle fitur per plan. Perubahan langsung disimpan.</p>

        <div class="table-wrap" v-if="features.length > 0">
          <table class="matrix-table">
            <thead>
              <tr>
                <th class="feature-col">Feature Key</th>
                <th v-for="plan in plans" :key="plan.id" class="plan-col">
                  <span :class="['plan-badge', plan.name.toLowerCase()]">{{ plan.name }}</span>
                </th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="feat in features" :key="feat">
                <td class="feature-key">
                  <code>{{ feat }}</code>
                </td>
                <td v-for="plan in plans" :key="plan.id" class="checkbox-cell">
                  <label :class="['toggle', { saving: savingCell[`${plan.id}:${feat}`] }]">
                    <input
                      type="checkbox"
                      :checked="matrix[plan.id]?.[feat] ?? false"
                      @change="toggleFeature(plan.id, feat)"
                      :disabled="!!savingCell[`${plan.id}:${feat}`]"
                      class="checkbox"
                      :aria-label="`${feat} — ${plan.name}`"
                    />
                    <span class="checkmark"></span>
                  </label>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <div v-else class="empty">Belum ada feature key terdaftar di feature matrix.</div>
      </section>

      <!-- Addon Gating Section -->
      <section class="section">
        <h2 class="section-title">🔒 Addon Tier Gating</h2>
        <p class="section-desc">Atur tier minimum yang diperlukan sebelum tenant bisa membeli addon ini.</p>

        <div v-if="addonsGating.length > 0" class="gating-grid">
          <div v-for="addon in addonsGating" :key="addon.feature_key" class="gating-card">
            <div class="gating-info">
              <div class="gating-name">{{ addon.feature_name || addon.feature_key }}</div>
              <code class="gating-key">{{ addon.feature_key }}</code>
            </div>
            <div class="gating-controls">
              <label class="gating-label" :for="`min-tier-${addon.feature_key}`">Min Tier</label>
              <select v-model="addon.min_tier" class="tier-select" :id="`min-tier-${addon.feature_key}`">
                <option value="">Semua tier (tidak ada min)</option>
                <option v-for="t in TIERS" :key="t" :value="t">{{ t.charAt(0).toUpperCase() + t.slice(1) }}</option>
              </select>
              <button
                class="btn-save-gating"
                @click="saveGating(addon)"
                :disabled="savingGating[addon.feature_key]"
              >
                {{ savingGating[addon.feature_key] ? '...' : 'Simpan' }}
              </button>
            </div>
          </div>
        </div>
        <div v-else class="empty">Tidak ada addon yang terdaftar untuk gating.</div>
      </section>
    </template>
  </div>
</template>

<style scoped>
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 24px;
}
h1 { font-size: 24px; margin-bottom: 4px; }
.subtitle { color: var(--muted); font-size: 14px; }

.btn-refresh {
  background: var(--bg);
  border: 1px solid var(--border);
  color: var(--text);
  padding: 8px 14px;
  border-radius: 6px;
  font-size: 13px;
}
.btn-refresh:hover { background: var(--border); }

.success-banner {
  background: rgba(16, 185, 129, 0.1);
  border: 1px solid rgba(16, 185, 129, 0.3);
  color: var(--success);
  border-radius: 8px;
  padding: 10px 16px;
  margin-bottom: 16px;
  font-size: 14px;
}
.error-banner {
  background: rgba(239, 68, 68, 0.1);
  border: 1px solid rgba(239, 68, 68, 0.3);
  color: var(--danger);
  border-radius: 8px;
  padding: 10px 16px;
  margin-bottom: 16px;
  font-size: 14px;
}
.loading, .empty {
  padding: 40px;
  text-align: center;
  color: var(--muted);
  background: var(--card);
  border-radius: 12px;
  border: 1px dashed var(--border);
}

.section { margin-bottom: 40px; }
.section-title { font-size: 16px; font-weight: 700; margin-bottom: 6px; }
.section-desc { font-size: 13px; color: var(--muted); margin-bottom: 16px; }

.table-wrap { overflow-x: auto; border-radius: 10px; border: 1px solid var(--border); }

.matrix-table {
  width: 100%;
  background: var(--card);
  border-collapse: collapse;
}

.feature-col { width: 260px; }
.plan-col { width: 120px; text-align: center; }

.plan-badge {
  font-size: 11px;
  font-weight: 700;
  padding: 3px 10px;
  border-radius: 20px;
  text-transform: uppercase;
}
.plan-badge.lite { background: rgba(148, 163, 184, 0.15); color: var(--muted); }
.plan-badge.pro { background: rgba(59, 130, 246, 0.15); color: var(--accent); }
.plan-badge.ultimate { background: rgba(245, 158, 11, 0.15); color: var(--warning); }

.feature-key { padding: 10px 16px; }
.feature-key code { font-size: 12px; color: #fbbf24; }

.checkbox-cell { text-align: center; padding: 10px; }

.toggle {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  position: relative;
}
.toggle.saving { opacity: 0.5; }

.checkbox {
  width: 18px;
  height: 18px;
  accent-color: var(--accent);
  cursor: pointer;
}

/* Gating grid */
.gating-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(360px, 1fr));
  gap: 12px;
}

.gating-card {
  background: var(--card);
  border: 1px solid var(--border);
  border-radius: 10px;
  padding: 16px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
}

.gating-info { flex: 1; min-width: 0; }
.gating-name { font-size: 14px; font-weight: 600; margin-bottom: 2px; }
.gating-key { font-size: 11px; color: var(--muted); }

.gating-controls {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}
.gating-label { font-size: 12px; color: var(--muted); white-space: nowrap; }

.tier-select {
  font-size: 13px;
  padding: 6px 10px;
  border-radius: 6px;
  min-width: 160px;
}

.btn-save-gating {
  background: var(--accent);
  color: white;
  border: none;
  padding: 6px 14px;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 600;
  white-space: nowrap;
}
.btn-save-gating:hover:not(:disabled) { background: #2563eb; }
</style>
