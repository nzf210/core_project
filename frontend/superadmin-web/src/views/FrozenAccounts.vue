<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { api } from '../api/client'

const frozen = ref<any[]>([])
const loading = ref(true)
const error = ref('')

onMounted(async () => {
  try {
    frozen.value = await api.getFrozenAccounts()
  } catch (e: any) {
    error.value = e.message
  } finally {
    loading.value = false
  }
})

function sendReminderWA(name: string) {
  // Placeholder: trigger WA via notification-service
  const msg = encodeURIComponent(`Halo ${name}, akun WCH Anda freeze. Redeem voucher untuk aktifkan kembali: ${globalThis.location.origin}/redeem`)
  globalThis.open(`https://wa.me/?text=${msg}`, '_blank')
}
</script>

<template>
  <div>
    <h1>Frozen Accounts</h1>
    <p class="subtitle">Akun yang melewati masa aktif subscription. User masih bisa login (read-only) sampai redeem voucher baru.</p>

    <div v-if="loading" class="loading">Loading...</div>
    <div v-else-if="error" class="error">{{ error }}</div>
    <div v-else-if="frozen.length === 0" class="empty">
      🎉 Tidak ada akun yang freeze saat ini.
    </div>
    <table v-else>
      <thead>
        <tr>
          <th>Tenant</th>
          <th>Plan (last)</th>
          <th>Expired At</th>
          <th>Frozen At</th>
          <th>Action</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="t in frozen" :key="t.id">
          <td>{{ t.name }}</td>
          <td><code>{{ t.plan }}</code></td>
          <td>{{ t.expired_at || '-' }}</td>
          <td>{{ t.frozen_at || '-' }}</td>
          <td>
            <button class="reminder-btn" @click="sendReminderWA(t.name)">📱 Kirim Reminder WA</button>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<style scoped>
h1 { font-size: 24px; margin-bottom: 4px; }
.subtitle { color: var(--muted); margin-bottom: 20px; font-size: 14px; }
table { background: var(--card); border-radius: 10px; overflow: hidden; border: 1px solid var(--border); }
.reminder-btn { background: var(--success); color: white; border: none; padding: 6px 10px; border-radius: 4px; font-size: 12px; }
.empty { padding: 60px 20px; text-align: center; color: var(--success); background: var(--card); border-radius: 10px; border: 1px dashed var(--border); }
.loading, .error { padding: 40px; text-align: center; }
.error { color: var(--danger); }
</style>
