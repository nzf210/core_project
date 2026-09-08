<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { api } from '../api/client'

const route = useRoute()
const programId = ref('')
const count = ref(50)
const validDays = ref(365)
const baseUrl = ref('https://app.wch.id')
const generating = ref(false)
const result = ref<any>(null)
const error = ref('')
const programs = ref<any[]>([])
const loadingPrograms = ref(false)
const copiedIndex = ref<number | null>(null)

onMounted(async () => {
  // Pre-fill from query parameter if available
  if (route.query.program_id) {
    programId.value = String(route.query.program_id)
  }

  // Load programs list for dropdown selection
  loadingPrograms.value = true
  try {
    const res = await api.listVoucherPrograms()
    programs.value = res.data || []
    if (!programId.value && programs.value.length > 0) {
      programId.value = programs.value[0].id
    }
  } catch (e: any) {
    console.error('Failed to load voucher programs:', e)
  } finally {
    loadingPrograms.value = false
  }
})

async function generate() {
  if (!programId.value || count.value < 1) {
    error.value = 'Program ID dan count wajib diisi'
    return
  }
  generating.value = true
  error.value = ''
  try {
    const res = await api.generateVoucherLinks({
      program_id: programId.value,
      count: Number(count.value),
      valid_days: Number(validDays.value),
      base_url: baseUrl.value,
    })
    result.value = res.data
  } catch (e: any) {
    error.value = e.message
  } finally {
    generating.value = false
  }
}

function downloadCSV() {
  if (!result.value?.links) return
  const rows = result.value.links.map((l: any) => `${l.url}`)
  const csv = 'redeem_url\n' + rows.join('\n')
  const blob = new Blob([csv], { type: 'text/csv' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `voucher-links-${Date.now()}.csv`
  a.click()
  URL.revokeObjectURL(url)
}

function copyToClipboard(text: string, idx?: number) {
  if (navigator.clipboard && navigator.clipboard.writeText) {
    navigator.clipboard.writeText(text).then(() => {
      if (idx !== undefined) {
        copiedIndex.value = idx
        setTimeout(() => {
          if (copiedIndex.value === idx) copiedIndex.value = null
        }, 2000)
      }
    })
  } else {
    const input = document.createElement('input')
    input.value = text
    document.body.appendChild(input)
    input.select()
    document.execCommand('copy')
    document.body.removeChild(input)
    if (idx !== undefined) {
      copiedIndex.value = idx
      setTimeout(() => {
        if (copiedIndex.value === idx) copiedIndex.value = null
      }, 2000)
    }
  }
}
</script>

<template>
  <div class="generate-vouchers-page">
    <h1>Generate Voucher Links</h1>
    <p class="subtitle">Bulk generate link voucher (1 link = 1 klaim). Distribusikan ke reseller/customer.</p>

    <form class="card form" @submit.prevent="generate">
      <!-- Program Selection -->
      <div class="row">
        <label v-if="programs.length > 0">Pilih Program Voucher
          <select v-model="programId" required>
            <option value="">-- Pilih Program Voucher --</option>
            <option v-for="p in programs" :key="p.id" :value="p.id">
              {{ p.name }} ({{ p.voucher_type }} — {{ p.duration_months }} bln{{ p.target_plan_id ? ' — ' + p.target_plan_id.toUpperCase() : '' }})
            </option>
          </select>
        </label>
        <label>Program ID (UUID)
          <input v-model="programId" placeholder="uuid dari voucher_programs" required />
        </label>
      </div>

      <div class="row">
        <label>Jumlah Link (max 1000)
          <input type="number" v-model="count" min="1" max="1000" required />
        </label>
        <label>Masa Aktif Link (hari)
          <input type="number" v-model="validDays" min="1" max="3650" />
        </label>
        <label>Base URL
          <input v-model="baseUrl" placeholder="https://app.wch.id" />
        </label>
      </div>

      <button type="submit" :disabled="generating">{{ generating ? 'Generating...' : 'Generate Links' }}</button>
      <div v-if="error" class="error">{{ error }}</div>
    </form>

    <div v-if="result" class="block">
      <div class="result-header">
        <h2>Generated: {{ result.count }} links</h2>
        <button class="btn-download" @click="downloadCSV">📥 Download CSV</button>
      </div>
      <div class="links-table">
        <table>
          <thead>
            <tr><th>#</th><th>URL Klaim</th><th>Token (Prefix)</th><th>Aksi</th></tr>
          </thead>
          <tbody>
            <tr v-for="(l, idx) in result.links.slice(0, 50)" :key="idx">
              <td>{{ (idx as number) + 1 }}</td>
              <td class="url-cell">{{ l.url }}</td>
              <td><code>{{ l.token.substring(0, 12) }}…</code></td>
              <td>
                <button class="copy-btn" @click="copyToClipboard(l.url, idx as number)">
                  {{ copiedIndex === idx ? '✅ Disalin!' : '📋 Copy Link' }}
                </button>
              </td>
            </tr>
            <tr v-if="result.links.length > 50">
              <td colspan="4" class="empty">+ {{ result.links.length - 50 }} more — download CSV untuk lihat semua</td>
            </tr>
          </tbody>
        </table>
      </div>
      <p class="info">⚠️ Token hanya ditampilkan sekali. Simpan CSV di tempat aman. Expires at: {{ result.expires_at }}</p>
    </div>
  </div>
</template>

<style scoped>
.generate-vouchers-page { max-width: 1200px; margin: 0 auto; }
h1 { font-size: 24px; margin-bottom: 4px; font-weight: 700; }
.subtitle { color: var(--muted); margin-bottom: 20px; font-size: 14px; }
.card { background: var(--card); border: 1px solid var(--border); border-radius: 10px; padding: 24px; }
.form { display: flex; flex-direction: column; gap: 14px; }
.form .row { display: grid; grid-template-columns: repeat(auto-fit, minmax(220px, 1fr)); gap: 14px; }
.form label { display: flex; flex-direction: column; gap: 5px; font-size: 13px; color: var(--muted); }
.form input, .form select { font-size: 14px; padding: 8px 10px; border-radius: 6px; border: 1px solid var(--border); background: var(--bg); color: var(--text); }
button[type="submit"] { background: var(--accent); color: white; border: none; padding: 10px 20px; border-radius: 6px; align-self: flex-start; font-weight: 600; cursor: pointer; transition: background 0.2s; }
button[type="submit"]:hover { background: #2563eb; }
.error { color: var(--danger); font-size: 14px; }
.block { margin-top: 32px; }
.result-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; }
.result-header h2 { margin: 0; font-size: 18px; }
.btn-download { background: var(--success); color: white; border: none; padding: 8px 16px; border-radius: 6px; font-weight: 600; cursor: pointer; }
.btn-download:hover { opacity: 0.9; }
.links-table { background: var(--card); border: 1px solid var(--border); border-radius: 10px; overflow-x: auto; }
table { width: 100%; border-collapse: collapse; text-align: left; }
th, td { padding: 10px 14px; border-bottom: 1px solid var(--border); font-size: 13px; }
th { background: rgba(0,0,0,0.02); color: var(--muted); font-weight: 600; font-size: 12px; }
.url-cell { font-family: monospace; font-size: 12px; max-width: 450px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.copy-btn { background: var(--bg); border: 1px solid var(--border); padding: 4px 10px; border-radius: 4px; font-size: 12px; cursor: pointer; font-weight: 500; transition: all 0.2s; }
.copy-btn:hover { background: var(--accent); color: white; border-color: var(--accent); }
.empty { text-align: center; color: var(--muted); padding: 16px; font-style: italic; }
.info { margin-top: 12px; font-size: 13px; color: var(--warning); }
</style>
