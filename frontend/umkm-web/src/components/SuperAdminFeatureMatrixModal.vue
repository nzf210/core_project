<template>
  <div class="modal-overlay" @click.self="$emit('update:showFeatureMatrix', false)">
    <div class="modal-card" style="max-width: 900px; max-height: 90vh; overflow-y: auto;">
      <h3 style="margin: 0 0 0.25rem 0;">Feature Matrix</h3>
      <p style="color: var(--text-secondary); font-size: 0.85rem; margin-bottom: 1rem;">
        Toggle fitur per paket. Perubahan langsung aktif.
      </p>

      <div v-if="featureMatrixLoading" style="text-align:center; padding:2rem;">Memuat...</div>
      <div v-else>
        <!-- Feature Matrix Table -->
        <div style="overflow-x:auto;">
          <table style="width:100%; border-collapse:collapse; font-size:0.8rem;">
            <thead>
              <tr style="border-bottom: 1px solid var(--border-color);">
                <th style="text-align:left; padding:0.5rem 0.75rem; position:sticky; left:0; background:var(--bg); min-width:160px;">Fitur</th>
                <th v-for="pid in featureMatrixPlanIds" :key="pid" style="text-align:center; padding:0.5rem 0.5rem; min-width:80px;">
                  {{ featureMatrixPlans[pid]?.plan_name || pid }}
                </th>
              </tr>
            </thead>
            <tbody>
              <template v-for="key in featureMatrixOrder" :key="key">
                <tr style="border-bottom: 1px solid rgba(255,255,255,0.05);">
                  <td style="padding:0.5rem 0.75rem; position:sticky; left:0; background:var(--bg);">
                    <span :style="{ color: isAddonFeature(key) ? '#f59e0b' : '#60a5fa' }">{{ key }}</span>
                  </td>
                  <td v-for="pid in featureMatrixPlanIds" :key="pid" style="text-align:center; padding:0.5rem 0.25rem;">
                    <label>
                      <input
                        type="checkbox"
                        :checked="getFeatureEnabled(pid, key)"
                        @change="$emit('toggle', pid, key, $event)"
                        style="cursor:pointer; width:16px; height:16px;"
                        :aria-label="`${featureMatrixPlans[pid]?.plan_name}: ${key}`"
                      />
                    </label>
                  </td>
                </tr>
              </template>
            </tbody>
          </table>
        </div>

        <hr style="border-color:var(--border-color); margin:1.5rem 0;" />

        <!-- Addon Gating Section -->
        <h4 style="margin:0 0 0.75rem 0; color:var(--text-secondary);">Addon Gating — Minimum Tier untuk Membeli</h4>
        <div v-if="addonGatingLoading" style="text-align:center; padding:1rem;">Memuat...</div>
        <div v-else style="display:flex; flex-direction:column; gap:0.5rem;">
          <div v-for="addon in addonGatingList" :key="addon.feature_key" style="display:flex; align-items:center; gap:0.75rem; padding:0.5rem; background:rgba(255,255,255,0.03); border-radius:6px;">
            <span style="min-width:120px; font-size:0.8rem; color:#f59e0b;">{{ addon.feature_key }}</span>
            <span style="flex:1; font-size:0.8rem;">{{ addon.feature_name }}</span>
            <select
              :value="addon.min_tier || ''"
              @change="$emit('save-gating', addon.feature_key, ($event.target as HTMLSelectElement).value)"
              style="padding:0.25rem 0.5rem; border-radius:4px; background:var(--bg); border:1px solid var(--border-color); color:var(--text); font-size:0.8rem; min-width:100px;"
              :aria-label="`Minimum tier for ${addon.feature_key}`"
            >
              <option value="">Semua tier</option>
              <option value="lite">Lite+</option>
              <option value="pro">Pro+</option>
              <option value="ultimate">Ultimate only</option>
            </select>
          </div>
        </div>
      </div>

      <div style="display:flex; gap:0.75rem; justify-content:flex-end; margin-top:1.5rem;">
        <button class="btn btn-secondary" @click="$emit('update:showFeatureMatrix', false)">Tutup</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
const props = defineProps<{
  showFeatureMatrix: boolean
  featureMatrixLoading: boolean
  featureMatrixPlans: any
  featureMatrixPlanIds: any[]
  featureMatrixOrder: any[]
  featureMatrixData: any
  addonGatingLoading: boolean
  addonGatingList: any[]
}>()

defineEmits<{
  'update:showFeatureMatrix': [value: boolean]
  'toggle': [planId: string, featureKey: string, event: Event]
  'save-gating': [featureKey: string, minTier: string]
}>()

function isAddonFeature(key: string): boolean {
  return ['ai_vision', 'ai_audio', 'wa_blast', 'extra_store', 'extra_user'].includes(key)
}

function getFeatureEnabled(planId: string, featureKey: string): boolean {
  return props.featureMatrixData?.[planId]?.[featureKey]?.is_enabled ?? false
}
</script>
