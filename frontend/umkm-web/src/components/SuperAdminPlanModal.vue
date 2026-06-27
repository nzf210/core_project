<template>
  <div class="modal-overlay" @click.self="$emit('update:showPlanEditor', false)">
    <div class="modal-card" style="max-width: 520px;">
      <h3 style="margin: 0 0 0.25rem 0;">Kelola Paket Langganan</h3>
      <p style="color: var(--text-secondary); font-size: 0.85rem; margin-bottom: 1.5rem;">
        Ubah harga paket SaaS untuk semua tenant.
      </p>

      <div v-if="loadingPlans" style="text-align: center; padding: 2rem; color: var(--text-secondary);">
        Memuat...
      </div>

      <div v-else>
        <div v-for="plan in editablePlans" :key="plan.id" class="plan-editor-row">
          <div class="plan-editor-header">
            <div>
              <span class="badge" :class="['badge-' + plan.id]">{{ plan.name.toUpperCase() }}</span>
            </div>
            <div style="display: flex; gap: 0.5rem; align-items: center;">
              <span style="font-size: 0.75rem; color: var(--text-secondary);">Aktif</span>
              <label class="toggle-switch" :aria-label="`Aktifkan paket ${plan.name}`">
                <input type="checkbox" v-model="plan.is_active" />
                <span class="toggle-slider"></span>
              </label>
            </div>
          </div>
          <div class="plan-editor-fields">
            <div class="form-group">
              <label>Harga Bulanan (Rp)
                <div style="display: flex; align-items: center; gap: 0.25rem; margin-top: 0.25rem;">
                  <span style="font-size: 0.85rem; color: var(--text-secondary);">Rp</span>
                  <input v-model.number="plan.price_monthly_display" type="number" class="form-control" min="0"
                    step="1000" style="width: 100%;" @input="$emit('sync-price', plan, 'monthly')" />
                </div>
              </label>
              <small style="color: var(--text-secondary); font-size: 0.7rem;">
                Dalam sen: {{ plan.price_monthly?.toLocaleString() }} sen
              </small>
            </div>
            <div class="form-group">
              <label>Harga Tahunan (Rp)
                <div style="display: flex; align-items: center; gap: 0.25rem; margin-top: 0.25rem;">
                  <span style="font-size: 0.85rem; color: var(--text-secondary);">Rp</span>
                  <input v-model.number="plan.price_yearly_display" type="number" class="form-control" min="0"
                    step="1000" style="width: 100%;" @input="$emit('sync-price', plan, 'yearly')" />
                </div>
              </label>
              <small style="color: var(--text-secondary); font-size: 0.7rem;">
                Dalam sen: {{ plan.price_yearly?.toLocaleString() }} sen
              </small>
            </div>
          </div>
        </div>
      </div>

      <div style="display: flex; gap: 0.75rem; justify-content: flex-end; margin-top: 1.5rem;">
        <button class="btn btn-secondary" @click="$emit('update:showPlanEditor', false)" :disabled="savingPlans">Tutup</button>
        <button class="btn btn-primary" @click="$emit('save')" :disabled="savingPlans">
          {{ savingPlans ? 'Menyimpan...' : 'Simpan Semua' }}
        </button>
      </div>
      <div v-if="planError" style="margin-top: 0.75rem; color: #ef4444; font-size: 0.85rem;">{{ planError }}</div>
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  showPlanEditor: boolean
  loadingPlans: boolean
  savingPlans: boolean
  editablePlans: any[]
  planError: string
}>()

defineEmits<{
  'update:showPlanEditor': [value: boolean]
  'save': []
  'sync-price': [plan: any, kind: 'monthly' | 'yearly']
}>()
</script>

<style scoped>
.form-group {
  margin-bottom: 0;
}

.form-group label {
  display: block;
  font-size: 0.8rem;
  font-weight: 500;
  margin-bottom: 0.3rem;
  color: var(--text-primary);
}

.form-control {
  width: 100%;
  padding: 0.6rem 0.75rem;
  background: var(--bg-tertiary);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-sm);
  color: var(--text-primary);
  font-family: inherit;
  font-size: 0.875rem;
  box-sizing: border-box;
}

.form-control:focus {
  outline: none;
  border-color: var(--accent-primary);
}

.badge {
  padding: 0.2rem 0.6rem;
  border-radius: 20px;
  font-size: 0.7rem;
  font-weight: 700;
  text-transform: uppercase;
}

.badge-lite {
  background: rgba(245, 158, 11, 0.2);
  color: #fbbf24;
}

.badge-pro {
  background: rgba(59, 130, 246, 0.2);
  color: #60a5fa;
}

.badge-ultimate {
  background: rgba(168, 85, 247, 0.2);
  color: #c084fc;
}

/* Plan Editor */
.plan-editor-row {
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  padding: 1rem;
  margin-bottom: 1rem;
}

.plan-editor-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 0.75rem;
}

.plan-editor-fields {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0.75rem;
}

/* Toggle Switch */
.toggle-switch {
  position: relative;
  display: inline-block;
  width: 40px;
  height: 22px;
  cursor: pointer;
}

.toggle-switch input {
  opacity: 0;
  width: 0;
  height: 0;
}

.toggle-slider {
  position: absolute;
  inset: 0;
  background-color: rgba(100, 116, 139, 0.4);
  border-radius: 22px;
  transition: 0.2s;
}

.toggle-slider::before {
  content: '';
  position: absolute;
  height: 16px;
  width: 16px;
  left: 3px;
  bottom: 3px;
  background-color: white;
  border-radius: 50%;
  transition: 0.2s;
}

.toggle-switch input:checked + .toggle-slider {
  background-color: #10b981;
}

.toggle-switch input:checked + .toggle-slider::before {
  transform: translateX(18px);
}

@media (max-width: 480px) {
  .plan-editor-fields {
    grid-template-columns: 1fr;
  }
}
</style>
