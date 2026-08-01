<template>
  <div class="report-generator">
    <h3 style="margin-bottom: 1rem; color: var(--text-secondary)">Reporting & Analytics</h3>
    <div class="actions" style="margin-bottom: 2rem;">
      <button type="button" class="btn-primary" @click="downloadExcel">⬇️ Export to Excel</button>
      <button type="button" class="btn-secondary" @click="downloadPDF">⬇️ Export to PDF</button>
    </div>

    <div v-if="reportData" class="report-preview">
      <h4 style="margin-bottom: 1rem;">Laporan Dukungan Wilayah</h4>
      <div class="summary-cards">
        <div class="card">Total Pemilih: {{ reportData.summary.total_voters }}</div>
        <div class="card">Total Relawan: {{ reportData.summary.total_volunteers }}</div>
      </div>
      <table class="data-table" style="margin-top: 1rem;">
        <thead>
          <tr><th>Wilayah</th><th>Dukungan Terkumpul</th></tr>
        </thead>
        <tbody>
          <tr v-for="(r, idx) in reportData.regions" :key="idx">
            <td>{{ r.region }}</td><td>{{ r.voters }}</td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script setup lang="ts">
import { apiClient } from '../api'

import { ref, onMounted } from 'vue'

const reportData = ref<any>(null)

const fetchReport = async () => {
  try {
    const res = await apiClient('/reports', { headers: { 'X-Tenant-ID': 'default' } })
    const data = await res.json()
    if (data.success) reportData.value = data.data
  } catch (err) { console.error(err) }
}

const downloadExcel = () => { alert('Downloading Excel... (Mocked: Use xlsx library in frontend)') }
const downloadPDF = () => { alert('Downloading PDF... (Mocked: Use jspdf library in frontend)') }

onMounted(fetchReport)
</script>

<style scoped>
.btn-primary {
  background: var(--accent-gradient); color: white; border: none;
  padding: 0.5rem 1.5rem; border-radius: var(--radius-sm);
  cursor: pointer; font-weight: 600; margin-right: 1rem;
}
.btn-secondary {
  background: var(--bg-tertiary); color: var(--text-primary); border: 1px solid var(--border-color);
  padding: 0.5rem 1.5rem; border-radius: var(--radius-sm);
  cursor: pointer; font-weight: 600;
}
.summary-cards { display: flex; gap: 1rem; }
.card { background: var(--bg-secondary); padding: 1rem; border-radius: var(--radius-sm); flex: 1; border: 1px solid var(--border-color); }
.data-table { width: 100%; border-collapse: collapse; }
.data-table th, .data-table td { padding: 1rem; text-align: left; border-bottom: 1px solid var(--border-color); }
.data-table th { font-weight: 600; background: var(--bg-tertiary); }
</style>
