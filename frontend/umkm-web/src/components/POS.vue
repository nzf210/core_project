<template>
  <div class="pos-page">
    <div class="header-actions flex items-center justify-between" style="margin-bottom: 2rem;">
      <div>
        <h2>Kasir (Point of Sale)</h2>
        <p>Catat penjualan dan proses pembayaran</p>
      </div>
    </div>

    <div class="pos-container">
      <!-- Left: Products Grid -->
      <div class="products-section glass-card">
        <h3 style="margin-bottom: 1rem;">Pilih Produk</h3>
        <div v-if="loading" class="text-center text-muted" style="padding: 2rem;">
          Memuat produk...
        </div>
        <div v-else-if="products.length === 0" class="text-center text-muted empty-state">
          Belum ada produk. Tambahkan produk di menu Katalog Produk.
        </div>
        <div v-else class="product-grid">
          <div v-for="product in products" :key="product.id" class="pos-item" @click="addToCart(product)">
            <div class="pos-item-img-container">
              <img v-if="product.photo_url" :src="product.photo_url" alt="Product Image" class="pos-item-img" />
              <div v-else class="pos-item-placeholder">No Photo</div>
            </div>
            <div class="pos-item-details">
              <h4>{{ product.name }}</h4>
              <p class="text-accent-primary">{{ formatCurrency(product.price) }}</p>
              <div style="margin-top: 0.5rem; font-size: 0.85rem;">
                <span :class="['badge', product.stock_quantity <= 0 ? 'badge-danger' : 'badge-success']">
                  Stok: {{ product.stock_quantity }}
                </span>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Right: Cart -->
      <div class="cart-section glass-card">
        <h3 style="margin-bottom: 1rem;">Keranjang Belanja</h3>
        
        <div v-if="cart.length === 0" class="empty-cart text-center text-muted">
          Keranjang kosong
        </div>
        <div v-else class="cart-items">
          <div v-for="(item, idx) in cart" :key="item.id" class="cart-item">
            <div class="cart-item-info">
              <h5 style="margin:0;">{{ item.name }}</h5>
              <div class="text-muted" style="font-size: 0.85rem;">{{ formatCurrency(item.price) }}</div>
            </div>
            <div class="cart-item-actions flex items-center gap-2">
              <button class="btn btn-sm btn-outline" @click="updateQty(idx, -1)">-</button>
              <span style="font-weight:600; width:20px; text-align:center;">{{ item.quantity }}</span>
              <button class="btn btn-sm btn-outline" @click="updateQty(idx, 1)">+</button>
            </div>
          </div>
        </div>

        <div class="cart-summary" v-if="cart.length > 0">
          <div class="flex items-center justify-between" style="font-size:1.25rem; font-weight:700; margin-bottom:1.5rem;">
            <span>Total</span>
            <span class="text-accent-primary">{{ formatCurrency(cartTotal) }}</span>
          </div>
          <button class="btn btn-primary" style="width: 100%; font-size:1.1rem; padding:1rem;" @click="showPaymentModal = true">
            Bayar
          </button>
        </div>
      </div>
    </div>

    <!-- Payment Modal -->
    <Teleport to="body">
    <div v-if="showPaymentModal" class="modal-overlay">
      <div class="modal-content animate-fade-in">
        <div v-if="!checkoutSuccess">
          <h3 style="margin-bottom: 1.5rem; text-align:center;">Pilih Metode Pembayaran</h3>
          <div class="payment-methods">
            <button :class="['payment-btn', paymentMethod === 'cash' ? 'active' : '']" @click="paymentMethod = 'cash'">
              💵 Tunai
            </button>
            <button :class="['payment-btn', paymentMethod === 'qris' ? 'active' : '']" @click="paymentMethod = 'qris'">
              📱 QRIS
            </button>
          </div>

          <div style="margin-top:2rem; text-align:center;">
            <p style="font-size:1.25rem; font-weight:600; margin-bottom:1.5rem;">Total Tagihan: {{ formatCurrency(cartTotal) }}</p>
            <div class="flex gap-4 justify-center">
              <button class="btn btn-outline" @click="showPaymentModal = false">Batal</button>
              <button class="btn btn-primary" @click="processCheckout" :disabled="loadingCheckout">
                {{ loadingCheckout ? 'Memproses...' : 'Proses Pembayaran' }}
              </button>
            </div>
          </div>
        </div>

        <div v-else class="text-center">
          <div v-if="paymentStatus === 'paid'">
            <div style="font-size: 4rem; margin-bottom: 1rem;">✅</div>
            <h3 style="margin-bottom: 1rem;">Transaksi Berhasil!</h3>
            <p style="margin-bottom: 1.5rem; color: var(--text-secondary); font-size: 0.9rem;">
              Pembayaran otomatis terverifikasi. Struk/bukti pembayaran telah dikirim ke WhatsApp kasir.
            </p>
            <button class="btn btn-primary" @click="finishTransaction">Tutup & Transaksi Baru</button>
          </div>
          
          <div v-else-if="paymentStatus === 'pending'">
            <h3 style="margin-bottom: 1rem; color: var(--accent-primary);">Menunggu Pembayaran...</h3>
            <p style="margin-bottom: 1rem;">Minta pelanggan untuk scan QR di bawah ini (Buka Xendit Checkout):</p>
            <div style="margin-bottom:1.5rem; display: flex; flex-direction: column; align-items: center; gap: 1rem;">
              <img :src="'https://api.qrserver.com/v1/create-qr-code/?size=300x300&data=' + encodeURIComponent(qrisUrl)" alt="QR Code Invoice" style="width: 250px; height: 250px; border-radius: 8px; border: 4px solid var(--accent-primary);" />
              <a :href="qrisUrl" target="_blank" class="btn btn-outline">Buka Link Pembayaran di Tab Baru</a>
            </div>
            <p style="color: var(--text-muted); font-size: 0.85rem; margin-bottom: 1.5rem;">
              Sistem akan mengecek secara otomatis. Jangan tutup halaman ini sebelum pembayaran berhasil.
            </p>
            <button class="btn btn-outline" @click="finishTransaction">Batal Transaksi</button>
          </div>
        </div>
      </div>
    </div>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { api } from '../api'
import { useModalState } from '../utils/modalState'

const { openModal, closeModal } = useModalState()


const products = ref<any[]>([])
const loading = ref(false)

const cart = ref<any[]>([])
const showPaymentModal = ref(false)
const paymentMethod = ref('cash')
const loadingCheckout = ref(false)

watch(showPaymentModal, (v) => { if (v) openModal(); else closeModal(); })
const checkoutSuccess = ref(false)
const qrisUrl = ref('')
const paymentStatus = ref('')
const transactionRef = ref('')
let pollInterval: any = null

const fetchProducts = async () => {
  loading.value = true
  try {
    const data = await api.get('/api/umkm/products')
    if (data.success && data.data) {
      products.value = data.data
    }
  } catch (error) {
    console.error("Gagal mengambil produk:", error)
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchProducts()
})

onUnmounted(() => {
  if (pollInterval) clearInterval(pollInterval)
})

const formatCurrency = (val: number) => {
  return new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR' }).format(val)
}

const addToCart = (product: any) => {
  if (product.stock_quantity <= 0) {
    return
  }

  const existing = cart.value.find(item => item.id === product.id)
  if (existing) {
    const totalQty = existing.quantity + 1
    if (totalQty > product.stock_quantity) return
    existing.quantity = totalQty
  } else {
    cart.value.push({
      id: product.id,
      name: product.name,
      price: product.price,
      quantity: 1,
    })
  }
}

const updateQty = (index: number, delta: number) => {
  const item = cart.value[index]
  const newQty = item.quantity + delta
  if (newQty <= 0) {
    cart.value.splice(index, 1)
    return
  }
  const product = products.value.find(p => p.id === item.id)
  if (product && newQty > product.stock_quantity) return
  item.quantity = newQty
}

const cartTotal = computed(() => {
  return cart.value.reduce((total, item) => total + (item.price * item.quantity), 0)
})

const processCheckout = async () => {
  loadingCheckout.value = true
  try {
    const payload = {
      payment_method: paymentMethod.value,
      total_amount: cartTotal.value,
      items: cart.value
    }
    
    const data = await api.post('/api/umkm/checkout', payload)
    if (data.success) {
      checkoutSuccess.value = true
      paymentStatus.value = data.status || 'paid'
      transactionRef.value = data.reference || ''
      if (data.qris_url) {
        qrisUrl.value = data.qris_url
      }
      
      if (paymentStatus.value === 'pending') {
        startPolling()
      }
    } else {
      alert(data.message || "Gagal memproses transaksi")
    }
  } catch (error) {
    console.error("Checkout error:", error)
    alert("Terjadi kesalahan jaringan")
  } finally {
    loadingCheckout.value = false
  }
}

const startPolling = () => {
  if (pollInterval) clearInterval(pollInterval)
  pollInterval = setInterval(async () => {
    try {
      const data = await api.get(`/api/umkm/transactions/status?reference=${transactionRef.value}`)
      if (data.success && data.status === 'paid') {
        paymentStatus.value = 'paid'
        clearInterval(pollInterval)
      }
    } catch (e) {
      console.error('Polling error', e)
    }
  }, 3000)
}

const finishTransaction = () => {
  if (pollInterval) clearInterval(pollInterval)
  cart.value = []
  showPaymentModal.value = false
  checkoutSuccess.value = false
  qrisUrl.value = ''
  paymentStatus.value = ''
  transactionRef.value = ''
  paymentMethod.value = 'cash'
}
</script>

<style scoped>
.pos-container {
  display: grid;
  grid-template-columns: 2fr 1fr;
  gap: 1.5rem;
  align-items: flex-start;
}

@media (max-width: 900px) {
  .pos-container {
    grid-template-columns: 1fr;
  }
}

.products-section {
  padding: 1.5rem;
}

.product-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
  gap: 1rem;
}

.pos-item {
  background: var(--surface-0);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  overflow: hidden;
  cursor: pointer;
  transition: transform 0.2s, box-shadow 0.2s;
}

.pos-item:hover {
  transform: translateY(-2px);
  box-shadow: var(--shadow-md);
  border-color: var(--accent-primary);
}

.pos-item-img-container {
  width: 100%;
  height: 120px;
  background: var(--surface-1);
}

.pos-item-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.pos-item-placeholder {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-secondary);
  font-size: 0.8rem;
  background: linear-gradient(135deg, var(--surface-1) 0%, #e2e8f0 100%);
}

.pos-item-details {
  padding: 0.75rem;
  text-align: center;
}
.pos-item-details h4 {
  font-size: 0.9rem;
  margin-bottom: 0.25rem;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.pos-item-details p {
  font-weight: 700;
  margin: 0;
}

.cart-section {
  padding: 1.5rem;
  position: sticky;
  top: 80px;
}

.cart-items {
  max-height: 400px;
  overflow-y: auto;
  margin-bottom: 1.5rem;
  padding-right: 0.5rem;
}

.cart-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1rem 0;
  border-bottom: 1px dashed var(--border-color);
}

.cart-item-actions {
  display: flex;
  align-items: center;
}

.btn-sm {
  padding: 0.25rem 0.5rem;
  font-size: 1rem;
  min-width: 30px;
}

.cart-summary {
  padding-top: 1rem;
  border-top: 2px solid var(--border-color);
}

.modal-content {
  width: 100%;
  max-width: 450px;
  padding: 2.5rem;
}

.payment-methods {
  display: flex;
  gap: 1rem;
  justify-content: center;
}

.payment-btn {
  flex: 1;
  padding: 1.5rem 1rem;
  border: 2px solid var(--border-color);
  background: var(--surface-0);
  border-radius: var(--radius-md);
  font-size: 1.1rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
  color: var(--text-primary);
}

.payment-btn:hover {
  border-color: var(--accent-primary);
  background: rgba(59, 130, 246, 0.05);
}

.payment-btn.active {
  border-color: var(--accent-primary);
  background: rgba(59, 130, 246, 0.1);
  color: var(--accent-primary);
}
</style>
