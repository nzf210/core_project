import re

with open("frontend/umkm-web/src/components/POS.vue", "r") as f:
    content = f.read()

# Replace the template section
template_old = """        <div v-else class="text-center">
          <div style="font-size: 4rem; margin-bottom: 1rem;">✅</div>
          <h3 style="margin-bottom: 1rem;">Transaksi Berhasil!</h3>
          <p style="margin-bottom: 1rem;" v-if="qrisUrl">Silakan minta pelanggan untuk scan QR Code dinamis ini:</p>
          <div v-if="qrisUrl" style="margin-bottom:1.5rem;">
            <img :src="qrisUrl" alt="QRIS Code" style="width: 250px; height: 250px; border-radius: 8px;" />
          </div>
          <p style="margin-bottom: 1.5rem; color: var(--text-secondary); font-size: 0.9rem;">
            Catatan: Untuk mencetak struk pembayaran, silakan buka menu <b>Jurnal Keuangan</b>.
          </p>
          <button class="btn btn-primary" @click="finishTransaction">Tutup & Transaksi Baru</button>
        </div>"""

template_new = """        <div v-else class="text-center">
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
            <p style="margin-bottom: 1rem;">Silakan minta pelanggan untuk scan QR Code di bawah ini:</p>
            <div style="margin-bottom:1.5rem; display: flex; justify-content: center;">
              <img :src="qrisUrl" alt="QRIS Code" style="width: 250px; height: 250px; border-radius: 8px; border: 4px solid var(--accent-primary);" />
            </div>
            <p style="color: var(--text-muted); font-size: 0.85rem; margin-bottom: 1.5rem;">
              Sistem akan mengecek secara otomatis. Jangan tutup halaman ini sebelum pembayaran berhasil.
            </p>
            <button class="btn btn-outline" @click="finishTransaction">Batal Transaksi</button>
          </div>
        </div>"""

content = content.replace(template_old, template_new)

# Add state to script
script_vars_old = """const loadingCheckout = ref(false)
const checkoutSuccess = ref(false)
const qrisUrl = ref('')"""

script_vars_new = """const loadingCheckout = ref(false)
const checkoutSuccess = ref(false)
const qrisUrl = ref('')
const paymentStatus = ref('')
const transactionRef = ref('')
let pollInterval: any = null"""

content = content.replace(script_vars_old, script_vars_new)

# Update processCheckout
process_old = """    const data = await api.post('/api/umkm/checkout', payload)
    if (data.success) {
      checkoutSuccess.value = true
      if (data.qris_url) {
        qrisUrl.value = data.qris_url
      }
    } else {"""

process_new = """    const data = await api.post('/api/umkm/checkout', payload)
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
    } else {"""

content = content.replace(process_old, process_new)

# Update finishTransaction and add polling logic
finish_old = """const finishTransaction = () => {
  cart.value = []
  showPaymentModal.value = false
  checkoutSuccess.value = false
  qrisUrl.value = ''
  paymentMethod.value = 'cash'
}"""

finish_new = """const startPolling = () => {
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
}"""

content = content.replace(finish_old, finish_new)

with open("frontend/umkm-web/src/components/POS.vue", "w") as f:
    f.write(content)

