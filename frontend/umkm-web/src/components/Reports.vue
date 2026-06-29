<template>
  <div class="reports-page animate-fade-in">
    <div class="header-actions" style="margin-bottom: 1.5rem;">
      <h2>📊 Laporan Keuangan</h2>
      <p>Laporan Laba Rugi, Neraca, dan Arus Kas (SAK-EMKM compliant).</p>
    </div>

    <!-- Tabs -->
    <div class="tab-bar" style="display: flex; gap: 0.5rem; margin-bottom: 1.5rem; flex-wrap: wrap;">
      <button v-for="t in tabs" :key="t.key" class="tab-btn" :class="{ active: activeTab === t.key }" @click="activeTab = t.key">
        {{ t.icon }} {{ t.label }}
      </button>
    </div>

    <!-- Date range (kecuali Neraca) -->
    <div v-if="activeTab !== 'balance-sheet'" class="glass-card" style="padding: 1rem; margin-bottom: 1rem; display: flex; gap: 1rem; align-items: end;">
      <label style="flex: 1;">
        <span style="display: block; font-size: 0.85rem; margin-bottom: 0.25rem;">Dari</span>
        <input v-model="dateRange.from" type="date" class="form-control" />
      </label>
      <label style="flex: 1;">
        <span style="display: block; font-size: 0.85rem; margin-bottom: 0.25rem;">Sampai</span>
        <input v-model="dateRange.to" type="date" class="form-control" />
      </label>
      <button class="btn btn-primary" @click="loadReport" :disabled="loading">
        {{ loading ? 'Memuat...' : '🔄 Refresh' }}
      </button>
    </div>
    <div v-else class="glass-card" style="padding: 1rem; margin-bottom: 1rem; display: flex; gap: 1rem; align-items: end;">
      <label style="flex: 1;">
        <span style="display: block; font-size: 0.85rem; margin-bottom: 0.25rem;">Per Tanggal</span>
        <input v-model="balanceDate" type="date" class="form-control" />
      </label>
      <button class="btn btn-primary" @click="loadReport" :disabled="loading">
        {{ loading ? 'Memuat...' : '🔄 Refresh' }}
      </button>
    </div>

    <!-- Error / empty -->
    <div v-if="error" class="glass-card" style="padding: 1rem; color: #dc2626;">{{ error }}</div>
    <div v-else-if="!reportData" class="glass-card" style="padding: 2rem; text-align: center; color: var(--text-secondary);">
      Pilih rentang tanggal dan klik Refresh.
    </div>

    <!-- INCOME STATEMENT (Laba Rugi) -->
    <div v-else-if="activeTab === 'income-statement'" class="glass-card" style="padding: 1.5rem;">
      <div class="flex items-center justify-between" style="margin-bottom: 1rem;">
        <h3 style="margin: 0;">Laba Rugi — {{ formatDateRange() }}</h3>
        <button class="btn btn-secondary" @click="downloadIncomePDF" :disabled="downloadingPDF">
          {{ downloadingPDF ? 'Membuat PDF...' : '📄 Download PDF' }}
        </button>
      </div>
      <div v-for="section in incomeStatementSections" :key="section.title" style="margin-bottom: 1.25rem;">
        <h4 style="margin-bottom: 0.5rem; color: var(--text-secondary); font-size: 0.95rem;">{{ section.title }}</h4>
        <table class="data-table">
          <tbody>
            <tr v-for="row in section.rows" :key="row.name">
              <td style="padding-left: 1.5rem;">{{ row.name }}</td>
              <td style="text-align: right;">{{ formatRupiah(row.balance) }}</td>
            </tr>
            <tr v-if="section.rows.length === 0">
              <td colspan="2" style="text-align: center; color: var(--text-secondary);">(tidak ada)</td>
            </tr>
            <tr style="font-weight: 600; border-top: 2px solid var(--border-color);">
              <td>Total {{ section.title }}</td>
              <td style="text-align: right;">{{ formatRupiah(section.total) }}</td>
            </tr>
          </tbody>
        </table>
      </div>
      <div style="padding: 1rem; background: var(--bg-tertiary); border-radius: 0.5rem; text-align: center;">
        <p style="margin: 0 0 0.25rem; font-size: 0.9rem; color: var(--text-secondary);">LABA / (RUGI) BERSIH</p>
        <p style="margin: 0; font-size: 1.5rem; font-weight: 700;" :class="{ 'text-red': netIncome < 0 }">
          {{ formatRupiah(netIncome) }}
        </p>
      </div>
    </div>

    <!-- BALANCE SHEET (Neraca) -->
    <div v-else-if="activeTab === 'balance-sheet'" class="glass-card" style="padding: 1.5rem;">
      <div class="flex items-center justify-between" style="margin-bottom: 1rem;">
        <h3 style="margin: 0;">Neraca — per {{ formatDate(balanceDate) }}</h3>
        <button class="btn btn-secondary" @click="downloadBalancePDF" :disabled="downloadingPDF">
          {{ downloadingPDF ? 'Membuat PDF...' : '📄 Download PDF' }}
        </button>
      </div>
      <div v-for="section in balanceSheetSections" :key="section.title" style="margin-bottom: 1.25rem;">
        <h4 style="margin-bottom: 0.5rem; color: var(--text-secondary); font-size: 0.95rem;">{{ section.title }}</h4>
        <table class="data-table">
          <tbody>
            <tr v-for="row in section.rows" :key="row.name">
              <td style="padding-left: 1.5rem;">{{ row.name }}</td>
              <td style="text-align: right;">{{ formatRupiah(row.balance) }}</td>
            </tr>
            <tr v-if="section.rows.length === 0">
              <td colspan="2" style="text-align: center; color: var(--text-secondary);">(tidak ada)</td>
            </tr>
            <tr style="font-weight: 600; border-top: 2px solid var(--border-color);">
              <td>Total {{ section.title }}</td>
              <td style="text-align: right;">{{ formatRupiah(section.total) }}</td>
            </tr>
          </tbody>
        </table>
      </div>
      <div style="padding: 1rem; background: var(--bg-tertiary); border-radius: 0.5rem; text-align: center; font-size: 0.95rem;">
        Total Aset: <b>{{ formatRupiah(totalAssets) }}</b> &nbsp;|&nbsp;
        Total Liabilitas + Ekuitas: <b>{{ formatRupiah(totalLiabEquity) }}</b>
        <span v-if="Math.abs(totalAssets - totalLiabEquity) > 1" style="display: block; margin-top: 0.5rem; color: #f59e0b; font-size: 0.85rem;">
          ⚠️ Neraca tidak balance (selisih {{ formatRupiah(Math.abs(totalAssets - totalLiabEquity)) }})
        </span>
      </div>
    </div>

    <!-- CASH FLOW (Arus Kas) -->
    <div v-else-if="activeTab === 'cash-flow'" class="glass-card" style="padding: 1.5rem;">
      <div class="flex items-center justify-between" style="margin-bottom: 1rem;">
        <h3 style="margin: 0;">Arus Kas — {{ formatDateRange() }}</h3>
        <button class="btn btn-secondary" @click="downloadCashFlowPDF" :disabled="downloadingPDF">
          {{ downloadingPDF ? 'Membuat PDF...' : '📄 Download PDF' }}
        </button>
      </div>
      <div v-for="section in cashFlowSections" :key="section.title" style="margin-bottom: 1.25rem;">
        <h4 style="margin-bottom: 0.5rem; color: var(--text-secondary); font-size: 0.95rem;">{{ section.title }}</h4>
        <table class="data-table">
          <tbody>
            <tr v-for="row in section.lines" :key="row.id">
              <td style="padding-left: 1.5rem;">{{ formatDate(row.date) }} — {{ row.description }}</td>
              <td style="text-align: right;">{{ formatRupiah(row.inflow - row.outflow) }}</td>
            </tr>
            <tr v-if="section.lines.length === 0">
              <td colspan="2" style="text-align: center; color: var(--text-secondary);">(tidak ada)</td>
            </tr>
            <tr style="font-weight: 600; border-top: 2px solid var(--border-color);">
              <td>Arus Kas {{ section.title }}</td>
              <td style="text-align: right;" :class="{ 'text-red': section.net < 0 }">{{ formatRupiah(section.net) }}</td>
            </tr>
          </tbody>
        </table>
      </div>
      <div style="padding: 1rem; background: var(--bg-tertiary); border-radius: 0.5rem;">
        <div class="flex items-center justify-between" style="margin-bottom: 0.5rem;">
          <span>Kas Awal Periode</span>
          <b>{{ formatRupiah(reportData.opening_cash) }}</b>
        </div>
        <div class="flex items-center justify-between" style="margin-bottom: 0.5rem;">
          <span>Kenaikan/(Penurunan) Bersih</span>
          <b :class="{ 'text-red': reportData.net_cash_flow < 0 }">{{ formatRupiah(reportData.net_cash_flow) }}</b>
        </div>
        <div class="flex items-center justify-between" style="padding-top: 0.5rem; border-top: 1px solid var(--border-color);">
          <span style="font-weight: 600;">Kas Akhir Periode</span>
          <b style="font-size: 1.1rem;">{{ formatRupiah(reportData.closing_cash) }}</b>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { api } from '../api'
import { formatRupiah } from '../composables/useCurrency'

const tabs = [
  { key: 'income-statement', label: 'Laba Rugi', icon: '📈' },
  { key: 'balance-sheet', label: 'Neraca', icon: '⚖️' },
  { key: 'cash-flow', label: 'Arus Kas', icon: '💵' },
]
const activeTab = ref('income-statement')

const today = new Date()
const firstOfMonth = new Date(today.getFullYear(), today.getMonth(), 1)
const dateRange = reactive({
  from: firstOfMonth.toISOString().split('T')[0],
  to: today.toISOString().split('T')[0],
})
const balanceDate = ref(today.toISOString().split('T')[0])

const loading = ref(false)
const downloadingPDF = ref(false)
const error = ref('')
const reportData = ref<any>(null)

const endpointMap: Record<string, string> = {
  'income-statement': '/api/umkm/reports/income-statement',
  'balance-sheet': '/api/umkm/reports/balance-sheet',
  'cash-flow': '/api/umkm/reports/cash-flow',
}

async function loadReport() {
  loading.value = true
  error.value = ''
  reportData.value = null
  try {
    const ep = endpointMap[activeTab.value]
    const params = new URLSearchParams()
    if (activeTab.value === 'balance-sheet') {
      params.set('date', balanceDate.value)
    } else {
      params.set('from', dateRange.from)
      params.set('to', dateRange.to)
    }
    const res = await api.get(`${ep}?${params}`)
    if (res.success) {
      reportData.value = res.data
    } else {
      error.value = res.message || 'Gagal memuat laporan'
    }
  } catch (e: any) {
    error.value = 'Error: ' + (e?.message || e)
  } finally {
    loading.value = false
  }
}

async function downloadCashFlowPDF() {
  await downloadPDF(
    api.cashFlowPDFUrl(dateRange.from, dateRange.to),
    `arus-kas_${dateRange.from}_${dateRange.to}.pdf`,
  )
}

async function downloadIncomePDF() {
  await downloadPDF(
    api.incomeStatementPDFUrl(dateRange.from, dateRange.to),
    `laba-rugi_${dateRange.from}_${dateRange.to}.pdf`,
  )
}

async function downloadBalancePDF() {
  await downloadPDF(
    api.balanceSheetPDFUrl(balanceDate.value),
    `neraca_${balanceDate.value}.pdf`,
  )
}

async function downloadPDF(url: string, filename: string) {
  downloadingPDF.value = true
  try {
    const res = await fetch(url, {
      headers: {
        'X-Tenant-ID': localStorage.getItem('tenant_id') || '',
        'Authorization': `Bearer ${localStorage.getItem('access_token') || ''}`,
      },
    })
    if (!res.ok) throw new Error('HTTP ' + res.status)
    const blob = await res.blob()
    const blobUrl = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = blobUrl
    a.download = filename
    a.click()
    a.remove()
    URL.revokeObjectURL(blobUrl)
  } catch (e: any) {
    alert('Gagal download PDF: ' + (e?.message || e))
  } finally {
    downloadingPDF.value = false
  }
}

// === Income Statement derived ===
const incomeStatementSections = computed(() => {
  if (activeTab.value !== 'income-statement' || !reportData.value) return []
  const data: any[] = Array.isArray(reportData.value) ? reportData.value : []
  const revenue = data.filter((r: any) => r.type === 'revenue').map((r: any) => ({ name: r.name, balance: Number(r.balance) }))
  const expenses = data.filter((r: any) => r.type === 'expense').map((r: any) => ({ name: r.name, balance: Number(r.balance) }))
  return [
    { title: 'Pendapatan', rows: revenue, total: revenue.reduce((s, r) => s + r.balance, 0) },
    { title: 'Beban', rows: expenses, total: expenses.reduce((s, r) => s + r.balance, 0) },
  ]
})
const netIncome = computed(() => {
  const sections = incomeStatementSections.value
  return sections.length === 2 ? sections[0].total - sections[1].total : 0
})

// === Balance Sheet derived ===
const balanceSheetSections = computed(() => {
  if (activeTab.value !== 'balance-sheet' || !reportData.value) return []
  const data: any[] = Array.isArray(reportData.value) ? reportData.value : []
  const assets = data.filter((r: any) => r.type === 'asset').map((r: any) => ({ name: r.name, balance: Number(r.balance) }))
  const liab = data.filter((r: any) => r.type === 'liability').map((r: any) => ({ name: r.name, balance: Number(r.balance) }))
  const equity = data.filter((r: any) => r.type === 'equity').map((r: any) => ({ name: r.name, balance: Number(r.balance) }))
  return [
    { title: 'Aset', rows: assets, total: assets.reduce((s, r) => s + r.balance, 0) },
    { title: 'Liabilitas', rows: liab, total: liab.reduce((s, r) => s + r.balance, 0) },
    { title: 'Ekuitas', rows: equity, total: equity.reduce((s, r) => s + r.balance, 0) },
  ]
})
const totalAssets = computed(() => balanceSheetSections.value[0]?.total || 0)
const totalLiabEquity = computed(() => {
  const liab = balanceSheetSections.value[1]?.total || 0
  const equity = balanceSheetSections.value[2]?.total || 0
  return liab + equity
})

// === Cash Flow derived ===
const cashFlowSections = computed(() => {
  if (activeTab.value !== 'cash-flow' || !reportData.value?.activities) return []
  const a = reportData.value.activities
  return [
    { title: 'Operasional', lines: a.operating?.lines || [], net: a.operating?.net || 0 },
    { title: 'Investasi', lines: a.investing?.lines || [], net: a.investing?.net || 0 },
    { title: 'Pendanaan', lines: a.financing?.lines || [], net: a.financing?.net || 0 },
  ]
})

function formatDate(d: string): string {
  if (!d) return ''
  const parts = d.split('-')
  if (parts.length !== 3) return d
  return `${parts[2]}/${parts[1]}/${parts[0]}`
}

function formatDateRange(): string {
  return `${formatDate(dateRange.from)} – ${formatDate(dateRange.to)}`
}

onMounted(() => {
  loadReport()
})
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
.data-table {
  width: 100%;
  border-collapse: collapse;
}
.data-table td {
  padding: 0.5rem 0.25rem;
  font-size: 0.9rem;
  border-bottom: 1px dashed var(--border-color);
}
.text-red {
  color: #dc2626;
}
</style>
