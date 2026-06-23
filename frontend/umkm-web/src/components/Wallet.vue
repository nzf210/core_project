<template>
  <div class="wallet-page">
    <h2>💳 Wallet</h2>
    <p class="subtitle">Kelola saldo wallet dan top-up via Xendit</p>

    <div class="wallet-balance-card glass-card animate-fade-in">
      <div class="balance-label">Saldo Tersedia</div>
      <div class="balance-amount">
        <span class="currency">Rp</span>
        <span class="amount">{{ formattedBalance }}</span>
      </div>
      <p class="balance-subtitle">{{ walletData.balance_rupiah }} sen</p>
    </div>

    <!-- Top-up Section -->
    <div class="glass-card animate-fade-in" style="margin-top: 1.5rem;">
      <h3>Top-up Wallet</h3>
      <p class="text-muted" style="margin-bottom: 1rem; font-size: 0.9rem;">
        Isi saldo wallet via Xendit. Minimum top-up Rp 100.000.
      </p>

      <div class="form-group">
        <label>Jumlah Top-up (Rp)</label>
        <input
          v-model.number="topupAmount"
          type="number"
          class="form-control"
          min="100000"
          step="1000"
          placeholder="100000"
        />
      </div>

      <div v-if="topupError" class="error-msg">{{ topupError }}</div>

      <button
        class="btn btn-primary"
        @click="handleTopup"
        :disabled="loadingTopup || !topupAmount || topupAmount < 100000"
        style="width: 100%; margin-top: 0.75rem;"
      >
        {{ loadingTopup ? 'Memproses...' : 'Top-up via Xendit' }}
      </button>

      <div v-if="topupInvoiceUrl" class="topup-result">
        <p style="font-size: 0.85rem; color: var(--text-secondary); margin-bottom: 0.5rem;">
          Invoice dibuat! Silakan lanjutkan pembayaran:
        </p>
        <a :href="topupInvoiceUrl" target="_blank" rel="noopener" class="btn btn-secondary" style="width: 100%; text-align: center;">
          🔗 Bayar Sekarang
        </a>
        <p style="font-size: 0.75rem; color: var(--text-muted); margin-top: 0.5rem;">
          Invoice akan terverifikasi otomatis setelah pembayaran. Halaman akan refresh dalam beberapa detik.
        </p>
      </div>
    </div>

    <!-- Transaction History -->
    <div class="glass-card animate-fade-in" style="margin-top: 1.5rem;">
      <h3>Riwayat Transaksi</h3>

      <div v-if="!walletData.transactions || walletData.transactions.length === 0" class="empty-state">
        <p>Belum ada transaksi.</p>
      </div>

      <div v-else class="transaction-list">
        <div v-for="tx in walletData.transactions" :key="tx.id" class="transaction-item">
          <div class="tx-info">
            <span class="tx-desc">{{ tx.description || tx.reference }}</span>
            <span class="tx-time">{{ formatDate(tx.time) }}</span>
          </div>
          <div :class="['tx-amount', tx.type === 'topup' ? 'positive' : 'negative']">
            {{ tx.type === 'topup' ? '+' : '-' }}Rp {{ formatRupiah(Math.abs(tx.amount)) }}
          </div>
        </div>
      </div>

      <button
        v-if="walletData.transactions && walletData.transactions.length >= 20"
        class="btn btn-secondary"
        style="width: 100%; margin-top: 1rem;"
        @click="loadMore"
        :disabled="loadingMore"
      >
        {{ loadingMore ? 'Memuat...' : 'Muat Lebih Banyak' }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { api } from '../api'

const walletData = ref<any>({ balance_rupiah: 0, transactions: [] })
const loadingWallet = ref(false)
const loadingTopup = ref(false)
const loadingMore = ref(false)
const topupAmount = ref<number | null>(null)
const topupError = ref('')
const topupInvoiceUrl = ref('')

const formattedBalance = computed(() => {
  const rupiah = walletData.value.balance_rupiah || 0
  return (rupiah ).toLocaleString('id-ID')
})

const loadWallet = async () => {
  loadingWallet.value = true
  topupInvoiceUrl.value = ''
  try {
    const data = await api.getWallet()
    if (data.success && data.data) {
      walletData.value = data.data
    } else {
      walletData.value = { balance_rupiah: 0, transactions: [] }
    }
  } catch (e: any) {
    topupError.value = e.message || 'Gagal memuat wallet'
    walletData.value = { balance_rupiah: 0, transactions: [] }
  } finally {
    loadingWallet.value = false
  }
}

const handleTopup = async () => {
  if (!topupAmount.value || topupAmount.value < 100000) {
    topupError.value = 'Minimum top-up Rp 100.000'
    return
  }
  topupError.value = ''
  loadingTopup.value = true
  try {
    const data = await api.topupWallet(topupAmount.value)
    if (data.success && data.data?.invoice_url) {
      topupInvoiceUrl.value = data.data.invoice_url
      // Auto-refresh after 5 seconds to pick up payment confirmation
      setTimeout(() => loadWallet(), 5000)
    } else if (data.success && data.data?.status === 'free') {
      // Free transaction
      loadWallet()
    } else {
      topupError.value = data.message || 'Gagal membuat invoice'
    }
  } catch (e: any) {
    topupError.value = e.message || 'Gagal memproses top-up'
  } finally {
    loadingTopup.value = false
  }
}

const formatRupiah = (rupiah: number) => {
  return (rupiah ).toLocaleString('id-ID')
}

const formatDate = (time: string) => {
  if (!time) return '-'
  return new Date(time).toLocaleDateString('id-ID', {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  })
}

const loadMore = async () => {
  loadingMore.value = true
  // Reload full wallet (backend returns last 20)
  await loadWallet()
  loadingMore.value = false
}

onMounted(() => {
  loadWallet()
})
</script>

<style scoped>
.wallet-page {
  max-width: 600px;
}

.wallet-balance-card {
  text-align: center;
  padding: 2rem;
}

.balance-label {
  font-size: 0.85rem;
  color: var(--text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  margin-bottom: 0.5rem;
}

.balance-amount {
  display: flex;
  align-items: baseline;
  justify-content: center;
  gap: 0.25rem;
}

.currency {
  font-size: 1.25rem;
  font-weight: 600;
  color: var(--text-primary);
}

.amount {
  font-size: 2.5rem;
  font-weight: 700;
  color: var(--accent-primary);
}

.balance-subtitle {
  font-size: 0.85rem;
  color: var(--text-secondary);
  margin-top: 0.25rem;
}

.error-msg {
  color: #ef4444;
  font-size: 0.85rem;
  margin-bottom: 0.5rem;
  background: rgba(239, 68, 68, 0.1);
  padding: 0.5rem;
  border-radius: 6px;
}

.topup-result {
  margin-top: 1rem;
  padding: 1rem;
  background: rgba(16, 185, 129, 0.08);
  border-radius: 8px;
  border: 1px solid rgba(16, 185, 129, 0.2);
}

.empty-state {
  text-align: center;
  padding: 2rem;
  color: var(--text-secondary);
}

.transaction-list {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.transaction-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0.75rem 1rem;
  background: var(--bg-tertiary);
  border-radius: 6px;
}

.tx-info {
  display: flex;
  flex-direction: column;
  gap: 0.15rem;
}

.tx-desc {
  font-size: 0.9rem;
  font-weight: 500;
  color: var(--text-primary);
}

.tx-time {
  font-size: 0.75rem;
  color: var(--text-secondary);
}

.tx-amount {
  font-weight: 600;
  font-size: 0.95rem;
  white-space: nowrap;
}

.tx-amount.positive {
  color: #10b981;
}

.tx-amount.negative {
  color: #ef4444;
}
</style>