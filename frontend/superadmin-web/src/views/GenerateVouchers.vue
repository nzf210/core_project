<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { api } from '../api/client'

const route = useRoute()
const generationType = ref<'link' | 'code'>('link')
const programId = ref('')
const count = ref(50)
const validDays = ref(365)
const baseUrl = ref('https://app.wch.id')
const generating = ref(false)
const result = ref<any>(null)
const error = ref('')
const programs = ref<any[]>([])
const loadingPrograms = ref(false)
const copiedKey = ref<string | null>(null)

function resolveDefaultBaseUrl(): string {
  if (typeof window !== 'undefined') {
    const origin = window.location.origin
    if (origin.includes('localhost') || origin.includes('127.0.0.1')) {
      return 'http://localhost:3201'
    }
    if (origin.includes('22002')) {
      return origin.replace('22002', '22001')
    }
    if (origin.includes('12002')) {
      return origin.replace('12002', '12001')
    }
  }
  return 'https://app.wch.id'
}

onMounted(async () => {
  baseUrl.value = resolveDefaultBaseUrl()

  if (route.query.program_id) {
    programId.value = String(route.query.program_id)
  }

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
    error.value = 'Pilih program voucher dan tentukan jumlah'
    return
  }
  generating.value = true
  error.value = ''
  result.value = null

  try {
    const prog = programs.value.find((p: any) => p.id === programId.value)
    if (generationType.value === 'link') {
      const res = await api.generateVoucherLinks({
        program_id: programId.value,
        count: Number(count.value),
        valid_days: Number(validDays.value),
        base_url: baseUrl.value,
      })
      result.value = {
        type: 'link',
        ...res.data,
      }
    } else {
      const planId = prog?.target_plan_id || 'lite'
      const res = await api.generateVouchers({
        program_id: programId.value,
        plan_id: planId,
        validity_days: Number(validDays.value),
        quantity: Number(count.value),
        program_name: prog?.name || `Program ${planId.toUpperCase()}`,
        voucher_type: prog?.voucher_type || 'bonus_months',
        discount_value: prog?.discount_value || 0,
      })
      result.value = {
        type: 'code',
        ...res.data,
      }
    }
  } catch (e: any) {
    error.value = e.message
  } finally {
    generating.value = false
  }
}

function downloadCSV() {
  if (!result.value) return
  let csv = ''
  let filename = ''

  if (result.value.type === 'link' && result.value.links) {
    const rows = result.value.links.map((l: any) => `"${l.url}","${l.token}"`)
    csv = 'redeem_url,token\n' + rows.join('\n')
    filename = `voucher-links-${Date.now()}.csv`
  } else if (result.value.type === 'code' && result.value.codes) {
    const rows = result.value.codes.map((c: any) => `"${c.code}","${c.validity_days}"`)
    csv = 'voucher_code,validity_days\n' + rows.join('\n')
    filename = `voucher-codes-${Date.now()}.csv`
  }
  if (!csv) return

  const blob = new Blob([csv], { type: 'text/csv' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  a.click()
  URL.revokeObjectURL(url)
}

function copyText(text: string, key: string) {
  if (navigator.clipboard && navigator.clipboard.writeText) {
    navigator.clipboard.writeText(text).then(() => {
      copiedKey.value = key
      setTimeout(() => {
        if (copiedKey.value === key) copiedKey.value = null
      }, 2000)
    })
  } else {
    const input = document.createElement('input')
    input.value = text
    document.body.appendChild(input)
    input.select()
    document.execCommand('copy')
    input.remove()
    copiedKey.value = key
    setTimeout(() => {
      if (copiedKey.value === key) copiedKey.value = null
    }, 2000)
  }
}

function copyAll() {
  if (!result.value) return
  let allText = ''
  if (result.value.type === 'link' && result.value.links) {
    allText = result.value.links.map((l: any) => l.url).join('\n')
  } else if (result.value.type === 'code' && result.value.codes) {
    allText = result.value.codes.map((c: any) => c.code).join('\n')
  }
  if (allText) copyText(allText, 'all')
}
</script>

<template>
  <div class="generate-vouchers-page">
    <h1>Generate Voucher & Link Klaim</h1>
    <p class="subtitle">Buat batch voucher yang dapat langsung diklaim customer melalui URL link atau kode unik.</p>

    <form class="card form" @submit.prevent="generate">
      <!-- Mode Selection -->
      <div class="type-selector">
        <button
          type="button"
          :class="['type-btn', { active: generationType === 'link' }]"
          @click="generationType = 'link'"
        >
          🔗 Link Klaim (Instant URL)
        </button>
        <button
          type="button"
          :class="['type-btn', { active: generationType === 'code' }]"
          @click="generationType = 'code'"
        >
          🎟️ Kode Voucher (Alphanumeric)
        </button>
      </div>

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
        <label>Jumlah Voucher (max 1000)
          <input type="number" v-model="count" min="1" max="1000" required />
        </label>
        <label>Masa Aktif (hari)
          <input type="number" v-model="validDays" min="1" max="3650" />
        </label>
        <label v-if="generationType === 'link'">Base URL Frontend
          <input v-model="baseUrl" placeholder="https://app.wch.id" />
        </label>
      </div>

      <button type="submit" :disabled="generating">{{ generating ? 'Sedang Memproses...' : (generationType === 'link' ? 'Generate Link Klaim' : 'Generate Kode Voucher') }}</button>
      <div v-if="error" class="error">{{ error }}</div>
    </form>

    <!-- Results Display -->
    <div v-if="result" class="block">
      <div class="result-header">
        <h2>Berhasil dibuat: {{ result.count }} {{ result.type === 'link' ? 'link klaim' : 'kode voucher' }}</h2>
        <div class="header-actions">
          <button type="button" class="btn-copy-all" @click="copyAll">
            {{ copiedKey === 'all' ? '✅ Semua Disalin!' : '📋 Salin Semua' }}
          </button>
          <button type="button" class="btn-download" @click="downloadCSV">📥 Download CSV</button>
        </div>
      </div>

      <!-- Links Table -->
      <div v-if="result.type === 'link'" class="links-table">
        <table>
          <thead>
            <tr><th>#</th><th>URL Klaim</th><th>Token Prefix</th><th>Aksi</th></tr>
          </thead>
          <tbody>
            <tr v-for="(l, idx) in result.links.slice(0, 50)" :key="idx">
              <td>{{ (idx as number) + 1 }}</td>
              <td class="url-cell">{{ l.url }}</td>
              <td><code>{{ l.token.substring(0, 10) }}…</code></td>
              <td class="action-cell">
                <button type="button" class="copy-btn" @click="copyText(l.url, `url-${idx}`)">
                  {{ copiedKey === `url-${idx}` ? '✅ Link Disalin' : '🔗 Copy Link' }}
                </button>
                <button type="button" class="copy-btn btn-token" @click="copyText(l.token, `tok-${idx}`)">
                  {{ copiedKey === `tok-${idx}` ? '✅ Token Disalin' : '🔑 Copy Token' }}
                </button>
              </td>
            </tr>
            <tr v-if="result.links.length > 50">
              <td colspan="4" class="empty">+ {{ result.links.length - 50 }} link lainnya tersedia di file CSV.</td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- Codes Table -->
      <div v-else-if="result.type === 'code'" class="links-table">
        <table>
          <thead>
            <tr><th>#</th><th>Kode Voucher</th><th>Target Paket</th><th>Masa Aktif</th><th>Aksi</th></tr>
          </thead>
          <tbody>
            <tr v-for="(c, idx) in result.codes.slice(0, 50)" :key="idx">
              <td>{{ (idx as number) + 1 }}</td>
              <td><strong class="code-badge">{{ c.code }}</strong></td>
              <td>{{ (result.plan_id || 'LITE').toUpperCase() }}</td>
              <td>{{ c.validity_days }} Hari</td>
              <td>
                <button type="button" class="copy-btn" @click="copyText(c.code, `code-${idx}`)">
                  {{ copiedKey === `code-${idx}` ? '✅ Disalin!' : '📋 Copy Kode' }}
                </button>
              </td>
            </tr>
            <tr v-if="result.codes.length > 50">
              <td colspan="5" class="empty">+ {{ result.codes.length - 50 }} kode lainnya tersedia di file CSV.</td>
            </tr>
          </tbody>
        </table>
      </div>

      <div class="info-box">
        <p>💡 <strong>Panduan Klaim:</strong></p>
        <ul>
          <li><strong>Link Klaim:</strong> Cukup berikan URL ke customer. Customer bisa langsung membukanya di browser atau klik menu login/register untuk aktivasi otomatis.</li>
          <li><strong>Kode Voucher / Token:</strong> Dapat diinputkan langsung oleh tenant di menu <em>Onboarding</em>, menu <em>Pengaturan</em>, atau halaman <code>/redeem</code>.</li>
        </ul>
      </div>
    </div>
  </div>
</template>

<style scoped>
.generate-vouchers-page { max-width: 1200px; margin: 0 auto; }
h1 { font-size: 24px; margin-bottom: 4px; font-weight: 700; }
.subtitle { color: var(--muted); margin-bottom: 20px; font-size: 14px; }
.card { background: var(--card); border: 1px solid var(--border); border-radius: 10px; padding: 24px; }
.form { display: flex; flex-direction: column; gap: 14px; }
.type-selector { display: flex; gap: 10px; margin-bottom: 8px; }
.type-btn {
  padding: 8px 16px; border: 1px solid var(--border); border-radius: 6px;
  background: var(--bg); color: var(--text); font-weight: 500; cursor: pointer; transition: all 0.2s;
}
.type-btn.active { background: #2563eb; color: #ffffff; border-color: #2563eb; }
.form .row { display: grid; grid-template-columns: repeat(auto-fit, minmax(220px, 1fr)); gap: 14px; }
.form label { display: flex; flex-direction: column; gap: 5px; font-size: 13px; color: var(--muted); }
.form input, .form select { font-size: 14px; padding: 8px 10px; border-radius: 6px; border: 1px solid var(--border); background: var(--bg); color: var(--text); }
button[type="submit"] { background: #2563eb; color: #ffffff; border: none; padding: 10px 20px; border-radius: 6px; align-self: flex-start; font-weight: 600; cursor: pointer; transition: background 0.2s; }
button[type="submit"]:hover { background: #1d4ed8; }
.error { color: #f87171; font-size: 14px; }
.block { margin-top: 32px; }
.result-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; flex-wrap: wrap; gap: 10px; }
.result-header h2 { margin: 0; font-size: 18px; }
.header-actions { display: flex; gap: 8px; }
.btn-copy-all { background: var(--bg); border: 1px solid var(--border); color: var(--text); padding: 8px 14px; border-radius: 6px; font-weight: 600; cursor: pointer; }
.btn-download { background: #047857; color: #ffffff; border: none; padding: 8px 16px; border-radius: 6px; font-weight: 600; cursor: pointer; }
.btn-download:hover { background: #065f46; }
.links-table { background: var(--card); border: 1px solid var(--border); border-radius: 10px; overflow-x: auto; }
table { width: 100%; border-collapse: collapse; text-align: left; }
th, td { padding: 10px 14px; border-bottom: 1px solid var(--border); font-size: 13px; }
th { background: rgba(0,0,0,0.02); color: var(--muted); font-weight: 600; font-size: 12px; }
.url-cell { font-family: monospace; font-size: 12px; max-width: 420px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.code-badge { font-family: monospace; font-size: 13px; background: var(--bg); padding: 3px 8px; border-radius: 4px; border: 1px solid var(--border); }
.action-cell { display: flex; gap: 6px; }
.copy-btn { background: var(--bg); border: 1px solid var(--border); padding: 4px 10px; border-radius: 4px; font-size: 12px; cursor: pointer; font-weight: 500; transition: all 0.2s; }
.copy-btn:hover { background: #2563eb; color: #ffffff; border-color: #2563eb; }
.btn-token { opacity: 0.85; }
.empty { text-align: center; color: var(--muted); padding: 16px; font-style: italic; }
.info-box { margin-top: 14px; padding: 14px; background: var(--card); border: 1px solid var(--border); border-radius: 8px; font-size: 13px; }
.info-box ul { margin: 6px 0 0 16px; padding: 0; line-height: 1.6; }
</style>
