<template>
  <div class="journal">
    <div class="header-actions flex items-center justify-between">
      <div>
        <h2>Jurnal Umum</h2>
        <p>Pencatatan transaksi (Double-Entry Bookkeeping)</p>
      </div>
      <div class="flex gap-2">
        <button class="btn btn-primary" @click="showForm = true">
          + Catat Transaksi
        </button>
        <button class="btn btn-secondary" @click="triggerOCR">
          📸 Scan Nota
        </button>
        <input type="file" ref="ocrInput" accept="image/*" style="display:none" @change="handleOCR" />
      </div>
    </div>

    <div class="glass-card table-container">
      <table class="data-table">
        <thead>
          <tr>
            <th>Tanggal</th>
            <th>Referensi</th>
            <th>Keterangan</th>
            <th>Akun</th>
            <th class="text-right">Debit</th>
            <th class="text-right">Kredit</th>
            <th class="text-center">Aksi</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="(trx, idx) in transactions" :key="idx">
            <td>{{ trx.date }}</td>
            <td><span class="badge badge-warning">{{ trx.reference }}</span></td>
            <td>{{ trx.description }}</td>
            <td>
              <div v-for="line in trx.lines" :key="line.account_id">
                {{ line.account_name || line.account_id }}
              </div>
            </td>
            <td class="text-right">
              <div v-for="line in trx.lines" :key="'d'+line.account_id">
                {{ line.debit ? formatCurrency(line.debit) : '-' }}
              </div>
            </td>
            <td class="text-right">
              <div v-for="line in trx.lines" :key="'c'+line.account_id">
                {{ line.credit ? formatCurrency(line.credit) : '-' }}
              </div>
            </td>
            <td class="text-center">
              <button class="btn btn-secondary btn-sm" @click="printReceipt(trx)">
                🖨️ Cetak Struk
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Printable Receipt Area (Hidden from screen) -->
    <div id="printable-receipt" class="print-only">
      <div v-if="selectedTrx" class="receipt-box">
        <div class="receipt-header">
          <h2>{{ businessName }}</h2>
          <p>Struk Pembayaran / Jurnal</p>
          <div class="receipt-divider"></div>
        </div>
        
        <div class="receipt-info">
          <div><span>Tanggal:</span> <span>{{ selectedTrx.date }}</span></div>
          <div><span>Ref:</span> <span>{{ selectedTrx.reference }}</span></div>
          <div><span>Keterangan:</span> <span>{{ selectedTrx.description }}</span></div>
        </div>
        <div class="receipt-divider"></div>

        <div class="receipt-items">
          <template v-if="selectedTrx.metadata && selectedTrx.metadata.items">
            <div v-for="item in selectedTrx.metadata.items" :key="item.id" style="margin-bottom: 5px;">
              <div>{{ item.name }}</div>
              <div style="display: flex; justify-content: space-between; font-size: 0.85rem;">
                <span>{{ item.quantity }} x {{ formatCurrency(item.price) }}</span>
                <span>{{ formatCurrency(item.quantity * item.price) }}</span>
              </div>
            </div>
            <div class="receipt-divider"></div>
          </template>
          
          <div style="display: flex; justify-content: space-between; font-weight: bold; margin-bottom: 0.5rem;">
            <span>Total Transaksi</span>
            <span>{{ formatCurrency(getTrxTotal(selectedTrx)) }}</span>
          </div>
        </div>
        
        <div class="receipt-divider"></div>
        <div class="receipt-footer">
          <p>Terima kasih!</p>
          <p>Simpan struk ini sebagai bukti transaksi yang sah.</p>
        </div>
      </div>
    </div>

    <!-- Simple modal placeholder -->
    <div v-if="showForm" class="modal-overlay">
      <div class="glass-card modal-content animate-fade-in">
        <h3 style="margin-bottom: 1.5rem">Transaksi Baru</h3>
        
        <div class="input-group">
          <label class="input-label">Keterangan</label>
          <input type="text" class="input-field" placeholder="Contoh: Penjualan Produk A" />
        </div>

        <div class="flex justify-between" style="margin-top: 2rem;">
          <button class="btn btn-secondary" @click="showForm = false">Batal</button>
          <button class="btn btn-primary" @click="saveTransaction">Simpan Jurnal</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, nextTick } from 'vue'
import { api, API_BASE } from '../api'

const showForm = ref(false)
const ocrInput = ref<HTMLInputElement | null>(null)
const businessName = ref(localStorage.getItem('business_name') || 'Toko Kami')

const transactions = ref<any[]>([])

const fetchTransactions = async () => {
  try {
    const tenantID = localStorage.getItem('tenant_id')
    if (!tenantID) return

    // Fetch Transactions
    const txData = await api.get('/api/umkm/transactions')
    if (txData.success && txData.data) {
      transactions.value = txData.data
    }
  } catch (e) {
    console.error("Failed to fetch transactions:", e)
  }
}

onMounted(() => {
  fetchTransactions()
})

const formatCurrency = (val: number) => {
  return new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR' }).format(val)
}

const saveTransaction = () => {
  alert('Disimpan ke API /transactions')
  showForm.value = false
}

const triggerOCR = () => {
  ocrInput.value?.click()
}

const handleOCR = async (e: Event) => {
  const target = e.target as HTMLInputElement
  if (!target.files || target.files.length === 0) return
  const file = target.files[0]
  
  alert("Sedang dianalisis AI (OCR)...")
  const formData = new FormData()
  formData.append('image', file)
  
  try {
    const tenantID = localStorage.getItem('tenant_id') || ''
    const token = localStorage.getItem('access_token') || ''
    const res = await fetch(`${API_BASE}/api/umkm/ocr`, {
      method: 'POST',
      headers: {
        'X-Tenant-ID': tenantID,
        'Authorization': `Bearer ${token}`
      },
      body: formData
    })
    const data = await res.json()
    if (data.success) {
      alert("Nota berhasil di-scan!\n" + JSON.stringify(data.data, null, 2))
      // Mock saving directly for demo
      await api.post('/api/umkm/transactions', data.data)
      fetchTransactions()
    } else {
      alert("Gagal scan nota: " + data.message)
    }
  } catch (err) {
    alert("Error memanggil layanan OCR")
  } finally {
    if (ocrInput.value) ocrInput.value.value = ''
  }
}

// Print Receipt Logic
const selectedTrx = ref<any>(null)

const getTrxTotal = (trx: any) => {
  return trx.lines.reduce((sum: number, line: any) => sum + (line.debit || 0), 0)
}

const printReceipt = async (trx: any) => {
  selectedTrx.value = trx
  await nextTick()
  setTimeout(() => {
    window.print()
  }, 500)
}
</script>

<style scoped>
.header-actions {
  margin-bottom: 2rem;
}

.table-container {
  overflow-x: auto;
}

.data-table {
  width: 100%;
  border-collapse: collapse;
  text-align: left;
}

.data-table th, .data-table td {
  padding: 1rem;
  border-bottom: 1px solid var(--border-color);
}

.data-table th {
  color: var(--text-secondary);
  font-weight: 500;
  text-transform: uppercase;
  font-size: 0.85rem;
  letter-spacing: 0.05em;
}

.data-table tbody tr:last-child td {
  border-bottom: none;
}

.data-table tbody tr:hover {
  background-color: rgba(255, 255, 255, 0.02);
}

.text-right {
  text-align: right;
}

.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: rgba(15, 23, 42, 0.8);
  backdrop-filter: blur(4px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 50;
}

.modal-content {
  width: 100%;
  max-width: 500px;
}

.btn-sm {
  padding: 0.3rem 0.6rem;
  font-size: 0.8rem;
}

.text-center {
  text-align: center;
}

/* Print Only CSS */
.print-only {
  display: none;
}

@media print {
  /* Hide all elements by default visually */
  :global(body *) {
    visibility: hidden;
  }

  /* Make the receipt and its descendants visible */
  :global(#printable-receipt), :global(#printable-receipt *) {
    visibility: visible;
  }

  /* Position the receipt at the top left to override any layout flow */
  :global(#printable-receipt) {
    position: absolute;
    left: 0;
    top: 0;
    width: 100%;
    margin: 0;
    padding: 0;
    display: block !important;
  }

  /* Receipt styling for thermal printers (usually 58mm or 80mm) */
  .receipt-box {
    width: 300px;
    margin: 0 auto;
    font-family: monospace, 'Courier New', Courier;
    color: #000;
    background: #fff;
    padding: 10px;
  }

  .receipt-header {
    text-align: center;
    margin-bottom: 10px;
  }
  
  .receipt-header h2 {
    font-size: 1.2rem;
    margin: 0 0 5px 0;
    color: #000;
  }

  .receipt-header p {
    font-size: 0.9rem;
    margin: 0;
    color: #000;
  }

  .receipt-divider {
    border-top: 1px dashed #000;
    margin: 10px 0;
  }

  .receipt-info div {
    display: flex;
    justify-content: space-between;
    font-size: 0.9rem;
    margin-bottom: 3px;
  }

  .receipt-footer {
    text-align: center;
    font-size: 0.85rem;
    margin-top: 15px;
  }
}
</style>
