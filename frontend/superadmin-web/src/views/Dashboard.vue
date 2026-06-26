<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { api } from '../api/client'
import HAMonitoring from '../components/HAMonitoring.vue'

const data = ref<any>(null)
const loading = ref(true)
const error = ref('')

onMounted(async () => {
  try {
    const res = await api.getDashboard()
    data.value = res.data
  } catch (e: any) {
    error.value = e.message
  } finally {
    loading.value = false
  }
})

function formatRupiah(sen: number) {
  return 'Rp ' + sen.toLocaleString('id-ID')
}

async function impersonateTenant(tenantId: string) {
  if (!confirm('Login sebagai tenant ini? Tab baru akan dibuka.')) return
  try {
    const res = await fetch(`/api/superadmin/tenants/${tenantId}/impersonate`, {
      method: 'POST',
      headers: { 'Authorization': `Bearer ${localStorage.getItem('access_token')}` }
    })
    const data = await res.json()
    if (!res.ok) throw new Error(data.message || 'Impersonate failed')

    sessionStorage.setItem('superadmin_token', localStorage.getItem('access_token') || '')

    const url = new URL(globalThis.location.origin.replace('superadmin', 'app'))
    url.searchParams.set('impersonate_token', data.data.access_token)
    window.open(url.toString(), '_blank')

    alert(`Login sebagai ${data.data.tenant.business_name} berhasil. Cek tab baru.`)
  } catch (err: any) {
    alert(err.message || 'Impersonate failed')
  }
}
</script>

<template>
  <div>
    <h1>Overview Dashboard</h1>
    <p class="subtitle">Snapshot seluruh WCH Platform: UMKM, Campaign, voucher & revenue</p>

    <!-- HA Monitoring: WA Gateway + Chatbot -->
    <section class="block">
      <HAMonitoring />
    </section>

    
    <div v-if="loading" class="loading">Loading...</div>
    <div v-else-if="error" class="error">{{ error }}</div>
    <div v-else-if="data">
      <!-- Tenants -->
      <section class="grid">
        <div class="card">
          <div class="label">Total Tenants</div>
          <div class="value">{{ data.tenants.total }}</div>
        </div>
        <div class="card success">
          <div class="label">Active</div>
          <div class="value">{{ data.tenants.active }}</div>
        </div>
        <div class="card danger">
          <div class="label">Frozen</div>
          <div class="value">{{ data.tenants.frozen }}</div>
        </div>
        <div class="card accent">
          <div class="label">Revenue 30d (Xendit)</div>
          <div class="value">{{ formatRupiah(data.revenue_30d_sen) }}</div>
        </div>
      </section>

      <!-- Vouchers -->
      <section class="block">
        <h2>Vouchers (30 hari terakhir)</h2>
        <div class="grid">
          <div class="card">
            <div class="label">Links Generated</div>
            <div class="value">{{ data.vouchers_30d.links_generated }}</div>
          </div>
          <div class="card">
            <div class="label">Links Redeemed</div>
            <div class="value">{{ data.vouchers_30d.links_redeemed }}</div>
          </div>
          <div class="card">
            <div class="label">Active Programs</div>
            <div class="value">{{ data.vouchers_30d.active_programs }}</div>
          </div>
        </div>
      </section>

      <!-- Subs by plan -->
      <section class="block">
        <h2>Active Subscriptions by Plan</h2>
        <table>
          <thead>
            <tr><th>Plan ID</th><th>Count</th></tr>
          </thead>
          <tbody>
            <tr v-for="(cnt, planId) in data.subs_by_plan" :key="planId">
              <td><code>{{ planId }}</code></td>
              <td>{{ cnt }}</td>
            </tr>
            <tr v-if="Object.keys(data.subs_by_plan).length === 0">
              <td colspan="2" class="empty">No active subscriptions</td>
            </tr>
          </tbody>
        </table>
      </section>

      <!-- Recent frozen -->
      <section class="block">
        <h2>Recent Frozen Accounts (top 10)</h2>
        <table>
          <thead>
            <tr><th>Tenant</th><th>Plan</th><th>Expired At</th><th>Frozen At</th><th>Actions</th></tr>
          </thead>
          <tbody>
            <tr v-for="t in data.recent_frozen" :key="t.id">
              <td>{{ t.name }}</td>
              <td><code>{{ t.plan }}</code></td>
              <td>{{ t.expired_at || '-' }}</td>
              <td>{{ t.frozen_at || '-' }}</td>
              <td>
                <button class="btn-impersonate" @click="impersonateTenant(t.id)">🔓 Login Sebagai</button>
              </td>
            </tr>
            <tr v-if="data.recent_frozen.length === 0">
              <td colspan="5" class="empty">No frozen accounts</td>
            </tr>
          </tbody>
        </table>
      </section>
    </div>
  </div>
</template>

<style scoped>
h1 { font-size: 24px; margin-bottom: 4px; }
.subtitle { color: var(--muted); margin-bottom: 24px; font-size: 14px; }
.block { margin-top: 32px; }
.block h2 { font-size: 16px; margin-bottom: 12px; color: var(--muted); text-transform: uppercase; letter-spacing: 0.5px; }
.grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 16px; }
.card { background: var(--card); padding: 20px; border-radius: 10px; border: 1px solid var(--border); }
.card .label { font-size: 12px; color: var(--muted); text-transform: uppercase; letter-spacing: 0.5px; }
.card .value { font-size: 28px; font-weight: 700; margin-top: 6px; }
.card.success .value { color: var(--success); }
.card.danger .value { color: var(--danger); }
.card.accent .value { color: var(--accent); }
.loading, .error { padding: 40px; text-align: center; color: var(--muted); }
.error { color: var(--danger); }
.empty { text-align: center; color: var(--muted); padding: 20px; }
.btn-impersonate {
  background: linear-gradient(135deg, #7c3aed, #4f46e5);
  color: white;
  border: none;
  padding: 6px 12px;
  border-radius: 6px;
  font-size: 13px;
  cursor: pointer;
  font-weight: 600;
  transition: transform 0.2s, box-shadow 0.2s;
}
.btn-impersonate:hover {
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(139, 92, 246, 0.3);
}
</style>
