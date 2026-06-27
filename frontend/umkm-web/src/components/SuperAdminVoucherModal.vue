<template>
  <!-- Generate Voucher Modal -->
  <Teleport to="body">
    <div v-if="showGenerateVoucherModal" class="modal-overlay" @click.self="$emit('update:showGenerateVoucherModal', false)">
      <div class="modal-card" style="max-width: 520px;">
        <h3 style="margin: 0 0 0.25rem 0;">Generate Voucher</h3>
        <p style="color: var(--text-secondary); font-size: 0.85rem; margin-bottom: 1.5rem;">
          Generate kode voucher untuk distribusi ke customer B2B.
        </p>

        <div class="form-group">
          <label>Program Name <span style="color: var(--text-secondary); font-weight: 400; font-size: 0.75rem;">(opsional, untuk grouping)</span>
            <input v-model="voucherForm.program_name" type="text" class="form-control" placeholder="cth: Program B2B Juni 2026" /></label>
        </div>

        <div class="form-group">
          <label>Tipe Voucher
            <select v-model="voucherForm.voucher_type" class="form-control">
            <option value="bonus_months">Bonus Bulan (Akses Gratis)</option>
            <option value="discount_percent">Diskon Persentase (%)</option>
            <option value="discount_fixed">Potongan Harga Tetap (Rp)</option>
          </select></label>
        </div>

        <div class="form-group" v-if="voucherForm.voucher_type !== 'bonus_months'">
          <label>Nilai Diskon <span v-if="voucherForm.voucher_type === 'discount_percent'">(%)</span><span v-else>(Rp)</span>
            <input v-model.number="voucherForm.discount_value" type="number" class="form-control" min="1" :max="voucherForm.voucher_type === 'discount_percent' ? 100 : undefined" placeholder="Nominal diskon" /></label>
        </div>

        <div class="form-group">
          <label>Paket
            <select v-model="voucherForm.plan_id" class="form-control">
            <option value="">-- Pilih Paket --</option>
            <option v-for="plan in planOptions" :key="plan.id" :value="plan.id">{{ plan.name }} — Rp {{ (plan.price_monthly/100).toLocaleString('id-ID') }}/bln</option>
          </select></label>
        </div>

        <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 0.75rem;">
          <div class="form-group">
            <label>Jumlah Voucher
              <input v-model.number="voucherForm.quantity" type="number" class="form-control" min="1" max="1000" placeholder="1-1000" /></label>
          </div>
          <div class="form-group">
            <label>Masa Aktif (hari)
              <input v-model.number="voucherForm.validity_days" type="number" class="form-control" min="1" max="3650" placeholder="cth: 30" /></label>
          </div>
        </div>

        <div class="form-group">
          <label>Max Uses per Voucher <span style="color: var(--text-secondary); font-weight: 400; font-size: 0.75rem;">(opsional)</span>
            <input v-model.number="voucherForm.max_uses" type="number" class="form-control" min="1" placeholder="default: unlimited" /></label>
        </div>

        <div v-if="voucherError" style="color: #ef4444; font-size: 0.85rem; margin-bottom: 0.75rem;">{{ voucherError }}</div>

        <!-- Result: show generated codes -->
        <div v-if="generatedVoucherCodes.length > 0" class="voucher-result">
          <div class="voucher-result-header">
            <span>{{ generatedVoucherCodes.length }} kode berhasil di-generate</span>
            <button class="btn btn-sm" style="background: var(--accent-primary); color: white; border: none; padding: 0.2rem 0.5rem; font-size: 0.7rem;" @click="$emit('download-csv')">
              📥 Download CSV
            </button>
          </div>
          <div class="voucher-codes-list">
            <div v-for="v in generatedVoucherCodes" :key="v.code" class="voucher-code-row">
              <code>{{ v.code }}</code>
              <button class="copy-btn" @click="$emit('copy', v.code)" title="Copy">📋</button>
            </div>
            <div v-if="generatedVoucherCodes.length > 20" style="text-align: center; color: var(--text-secondary); font-size: 0.8rem; padding: 0.5rem;">
              + {{ generatedVoucherCodes.length - 20 }} lagi — download CSV untuk semua
            </div>
          </div>
        </div>

        <div style="display: flex; gap: 0.75rem; justify-content: flex-end; margin-top: 1.5rem;">
          <button class="btn btn-secondary" @click="$emit('update:showGenerateVoucherModal', false)" :disabled="generatingVoucher">Tutup</button>
          <button v-if="generatedVoucherCodes.length === 0" class="btn btn-primary" @click="$emit('generate')" :disabled="generatingVoucher || !voucherForm.plan_id || !voucherForm.quantity || !voucherForm.validity_days">
            {{ generatingVoucher ? 'Generating...' : 'Generate Sekarang' }}
          </button>
          <button v-else class="btn btn-primary" @click="generatedVoucherCodes.splice(0)">
            Generate Lagi
          </button>
        </div>
      </div>
    </div>
  </Teleport>

  <!-- Voucher List Modal -->
  <Teleport to="body">
    <div v-if="showVoucherListModal" class="modal-overlay" @click.self="$emit('update:showVoucherListModal', false)">
      <div class="modal-card" style="max-width: 1100px; max-height: 90vh; overflow-y: auto; width: 90vw;">
        <div style="display: flex; align-items: center; justify-content: space-between; margin-bottom: 0.5rem;">
          <h3 style="margin: 0;">Daftar Voucher</h3>
          <span style="font-size: 0.8rem; color: var(--text-secondary);">{{ voucherList.length }} voucher</span>
        </div>
        <p style="color: var(--text-secondary); font-size: 0.85rem; margin-bottom: 1rem;">
          Semua voucher yang pernah di-generate. Voucher yang sudah digunakan tidak bisa di-redeem ulang.
        </p>

        <div style="display: flex; gap: 0.75rem; margin-bottom: 1.25rem; flex-wrap: wrap; align-items: center;">
          <select v-model="voucherListFilter.used" class="form-control" style="width: auto; min-width: 150px;" @change="$emit('fetch-list')" aria-label="Filter status voucher">
            <option value="">Semua</option>
            <option value="false">Belum Terpakai</option>
            <option value="true">Sudah Terpakai</option>
          </select>
          <select v-model="voucherListFilter.plan_id" class="form-control" style="width: auto; min-width: 150px;" @change="$emit('fetch-list')" aria-label="Filter paket">
            <option value="">Semua Paket</option>
            <option v-for="plan in planOptions" :key="plan.id" :value="plan.id">{{ plan.name }}</option>
          </select>
          <button class="btn btn-secondary" style="padding: 0.5rem 1rem; font-size: 0.85rem;" @click="$emit('fetch-list')" :disabled="loadingVoucherList">
            ↻ {{ loadingVoucherList ? 'Memuat...' : 'Refresh' }}
          </button>
        </div>

        <div v-if="loadingVoucherList" style="text-align: center; padding: 3rem; color: var(--text-secondary);">
          <div style="font-size: 1.5rem; margin-bottom: 0.5rem;">⏳</div>
          Memuat daftar voucher...
        </div>
        <div v-else-if="voucherList.length === 0" style="text-align: center; padding: 3rem; color: var(--text-secondary);">
          <div style="font-size: 2rem; margin-bottom: 0.5rem;">📋</div>
          Belum ada voucher. Generate dulu dari card "Voucher Billing".
        </div>
        <div v-else class="table-container">
          <table class="data-table">
            <thead>
              <tr>
                <th style="width: 50px;">#</th>
                <th>Kode Voucher</th>
                <th>Program</th>
                <th>Paket</th>
                <th>Status</th>
                <th>Digunakan Oleh</th>
                <th>Tanggal</th>
                <th style="width: 100px;">Aksi</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="(v, idx) in voucherList" :key="v.id">
                <td style="color: var(--text-muted); font-size: 0.8rem;">{{ idx + 1 }}</td>
                <td>
                  <div style="display: flex; align-items: center; gap: 0.5rem;">
                    <code style="font-size: 0.85rem; color: var(--accent-primary); background: rgba(99,102,241,0.1); padding: 0.25rem 0.5rem; border-radius: 4px; font-weight: 600;">{{ v.code }}</code>
                    <button
                      class="btn btn-secondary"
                      style="padding: 0.25rem 0.5rem; font-size: 0.7rem; border-radius: 4px;"
                      @click="$emit('copy', v.code)"
                      title="Copy kode voucher"
                    >
                      📋
                    </button>
                  </div>
                </td>
                <td style="font-size: 0.85rem;">{{ v.program_name || '-' }}</td>
                <td><span class="badge" :class="'badge-' + (v.target_plan || 'lite')">{{ (v.target_plan || '?').toUpperCase() }}</span></td>
                <td>
                  <span v-if="v.is_redeemed" class="badge" style="background: rgba(16,185,129,0.15); color: #10b981;">✓ Terpakai</span>
                  <span v-else class="badge" style="background: rgba(245,158,11,0.15); color: #fbbf24;">○ Unused</span>
                </td>
                <td style="font-size: 0.85rem;">{{ v.used_by || '-' }}</td>
                <td style="font-size: 0.8rem; color: var(--text-secondary);">
                  {{ v.created_at ? new Date(v.created_at).toLocaleDateString('id-ID') : '-' }}
                </td>
                <td>
                  <button
                    v-if="!v.is_redeemed"
                    class="btn btn-danger"
                    style="padding: 0.35rem 0.75rem; font-size: 0.75rem;"
                    @click="$emit('delete-voucher', v.id, v.code)"
                    :disabled="deletingVoucherId === v.id"
                  >
                    {{ deletingVoucherId === v.id ? '...' : '🗑️' }}
                  </button>
                  <span v-else style="font-size: 0.7rem; color: var(--text-muted);">—</span>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <div style="display: flex; gap: 0.75rem; justify-content: flex-end; margin-top: 1.5rem;">
          <button class="btn btn-secondary" @click="$emit('update:showVoucherListModal', false)">Tutup</button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
defineProps<{
  showGenerateVoucherModal: boolean
  showVoucherListModal: boolean
  voucherForm: any
  planOptions: any[]
  generatedVoucherCodes: any[]
  voucherList: any[]
  loadingVoucherList: boolean
  voucherListFilter: any
  generatingVoucher: boolean
  voucherError: string
  deletingVoucherId: any
}>()

defineEmits<{
  'update:showGenerateVoucherModal': [value: boolean]
  'update:showVoucherListModal': [value: boolean]
  'generate': []
  'delete-voucher': [id: string, code: string]
  'fetch-list': []
  'download-csv': []
  'copy': [text: string]
}>()
</script>

<style scoped>
.form-group {
  margin-bottom: 0.75rem;
}

.form-group label {
  display: block;
  font-size: 0.8rem;
  font-weight: 500;
  margin-bottom: 0.3rem;
  color: var(--text-primary);
}

.form-control {
  width: 100%;
  padding: 0.6rem 0.75rem;
  background: var(--bg-tertiary);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-sm);
  color: var(--text-primary);
  font-family: inherit;
  font-size: 0.875rem;
  box-sizing: border-box;
}

.form-control:focus {
  outline: none;
  border-color: var(--accent-primary);
}

.badge {
  padding: 0.2rem 0.6rem;
  border-radius: 20px;
  font-size: 0.7rem;
  font-weight: 700;
  text-transform: uppercase;
}

.badge-lite {
  background: rgba(245, 158, 11, 0.2);
  color: #fbbf24;
}

.badge-pro {
  background: rgba(59, 130, 246, 0.2);
  color: #60a5fa;
}

.badge-ultimate {
  background: rgba(168, 85, 247, 0.2);
  color: #c084fc;
}

.table-container {
  overflow-x: auto;
}

.data-table {
  width: 100%;
  border-collapse: collapse;
  text-align: left;
}

.data-table th,
.data-table td {
  padding: 1rem;
  border-bottom: 1px solid var(--border-color);
}

.data-table th {
  color: var(--text-secondary);
  font-weight: 500;
  text-transform: uppercase;
  font-size: 0.75rem;
  letter-spacing: 0.05em;
}

/* Voucher generation result */
.voucher-result {
  background: var(--bg-tertiary);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  padding: 1rem;
  margin-bottom: 1rem;
  max-height: 280px;
  overflow-y: auto;
}

.voucher-result-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 0.75rem;
  font-size: 0.85rem;
  font-weight: 600;
  color: var(--text-secondary);
}

.voucher-codes-list {
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
}

.voucher-code-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
}

.voucher-code-row code {
  font-size: 0.8rem;
  color: var(--accent-primary);
  background: var(--bg-secondary);
  padding: 0.2rem 0.5rem;
  border-radius: 4px;
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.copy-btn {
  background: transparent;
  border: 1px solid var(--border-color);
  padding: 0.2rem 0.4rem;
  border-radius: 4px;
  font-size: 0.75rem;
  cursor: pointer;
  flex-shrink: 0;
}

.copy-btn:hover {
  background: var(--bg-secondary);
}
</style>
