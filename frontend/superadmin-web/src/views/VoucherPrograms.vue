<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { api } from '../api/client'

const router = useRouter()
const programs = ref<any[]>([])
const plans = ref<any[]>([])
const loading = ref(true)
const showCreate = ref(false)
const editingId = ref<string | null>(null)
const copiedId = ref<string | null>(null)

const form = ref({
  name: '',
  description: '',
  voucher_type: 'free_months',
  discount_value: 0,
  target_plan_id: '',
  duration_months: 1,
  max_uses: 0,
  starts_at: '',
  expires_at: '',
  is_active: true,
})

onMounted(async () => {
  await load()
})

async function load() {
  loading.value = true
  try {
    programs.value = (await api.listVoucherPrograms()).data || []
    plans.value = (await api.listPlans()).data || []
  } catch (e: any) {
    console.error('Failed to load voucher programs:', e)
  } finally {
    loading.value = false
  }
}

function formatDateForInput(dtStr: string | null | undefined): string {
  if (!dtStr) return ''
  try {
    const d = new Date(dtStr)
    if (isNaN(d.getTime())) return ''
    const pad = (n: number) => n.toString().padStart(2, '0')
    return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
  } catch {
    return ''
  }
}

function startEdit(prog: any) {
  editingId.value = prog.id
  form.value = {
    name: prog.name || '',
    description: prog.description || '',
    voucher_type: prog.voucher_type || 'free_months',
    discount_value: prog.discount_value || 0,
    target_plan_id: prog.target_plan_id || '',
    duration_months: prog.duration_months || 1,
    max_uses: prog.max_uses || 0,
    starts_at: formatDateForInput(prog.starts_at),
    expires_at: formatDateForInput(prog.expires_at),
    is_active: prog.is_active !== undefined ? prog.is_active : true,
  }
  showCreate.value = false
  window.scrollTo({ top: 0, behavior: 'smooth' })
}

async function create() {
  if (!form.value.name || !form.value.target_plan_id) {
    alert('Nama dan target plan wajib diisi')
    return
  }
  try {
    await api.createVoucherProgram({
      ...form.value,
      discount_value: Number(form.value.discount_value),
      duration_months: Number(form.value.duration_months),
      max_uses: Number(form.value.max_uses),
    })
    resetForm()
    await load()
  } catch (e: any) {
    alert('Gagal membuat program: ' + e.message)
  }
}

async function saveEdit() {
  if (!editingId.value) return
  try {
    await api.updateVoucherProgram(editingId.value, {
      ...form.value,
      discount_value: Number(form.value.discount_value),
      duration_months: Number(form.value.duration_months),
      max_uses: Number(form.value.max_uses),
    })
    resetForm()
    await load()
  } catch (e: any) {
    alert('Gagal menyimpan program: ' + e.message)
  }
}

async function deleteProgram(id: string) {
  if (!confirm('Hapus / nonaktifkan program ini?')) return
  try {
    await api.deleteVoucherProgram(id)
    await load()
  } catch (e: any) {
    alert('Gagal menghapus: ' + e.message)
  }
}

function resetForm() {
  showCreate.value = false
  editingId.value = null
  form.value = {
    name: '',
    description: '',
    voucher_type: 'free_months',
    discount_value: 0,
    target_plan_id: '',
    duration_months: 1,
    max_uses: 0,
    starts_at: '',
    expires_at: '',
    is_active: true,
  }
}

function copyId(id: string) {
  if (!id) return
  if (navigator.clipboard && navigator.clipboard.writeText) {
    navigator.clipboard.writeText(id).then(() => {
      copiedId.value = id
      setTimeout(() => {
        if (copiedId.value === id) copiedId.value = null
      }, 2000)
    }).catch(() => fallbackCopy(id))
  } else {
    fallbackCopy(id)
  }
}

function fallbackCopy(text: string) {
  const input = document.createElement('input')
  input.value = text
  document.body.appendChild(input)
  input.select()
  document.execCommand('copy')
  document.body.removeChild(input)
  copiedId.value = text
  setTimeout(() => {
    if (copiedId.value === text) copiedId.value = null
  }, 2000)
}

function goToGenerate(progId: string) {
  router.push({ path: '/vouchers/generate', query: { program_id: progId } })
}

function getPlanName(planId: string): string {
  const p = plans.value.find((item: any) => item.id === planId)
  return p ? p.name : planId.toUpperCase()
}

function formatVoucherType(type: string): string {
  const map: Record<string, string> = {
    free_months: 'Free Months',
    bonus_months: 'Bonus Months',
    discount_percent: 'Diskon %',
    discount_fixed: 'Potongan Tetap',
    plan_upgrade: 'Plan Upgrade',
  }
  return map[type] || type
}

function formatVoucherValue(prog: any): string {
  if (prog.voucher_type === 'discount_percent') {
    return `${prog.discount_value}%`
  }
  if (prog.voucher_type === 'discount_fixed') {
    return `Rp ${(prog.discount_value / 100).toLocaleString('id-ID')}`
  }
  return `${prog.duration_months} bln gratis`
}

function formatExpires(expiresAt: string | null | undefined): string {
  if (!expiresAt) return 'Selamanya (Tanpa Expired)'
  try {
    const d = new Date(expiresAt)
    if (isNaN(d.getTime())) return 'Tanpa Expired'
    return d.toLocaleDateString('id-ID', { year: 'numeric', month: 'short', day: 'numeric' })
  } catch {
    return 'Tanpa Expired'
  }
}
</script>

<template>
  <div class="voucher-programs-page">
    <div class="header">
      <div>
        <h1>Voucher Programs</h1>
        <p class="subtitle">Kelola program voucher dan buat link/kode voucher untuk pelanggan & reseller.</p>
      </div>
      <button class="btn-primary" @click="showCreate = !showCreate">
        {{ showCreate ? '✕ Batal' : '+ Program Baru' }}
      </button>
    </div>

    <!-- Create / Edit Form -->
    <form v-if="showCreate || editingId" class="card form" @submit.prevent="editingId ? saveEdit() : create()">
      <div class="form-title-row">
        <h3>{{ editingId ? 'Edit Program Voucher' : 'Buat Program Voucher Baru' }}</h3>
        <span v-if="editingId" class="editing-badge">Editing: {{ form.name }}</span>
      </div>

      <div class="row">
        <label>Nama Program
          <input v-model="form.name" placeholder="cth: Promo Merdeka 2026" required />
        </label>
        <label>Tipe Voucher
          <select v-model="form.voucher_type">
            <option value="free_months">Free Months (Akses Gratis)</option>
            <option value="discount_percent">Diskon Persentase (%)</option>
            <option value="discount_fixed">Potongan Harga Tetap (Rp)</option>
            <option value="plan_upgrade">Upgrade Paket</option>
          </select>
        </label>
      </div>

      <div class="row">
        <label>Target Plan
          <select v-model="form.target_plan_id" required>
            <option value="">-- Pilih Paket --</option>
            <option v-for="p in plans" :key="p.id" :value="p.id">{{ p.name }}</option>
          </select>
        </label>
        <label>Durasi (Bulan)
          <input type="number" v-model="form.duration_months" min="1" />
        </label>
        <label>Nilai Diskon (0 jika gratis)
          <input type="number" v-model="form.discount_value" min="0" />
        </label>
        <label>Maksimal Pemakaian (0 = tanpa batas)
          <input type="number" v-model="form.max_uses" min="0" />
        </label>
      </div>

      <div class="row">
        <label>Mulai Berlaku
          <input type="datetime-local" v-model="form.starts_at" />
        </label>
        <label>Berakhir Pada
          <input type="datetime-local" v-model="form.expires_at" />
        </label>
        <label v-if="editingId" class="checkbox-label">
          <span>Status Aktif</span>
          <div class="switch-row">
            <input type="checkbox" v-model="form.is_active" id="is_active_toggle" />
            <label for="is_active_toggle">{{ form.is_active ? 'Aktif' : 'Nonaktif' }}</label>
          </div>
        </label>
      </div>

      <label>Deskripsi Program
        <textarea v-model="form.description" rows="2" placeholder="Catatan atau syarat ketentuan voucher..."></textarea>
      </label>

      <div class="form-actions">
        <button type="submit" class="btn-primary">{{ editingId ? 'Simpan Perubahan' : 'Buat Program' }}</button>
        <button type="button" class="btn-secondary" @click="resetForm">Batal</button>
      </div>
    </form>

    <!-- Program List -->
    <div v-if="loading" class="loading">Memuat daftar voucher program...</div>
    <div v-else-if="programs.length === 0" class="empty">
      Belum ada voucher program. Klik tombol <strong>+ Program Baru</strong> untuk mulai membuat program voucher.
    </div>
    <div v-else class="programs-list">
      <div v-for="prog in programs" :key="prog.id" class="card program-item">
        <div class="program-header">
          <div class="program-info">
            <div class="title-row">
              <h3>{{ prog.name }}</h3>
              <span class="badge" :class="prog.is_active ? 'badge-active' : 'badge-inactive'">
                {{ prog.is_active ? 'Aktif' : 'Nonaktif' }}
              </span>
              <span v-if="prog.target_plan_id" class="badge badge-plan">
                {{ getPlanName(prog.target_plan_id) }}
              </span>
            </div>
            <p class="desc">{{ prog.description || 'Tidak ada deskripsi' }}</p>

            <!-- Prominent ID Display & Copy -->
            <div class="id-row">
              <span class="id-label">ID Program:</span>
              <code class="id-code">{{ prog.id }}</code>
              <button class="btn-copy" @click="copyId(prog.id)" :title="copiedId === prog.id ? 'Tersalin!' : 'Copy Program ID'">
                {{ copiedId === prog.id ? '✅ Disalin!' : '📋 Copy ID' }}
              </button>
            </div>
          </div>

          <div class="actions">
            <button class="btn-generate" @click="goToGenerate(prog.id)" title="Buat link klaim untuk program ini">
              🔗 Pakai / Generate Links
            </button>
            <button class="btn-edit" @click="startEdit(prog)">Edit</button>
            <button class="btn-delete" @click="deleteProgram(prog.id)">Hapus</button>
          </div>
        </div>

        <div class="program-details">
          <div><span class="detail-label">Tipe:</span> {{ formatVoucherType(prog.voucher_type) }}</div>
          <div><span class="detail-label">Nilai:</span> {{ formatVoucherValue(prog) }}</div>
          <div><span class="detail-label">Durasi:</span> {{ prog.duration_months }} bulan</div>
          <div><span class="detail-label">Pemakaian:</span> {{ prog.uses_count }} / {{ prog.max_uses > 0 ? prog.max_uses : '∞' }}</div>
          <div><span class="detail-label">Kedaluwarsa:</span> {{ formatExpires(prog.expires_at) }}</div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.voucher-programs-page { max-width: 1200px; margin: 0 auto; }
.header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 24px; gap: 16px; flex-wrap: wrap; }
h1 { font-size: 24px; margin: 0 0 4px 0; font-weight: 700; }
.subtitle { color: var(--muted); margin: 0; font-size: 14px; }
.btn-primary { background: var(--accent); color: white; border: none; padding: 9px 16px; border-radius: 6px; font-weight: 600; cursor: pointer; transition: background 0.2s; }
.btn-primary:hover { background: #2563eb; }
.btn-secondary { background: var(--bg); color: var(--text); border: 1px solid var(--border); padding: 9px 16px; border-radius: 6px; cursor: pointer; }
.btn-secondary:hover { background: var(--border); }
.card { background: var(--card); border: 1px solid var(--border); border-radius: 10px; padding: 20px; }
.form { display: flex; flex-direction: column; gap: 14px; margin-bottom: 24px; box-shadow: 0 4px 12px rgba(0,0,0,0.05); }
.form-title-row { display: flex; align-items: center; justify-content: space-between; margin-bottom: 4px; }
.form-title-row h3 { margin: 0; font-size: 18px; }
.editing-badge { font-size: 12px; background: rgba(59, 130, 246, 0.15); color: #3b82f6; padding: 3px 8px; border-radius: 4px; }
.form .row { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 12px; }
.form label { display: flex; flex-direction: column; gap: 5px; font-size: 13px; color: var(--muted); }
.form input, .form select, .form textarea { font-size: 14px; padding: 8px 10px; border-radius: 6px; border: 1px solid var(--border); background: var(--bg); color: var(--text); }
.checkbox-label { justify-content: center; }
.switch-row { display: flex; align-items: center; gap: 8px; margin-top: 4px; }
.empty { padding: 48px; text-align: center; color: var(--muted); background: var(--card); border-radius: 10px; border: 1px dashed var(--border); }
.loading { padding: 48px; text-align: center; color: var(--muted); }
.programs-list { display: grid; gap: 16px; }
.program-item { padding: 18px; transition: border-color 0.2s; }
.program-item:hover { border-color: var(--accent); }
.program-header { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 14px; gap: 16px; flex-wrap: wrap; }
.program-info { flex: 1; min-width: 260px; }
.title-row { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; margin-bottom: 6px; }
.title-row h3 { margin: 0; font-size: 17px; font-weight: 600; }
.desc { margin: 0 0 10px 0; font-size: 13px; color: var(--muted); }
.id-row { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; margin-top: 4px; }
.id-label { font-size: 12px; color: var(--muted); font-weight: 500; }
.id-code { font-family: monospace; font-size: 12px; background: rgba(59, 130, 246, 0.1); color: #3b82f6; padding: 3px 7px; border-radius: 4px; border: 1px solid rgba(59, 130, 246, 0.2); word-break: break-all; }
.btn-copy { font-size: 11px; padding: 3px 8px; border-radius: 4px; border: 1px solid var(--border); background: var(--bg); color: var(--text); cursor: pointer; transition: all 0.2s; font-weight: 500; }
.btn-copy:hover { background: var(--accent); color: white; border-color: var(--accent); }
.badge { font-size: 11px; padding: 2px 7px; border-radius: 4px; font-weight: 600; text-transform: uppercase; }
.badge-active { background: rgba(16, 185, 129, 0.15); color: #10b981; }
.badge-inactive { background: rgba(239, 68, 68, 0.15); color: #ef4444; }
.badge-plan { background: rgba(99, 102, 241, 0.15); color: #6366f1; }
.actions { display: flex; gap: 8px; align-items: center; flex-wrap: wrap; }
.btn-generate { font-size: 12px; padding: 6px 12px; border-radius: 5px; border: 1px solid rgba(16, 185, 129, 0.3); background: rgba(16, 185, 129, 0.1); color: #10b981; font-weight: 600; cursor: pointer; transition: all 0.2s; }
.btn-generate:hover { background: #10b981; color: white; }
.btn-edit, .btn-delete { font-size: 12px; padding: 6px 12px; border-radius: 5px; border: none; cursor: pointer; font-weight: 500; }
.btn-edit { background: var(--accent); color: white; }
.btn-edit:hover { background: #2563eb; }
.btn-delete { background: #dc2626; color: white; }
.btn-delete:hover { background: #b91c1c; }
.program-details { display: grid; grid-template-columns: repeat(auto-fit, minmax(160px, 1fr)); gap: 10px; font-size: 13px; padding-top: 12px; border-top: 1px solid var(--border); color: var(--text); }
.detail-label { color: var(--muted); font-size: 12px; margin-right: 4px; }
.form-actions { display: flex; gap: 12px; margin-top: 16px; }
.form-actions button { flex: 1; }
</style>
