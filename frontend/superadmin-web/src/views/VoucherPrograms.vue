<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { api } from '../api/client'

const programs = ref<any[]>([])
const plans = ref<any[]>([])
const loading = ref(true)
const showCreate = ref(false)
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
    // For now, show empty (no endpoint yet for list programs, only generate)
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
    showCreate.value = false
    await load()
  } catch (e: any) {
    alert('Gagal: ' + e.message)
  }
}
</script>

<template>
  <div>
    <div class="header">
      <h1>Voucher Programs</h1>
      <button @click="showCreate = !showCreate">{{ showCreate ? 'Cancel' : '+ New Program' }}</button>
    </div>

    <form v-if="showCreate" class="card form" @submit.prevent="create">
      <h3>New Voucher Program</h3>
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
      <button type="submit">Create Program</button>
    </form>

    <div v-if="loading" class="loading">Loading...</div>
    <div v-else class="empty">Voucher programs will appear here. Use <strong>Generate Links</strong> tab to distribute.</div>
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
</style>
