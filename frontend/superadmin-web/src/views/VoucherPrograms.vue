<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { api } from '../api/client'

const programs = ref<any[]>([])
const plans = ref<any[]>([])
const loading = ref(true)
const showCreate = ref(false)
const editingId = ref<string | null>(null)
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
    console.error(e)
  } finally {
    loading.value = false
  }
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
    alert('Gagal: ' + e.message)
  }
}

function startEdit(prog: any) {
  editingId.value = prog.id
  form.value = { ...prog }
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
    alert('Gagal: ' + e.message)
  }
}

async function deleteProgram(id: string) {
  if (!confirm('Hapus program ini?')) return
  try {
    await api.deleteVoucherProgram(id)
    await load()
  } catch (e: any) {
    alert('Gagal: ' + e.message)
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
  }
}
</script>

<template>
  <div>
    <div class="header">
      <h1>Voucher Programs</h1>
      <button @click="showCreate = !showCreate">{{ showCreate ? 'Cancel' : '+ New Program' }}</button>
    </div>

    <form v-if="showCreate || editingId" class="card form" @submit.prevent="editingId ? saveEdit() : create()">
      <h3>{{ editingId ? 'Edit Program' : 'New Voucher Program' }}</h3>
      <div class="row">
        <label>Name <input v-model="form.name" placeholder="Promo Juni 2026" required /></label>
        <label>Voucher Type
          <select v-model="form.voucher_type">
            <option value="free_months">Free Months</option>
            <option value="discount_percent">Discount %</option>
            <option value="discount_fixed">Discount Fixed (sen)</option>
            <option value="plan_upgrade">Plan Upgrade</option>
          </select>
        </label>
      </div>
      <div class="row">
        <label>Target Plan
          <select v-model="form.target_plan_id" required>
            <option value="">-- pilih plan --</option>
            <option v-for="p in plans" :key="p.id" :value="p.id">{{ p.name }}</option>
          </select>
        </label>
        <label>Duration (months) <input type="number" v-model="form.duration_months" min="1" /></label>
        <label>Discount Value <input type="number" v-model="form.discount_value" min="0" /></label>
        <label>Max Uses (0=unlimited) <input type="number" v-model="form.max_uses" min="0" /></label>
      </div>
      <div class="row">
        <label>Starts At <input type="datetime-local" v-model="form.starts_at" /></label>
        <label>Expires At <input type="datetime-local" v-model="form.expires_at" /></label>
      </div>
      <label>Description <textarea v-model="form.description" rows="2"></textarea></label>
      <div class="form-actions">
        <button type="submit">{{ editingId ? 'Save' : 'Create' }} Program</button>
        <button type="button" @click="resetForm">Cancel</button>
      </div>
    </form>

    <div v-if="loading" class="loading">Loading...</div>
    <div v-else-if="programs.length === 0" class="empty">Voucher programs will appear here. Use <strong>Generate Links</strong> tab to distribute.</div>
    <div v-else class="programs-list">
      <div v-for="prog in programs" :key="prog.id" class="card program-item">
        <div class="program-header">
          <div>
            <h3>{{ prog.name }}</h3>
            <p class="desc">{{ prog.description }}</p>
          </div>
          <div class="actions">
            <button class="btn-edit" @click="startEdit(prog)">Edit</button>
            <button class="btn-delete" @click="deleteProgram(prog.id)">Delete</button>
          </div>
        </div>
        <div class="program-details">
          <div><strong>Value:</strong> {{ prog.discount_value }}</div>
          <div><strong>Duration:</strong> {{ prog.duration_months }} months</div>
          <div><strong>Expires:</strong> {{ new Date(prog.expires_at).toLocaleDateString() }}</div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px; }
h1 { font-size: 24px; }
button { background: var(--accent); color: white; border: none; padding: 8px 14px; border-radius: 6px; }
button:hover { background: #2563eb; }
.card { background: var(--card); border: 1px solid var(--border); border-radius: 10px; padding: 20px; }
.form { display: flex; flex-direction: column; gap: 14px; margin-bottom: 24px; }
.form h3 { margin-bottom: 4px; }
.form .row { display: grid; grid-template-columns: repeat(auto-fit, minmax(180px, 1fr)); gap: 12px; }
.form label { display: flex; flex-direction: column; gap: 4px; font-size: 13px; color: var(--muted); }
.form input, .form select, .form textarea { font-size: 14px; }
.empty { padding: 40px; text-align: center; color: var(--muted); background: var(--card); border-radius: 10px; border: 1px dashed var(--border); }
.loading { padding: 40px; text-align: center; color: var(--muted); }
.programs-list { display: grid; gap: 16px; }
.program-item { padding: 16px; }
.program-header { display: flex; justify-content: space-between; align-items: start; margin-bottom: 12px; }
.program-header h3 { margin: 0; font-size: 16px; }
.desc { margin: 4px 0 0 0; font-size: 13px; color: var(--muted); }
.actions { display: flex; gap: 8px; }
.btn-edit, .btn-delete { font-size: 12px; padding: 4px 10px; border-radius: 4px; border: none; cursor: pointer; }
.btn-edit { background: var(--accent); color: white; }
.btn-edit:hover { background: #2563eb; }
.btn-delete { background: #ef4444; color: white; }
.btn-delete:hover { background: #dc2626; }
.program-details { display: grid; grid-template-columns: repeat(auto-fit, minmax(150px, 1fr)); gap: 12px; font-size: 13px; }
.program-details div { color: var(--muted); }
.form-actions { display: flex; gap: 12px; margin-top: 16px; }
.form-actions button { flex: 1; }
</style>
