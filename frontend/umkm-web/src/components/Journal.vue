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
        <button class="btn btn-secondary" @click="downloadCashFlowPDF" :disabled="!cashFlowRange.from || !cashFlowRange.to" title="Download Laporan Arus Kas PDF">
          📄 PDF Arus Kas
        </button>
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
          <tr v-if="transactions.length === 0">
            <td colspan="7" style="text-align: center; color: var(--text-secondary); padding: 2rem;">
              Belum ada transaksi. Klik "Catat Transaksi" untuk mencatat jurnal pertama.
            </td>
          </tr>
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
    <div v-if="showForm" class="modal-overlay" @click.self="showForm = false">
      <div class="modal-content animate-fade-in" style="max-width: 700px; max-height: 90vh; overflow-y: auto; padding: 2rem;">
        <div class="flex items-center justify-between" style="margin-bottom: 1.5rem;">
          <h3>Catat Jurnal Baru</h3>
          <button @click="showForm = false" style="background:none;border:none;cursor:pointer;font-size:1.2rem;padding:0.25rem;">✕</button>
        </div>

        <div class="form-row-2">
          <div class="input-group">
            <label class="input-label">Tanggal</label>
            <input type="date" class="input-field" v-model="form.date" />
          </div>
          <div class="input-group">
            <label class="input-label">Referensi</label>
            <input type="text" class="input-field" v-model="form.reference" placeholder="cth: INV-001" />
          </div>
        </div>

        <div class="input-group" style="margin-bottom: 1rem;">
          <label class="input-label">Keterangan</label>
          <input type="text" class="input-field" v-model="form.description" placeholder="cth: Penjualan Produk A" />
        </div>

        <div style="margin-bottom: 0.75rem; font-weight: 600; font-size: 0.9rem; color: var(--text-secondary);">
          Ayat Jurnal (Double-Entry)
        </div>

        <div class="journal-lines">
          <div v-for="(line, idx) in form.lines" :key="idx" class="journal-line-row">
            <select class="input-field" v-model="line.account_id" style="flex: 2;">
              <option value="">Pilih Akun</option>
              <option v-for="acc in accounts" :key="acc.id" :value="acc.id">
                {{ acc.code }} — {{ acc.name }}
              </option>
            </select>
            <input type="number" class="input-field" v-model.number="line.debit" placeholder="Debit (Rp)" min="0" style="flex: 1;" @input="line.credit = 0" />
            <input type="number" class="input-field" v-model.number="line.credit" placeholder="Kredit (Rp)" min="0" style="flex: 1;" @input="line.debit = 0" />
            <button class="btn btn-secondary btn-sm" @click="removeLine(idx)" style="padding: 0.3rem 0.5rem; color: #ef4444;">✕</button>
          </div>
        </div>

        <button class="btn btn-secondary" @click="addLine" style="margin-top: 0.5rem; margin-bottom: 1rem; font-size: 0.85rem;">
          + Tambah Baris
        </button>

        <div v-if="lineError" class="error-msg" style="margin-bottom: 1rem; font-size: 0.85rem;">{{ lineError }}</div>

        <div class="flex justify-between" style="margin-top: 1.5rem; border-top: 1px solid var(--border-color); padding-top: 1rem;">
          <div class="totals">
            <span>Total Debit: <strong>{{ formatNumber(totalDebit) }}</strong></span>
            &nbsp;&nbsp;
            <span>Total Kredit: <strong>{{ formatNumber(totalCredit) }}</strong></span>
            <span v-if="totalDebit !== totalCredit" style="color: #ef4444; margin-left: 0.5rem;">⚠️ Belum平衡</span>
          </div>
          <div class="flex gap-2">
            <button class="btn btn-secondary" @click="showForm = false">Batal</button>
            <button class="btn btn-primary" @click="saveTransaction" :disabled="saving">
              {{ saving ? 'Menyimpan...' : 'Simpan Jurnal' }}
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, nextTick, computed } from 'vue'
import { api } from '../api'

const showForm = ref(false)
const ocrInput = ref<HTMLInputElement | null>(null)
const businessName = ref(localStorage.getItem('business_name') || 'Toko Kami')

const transactions = ref<any[]>([])
const accounts = ref<any[]>([])
const saving = ref(false)
const lineError = ref('')

// F021: Cash Flow PDF range
const today = new Date()
const firstOfMonth = new Date(today.getFullYear(), today.getMonth(), 1)
const cashFlowRange = ref({
  from: firstOfMonth.toISOString().split('T')[0],
  to: today.toISOString().split('T')[0],
})
const downloadingPDF = ref(false)
async function downloadCashFlowPDF() {
  if (!cashFlowRange.value.from || !cashFlowRange.value.to) {
    alert('Pilih rentang tanggal terlebih dahulu')
    return
  }
  downloadingPDF.value = true
  try {
    const url = api.cashFlowPDFUrl(cashFlowRange.value.from, cashFlowRange.value.to)
    // Trigger browser download (auth header via window.open doesn't work for
    // protected routes, so we use a temporary <a> with the token in query is
    // unsafe. Simpler: use fetch + blob)
    const res = await fetch(url, { headers: { 'X-Tenant-ID': localStorage.getItem('tenant_id') || '', 'Authorization': `Bearer ${localStorage.getItem('access_token') || ''}` } })
    if (!res.ok) throw new Error('HTTP ' + res.status)
    const blob = await res.blob()
    const blobUrl = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = blobUrl
    a.download = `arus-kas_${cashFlowRange.value.from}_${cashFlowRange.value.to}.pdf`
    document.body.appendChild(a)
    a.click()
    a.remove()
    URL.revokeObjectURL(blobUrl)
  } catch (e: any) {
    alert('Gagal download PDF: ' + (e?.message || e))
  } finally {
    downloadingPDF.value = false
  }
}

const todayDate = new Date().toISOString().split('T')[0]
const form = ref({
  date: todayDate,
  reference: '',
  description: '',
  lines: [
    { account_id: '', debit: 0, credit: 0 },
    { account_id: '', debit: 0, credit: 0 },
  ],
})

const totalDebit = computed(() => form.value.lines.reduce((s, l) => s + (l.debit || 0), 0))
const totalCredit = computed(() => form.value.lines.reduce((s, l) => s + (l.credit || 0), 0))

const formatNumber = (val: number) => new Intl.NumberFormat('id-ID').format(val)

const fetchAccounts = async () => {
  try {
    const data = await api.get('/api/umkm/accounts')
    if (data.success && data.data) accounts.value = data.data
  } catch (e) { console.error(e) }
}

const fetchTransactions = async () => {
  try {
    const tenantID = localStorage.getItem('tenant_id')
    if (!tenantID) return
    const txData = await api.get('/api/umkm/transactions')
    if (txData.success && txData.data) transactions.value = txData.data
  } catch (e) {
    console.error("Failed to fetch transactions:", e)
  }
}

onMounted(() => {
  fetchTransactions()
  fetchAccounts()
})

const formatCurrency = (val: number) => {
  return new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR' }).format(val)
}

const addLine = () => {
  form.value.lines.push({ account_id: '', debit: 0, credit: 0 })
  lineError.value = ''
}

const removeLine = (idx: number) => {
  if (form.value.lines.length > 2) form.value.lines.splice(idx, 1)
}

const saveTransaction = async () => {
  lineError.value = ''
  const lines = form.value.lines.filter(l => l.account_id && (l.debit > 0 || l.credit > 0))
  if (lines.length < 2) {
    lineError.value = 'Minimal perlu 2 baris dengan akun terisi.'
    return
  }
  if (totalDebit.value !== totalCredit.value) {
    lineError.value = `Total Debit (${formatNumber(totalDebit.value)}) harus sama dengan Kredit (${formatNumber(totalCredit.value)}).`
    return
  }
  saving.value = true
  try {
    const data = await api.post('/api/umkm/transactions', {
      date: form.value.date,
      description: form.value.description,
      reference: form.value.reference,
      lines: lines.map(l => ({ account_id: l.account_id, debit: l.debit || 0, credit: l.credit || 0 })),
    })
    if (data.success) {
      showForm.value = false
      form.value = { date: todayDate, reference: '', description: '', lines: [{ account_id: '', debit: 0, credit: 0 }, { account_id: '', debit: 0, credit: 0 }] }
      fetchTransactions()
    } else {
      lineError.value = data.message || 'Gagal menyimpan jurnal.'
    }
  } catch (e) {
    lineError.value = 'Kesalahan jaringan.'
  } finally {
    saving.value = false
  }
}

const triggerOCR = () => {
  ocrInput.value?.click()
}

const handleOCR = async (e: Event) => {
  const target = e.target as HTMLInputElement
  if (!target.files || target.files.length === 0) return
  const file = target.files[0]

  const formData = new FormData()
  formData.append('image', file)

  try {
    const data = await api.post('/api/umkm/ocr', formData, true)
    if (data.success && data.data) {
      if (data.data.description) form.value.description = data.data.description
      if (data.data.date) form.value.date = data.data.date
      showForm.value = true
      fetchAccounts()
    } else {
      alert(data.message || 'Gagal scan nota.')
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
  setTimeout(() => { window.print() }, 500)
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

.data-table tbody tr:first-child td[colspan] {
  text-align: center;
  color: var(--text-secondary);
  padding: 2rem;
}

.text-right {
  text-align: right;
}

.form-row-2 {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 1rem;
  margin-bottom: 1rem;
}

.journal-line-row {
  display: flex;
  gap: 0.5rem;
  align-items: center;
  margin-bottom: 0.5rem;
}

.totals {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.9rem;
}

.error-msg {
  color: var(--accent-error, #ef4444);
  padding: 0.5rem;
  background: rgba(239, 68, 68, 0.1);
  border-radius: 4px;
}

.modal-content {
  width: 100%;
  max-width: 700px;
  padding: 2rem;
  max-height: 90vh;
  overflow-y: auto;
}

.modal-content .input-field {
  width: 100%;
  padding: 0.75rem 1rem;
  background: var(--surface-0);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  color: var(--text-primary);
  font-size: 1rem;
  box-sizing: border-box;
  transition: border-color 0.2s, box-shadow 0.2s;
}

.modal-content .input-field:focus {
  outline: none;
  border-color: var(--accent);
  box-shadow: 0 0 0 3px rgba(99, 102, 241, 0.1);
}

.modal-content .input-label {
  display: block;
  font-weight: 500;
  margin-bottom: 0.5rem;
  font-size: 0.875rem;
  color: var(--text-primary);
}

.modal-content .input-group {
  display: flex;
  flex-direction: column;
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
