<script setup lang="ts">
import { ref } from 'vue'
import { api } from '../api/client'

const programId = ref('')
const count = ref(50)
const validDays = ref(365)
const baseUrl = ref('https://app.wch.id')
const generating = ref(false)
const result = ref<any>(null)
const error = ref('')

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

function copyToClipboard(text: string) {
  navigator.clipboard.writeText(text)
}
</script>

<template>
  <div>
    <h1>Generate Voucher Links</h1>
    <p class="subtitle">Bulk generate link voucher (1 link = 1 klaim). Distribusikan ke reseller/customer.</p>

    <form class="card form" @submit.prevent="generate">
      <div class="row">
        <label>Program ID (UUID)
          <input v-model="programId" placeholder="uuid dari voucher_programs" required />
        </label>
        <label>Count (max 1000)
          <input type="number" v-model="count" min="1" max="1000" required />
        </label>
        <label>Valid Days
          <input type="number" v-model="validDays" min="1" max="3650" />
        </label>
        <label>Base URL
          <input v-model="baseUrl" placeholder="https://app.wch.id" />
        </label>
      </div>
      <button type="submit" :disabled="generating">{{ generating ? 'Generating...' : 'Generate' }}</button>
      <div v-if="error" class="error">{{ error }}</div>
    </form>

    <div v-if="result" class="block">
      <div class="result-header">
        <h2>Generated: {{ result.count }} links</h2>
        <button @click="downloadCSV">📥 Download CSV</button>
      </div>
      <div class="links-table">
        <table>
          <thead>
            <tr><th>#</th><th>URL</th><th>Token (8 char prefix)</th><th>Action</th></tr>
          </thead>
          <tbody>
            <tr v-for="(l, idx) in result.links.slice(0, 50)" :key="idx">
              <td>{{ (idx as number) + 1 }}</td>
              <td class="url-cell">{{ l.url }}</td>
              <td><code>{{ l.token.substring(0, 12) }}…</code></td>
              <td><button class="copy-btn" @click="copyToClipboard(l.url)">Copy</button></td>
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
h1 { font-size: 24px; margin-bottom: 4px; }
.subtitle { color: var(--muted); margin-bottom: 20px; font-size: 14px; }
.card { background: var(--card); border: 1px solid var(--border); border-radius: 10px; padding: 24px; }
.form { display: flex; flex-direction: column; gap: 14px; }
.form .row { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 12px; }
.form label { display: flex; flex-direction: column; gap: 4px; font-size: 13px; color: var(--muted); }
.form input { font-size: 14px; }
button[type="submit"] { background: var(--accent); color: white; border: none; padding: 10px 18px; border-radius: 6px; align-self: flex-start; }
.error { color: var(--danger); font-size: 14px; }
.block { margin-top: 32px; }
.result-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; }
.result-header button { background: var(--success); color: white; border: none; padding: 8px 14px; border-radius: 6px; }
.links-table { background: var(--card); border: 1px solid var(--border); border-radius: 10px; overflow: hidden; }
.url-cell { font-family: monospace; font-size: 12px; max-width: 400px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.copy-btn { background: var(--bg); border: 1px solid var(--border); padding: 4px 8px; border-radius: 4px; font-size: 12px; }
.info { margin-top: 12px; font-size: 13px; color: var(--warning); }
.empty { text-align: center; color: var(--muted); padding: 12px; }
</style>
