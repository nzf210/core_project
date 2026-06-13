<template>
  <div class="data-transfer-page">
    <div class="header-actions" style="margin-bottom: 1.5rem;">
      <h2>📥 Impor / Ekspor Data</h2>
      <p>Backup, laporkan ke akuntan, atau migrasi data dari spreadsheet (Excel / Google Sheet).</p>
    </div>

    <!-- Tabs -->
    <div class="tab-bar" style="display: flex; gap: 0.5rem; margin-bottom: 1.5rem;">
      <button v-for="t in tabs" :key="t.key" class="tab-btn" :class="{ active: activeTab === t.key }" @click="activeTab = t.key">
        {{ t.icon }} {{ t.label }}
      </button>
    </div>

    <div class="glass-card" style="padding: 1.5rem;">
      <!-- Common: range filter for journal -->
      <div v-if="activeTab === 'journal'" style="display: flex; gap: 1rem; margin-bottom: 1rem; align-items: end;">
        <label style="flex: 1;">
          <span style="display: block; font-size: 0.85rem; margin-bottom: 0.25rem;">Dari</span>
          <input v-model="dateRange.from" type="date" class="form-control" />
        </label>
        <label style="flex: 1;">
          <span style="display: block; font-size: 0.85rem; margin-bottom: 0.25rem;">Sampai</span>
          <input v-model="dateRange.to" type="date" class="form-control" />
        </label>
      </div>

      <!-- Actions row -->
      <div class="actions-row" style="display: flex; gap: 0.75rem; flex-wrap: wrap; margin-bottom: 1rem;">
        <a :href="api.templateURL(activeTabTyped, 'csv')" class="btn btn-secondary" target="_blank">
          📋 Download Template CSV
        </a>
        <a :href="api.templateURL(activeTabTyped, 'xlsx')" class="btn btn-secondary" target="_blank">
          📋 Download Template XLSX
        </a>
        <button class="btn btn-primary" @click="exportData('xlsx')" :disabled="exporting">
          {{ exporting ? 'Mengekspor...' : '📤 Export XLSX' }}
        </button>
        <button class="btn btn-primary" @click="exportData('csv')" :disabled="exporting">
          📤 Export CSV
        </button>
        <label class="btn btn-secondary" style="cursor: pointer;">
          {{ importing ? 'Mengimpor...' : '📥 Import File' }}
          <input type="file" accept=".csv,.xlsx" @change="importData" :disabled="importing" style="display: none;" />
        </label>
      </div>

      <!-- Result panel -->
      <div v-if="result" class="result-panel" style="margin-top: 1rem; padding: 1rem; background: var(--bg-tertiary); border-radius: 0.5rem;">
        <h4 style="margin: 0 0 0.5rem;">Hasil Import</h4>
        <p style="margin: 0.25rem 0;">
          ✅ Berhasil: <b>{{ result.imported }}</b> &nbsp;|&nbsp;
          ⚠️ Dilewati: <b>{{ result.skipped }}</b>
        </p>
        <div v-if="result.errors && result.errors.length" style="margin-top: 0.75rem;">
          <p style="margin: 0 0 0.25rem; font-weight: 600;">Detail error:</p>
          <ul style="margin: 0; padding-left: 1.5rem; max-height: 200px; overflow-y: auto; font-size: 0.85rem;">
            <li v-for="(e, i) in result.errors.slice(0, 50)" :key="i">
              Baris {{ e.row }}: {{ e.error }}
            </li>
            <li v-if="result.errors.length > 50" style="color: var(--text-secondary);">
              ... dan {{ result.errors.length - 50 }} error lainnya
            </li>
          </ul>
        </div>
      </div>

      <!-- Info panel -->
      <div class="info-panel" style="margin-top: 1.5rem; padding: 1rem; background: rgba(79, 70, 229, 0.05); border-left: 3px solid #4f46e5; border-radius: 0.4rem; font-size: 0.85rem;">
        <strong>ℹ️ Tips:</strong>
        <ul style="margin: 0.5rem 0 0; padding-left: 1.5rem;">
          <li>Download template dulu untuk lihat format kolom yang diharapkan</li>
          <li>Import bersifat <b>upsert</b>: produk dikunci via SKU, kontak via phone, jurnal via reference</li>
          <li>Maks 5000 baris & 10 MB per file. Ekstensi: .csv atau .xlsx</li>
          <li v-if="activeTab === 'journal'">Untuk jurnal multi-baris, gunakan <code>reference</code> yang sama di beberapa baris (satu entry)</li>
        </ul>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed } from 'vue'
import { api } from '../api'

const tabs: { key: 'products' | 'contacts' | 'journal'; label: string; icon: string }[] = [
  { key: 'products', label: 'Produk', icon: '📦' },
  { key: 'contacts', label: 'Kontak', icon: '👥' },
  { key: 'journal', label: 'Jurnal', icon: '📒' },
]
const activeTab = ref<'products' | 'contacts' | 'journal'>('products')
const activeTabTyped = computed(() => activeTab.value)

const today = new Date()
const firstOfMonth = new Date(today.getFullYear(), today.getMonth(), 1)
const dateRange = reactive({
  from: firstOfMonth.toISOString().split('T')[0],
  to: today.toISOString().split('T')[0],
})

const exporting = ref(false)
const importing = ref(false)
const result = ref<any>(null)

const endpoints: Record<string, string> = {
  products: '/api/umkm/export/products',
  contacts: '/api/umkm/export/contacts',
  journal: '/api/umkm/export/journal',
}

const importEndpoints: Record<string, string> = {
  products: '/api/umkm/import/products',
  contacts: '/api/umkm/import/contacts',
  journal: '/api/umkm/import/journal',
}

async function exportData(format: 'xlsx' | 'csv') {
  exporting.value = true
  try {
    const extra: Record<string, string> = {}
    if (activeTab.value === 'journal') {
      extra.from = dateRange.from
      extra.to = dateRange.to
    }
    const blobUrl = await api.exportFile(endpoints[activeTab.value], format, extra)
    const a = document.createElement('a')
    a.href = blobUrl
    a.download = `${activeTab.value}_${new Date().toISOString().split('T')[0]}.${format}`
    document.body.appendChild(a)
    a.click()
    a.remove()
    URL.revokeObjectURL(blobUrl)
  } catch (e: any) {
    alert('Gagal export: ' + (e?.message || e))
  } finally {
    exporting.value = false
  }
}

async function importData(e: Event) {
  const input = e.target as HTMLInputElement
  if (!input.files || !input.files[0]) return
  const file = input.files[0]
  importing.value = true
  result.value = null
  try {
    const res = await api.importFile(importEndpoints[activeTab.value], file)
    if (res.success) {
      result.value = res.data
    } else {
      alert('Gagal import: ' + (res.message || 'Unknown error'))
    }
  } catch (e: any) {
    alert('Error: ' + (e?.message || e))
  } finally {
    importing.value = false
    input.value = '' // reset to allow re-import of same file
  }
}
</script>

<style scoped>
.tab-btn {
  padding: 0.6rem 1.2rem;
  border-radius: 0.5rem;
  background: var(--bg-tertiary);
  color: var(--text-secondary);
  border: 1px solid var(--border-color);
  cursor: pointer;
  font-size: 0.9rem;
  transition: all 0.15s;
}
.tab-btn:hover {
  background: rgba(79, 70, 229, 0.05);
}
.tab-btn.active {
  background: #4f46e5;
  color: white;
  border-color: #4f46e5;
}
</style>
