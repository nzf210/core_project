<template>
  <div class="superadmin-dashboard">
    <div style="margin-bottom: 2rem; display: flex; justify-content: space-between; align-items: center;">
      <div>
        <h2>Super Admin Dashboard</h2>
        <p class="text-muted">Kelola WhatsApp Verifier & pantau semua tenant</p>
        <a
          href="http://localhost:5678"
          target="_blank"
          rel="noopener noreferrer"
          class="btn btn-primary"
          style="padding: 0.5rem 1rem; margin-top: 0.5rem; display: inline-block;"
        >
          ⚡ Buka N8n Workflow
        </a>
      </div>
    </div>

    <div class="dashboard-grid">
      <!-- My Profile Card -->
      <div class="card glass-card">
        <div class="card-header">
          <h3>Profil Saya</h3>
        </div>
        <div class="card-body">
          <p class="text-muted" style="margin-bottom: 1rem;">
            <strong>{{ myProfile.username }}</strong> &mdash; {{ role }}
          </p>
          <p class="text-muted" style="font-size: 0.85rem;">
            {{ myProfile.phone_number || 'No phone' }}
          </p>
          <button class="btn btn-secondary" style="margin-top: 0.75rem; padding: 0.35rem 0.75rem; font-size: 0.8rem;"
            @click="showMyProfile = true">Edit Profil</button>
        </div>
      </div>

      <!-- WA Verifier Card -->
      <div class="card glass-card" style="grid-column: span 2;">
        <div class="card-header">
          <h3>WhatsApp Verifier</h3>
          <span :class="['status-badge', verifierStatus === 'connected' ? 'status-connected' : 'status-disconnected']">
            {{ verifierStatus === 'connected' ? 'Terhubung' : 'Tidak Terhubung' }}
          </span>
        </div>

        <div class="card-body">
          <div v-if="verifierStatus === 'disconnected' && !qrCode" class="verifier-actions">
            <p class="text-muted">WhatsApp Verifier belum terhubung. Hubungkan untuk mengaktifkan verifikasi OTP via
              WhatsApp.</p>
            <button class="btn btn-primary" @click="connectVerifier" :disabled="loadingQR">
              {{ loadingQR ? 'Menghubungkan...' : 'Hubungkan WhatsApp' }}
            </button>
          </div>

          <div v-if="qrCode" class="qr-section">
            <img :src="qrCode" alt="QR Code" class="qr-image" />
            <p class="text-muted" style="margin-top: 1rem;">Scan QR code ini menggunakan WhatsApp di HP Anda</p>
            <div class="qr-actions">
              <button class="btn btn-primary" @click="checkVerifierStatus" :disabled="checkingStatus">
                {{ checkingStatus ? 'Memeriksa...' : 'Cek Status' }}
              </button>
              <button class="btn"
                style="background: transparent; color: var(--text-secondary); border: 1px solid var(--border-color);"
                @click="qrCode = ''">
                Batal
              </button>
            </div>
          </div>

          <div v-if="verifierStatus === 'connected' && verifierJID" class="verifier-info">
            <div class="info-row">
              <span class="info-label">Nomor</span>
              <span class="info-value">{{ verifierJID }}</span>
            </div>
            <div class="info-row">
              <span class="info-label">Status</span>
              <span class="info-value" style="color: #10b981;">Online</span>
            </div>
            <div style="margin-top: 1.5rem;">
              <button class="btn btn-danger" @click="disconnectVerifier" :disabled="disconnecting"
                style="background: #ef4444; color: white; border: none;">
                {{ disconnecting ? 'Memutuskan...' : 'Putuskan WhatsApp' }}
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- Tenant Overview Card -->
      <div class="card glass-card">
        <div class="card-header">
          <h3>Total Tenant</h3>
        </div>
        <div class="card-body">
          <div class="stat-number">{{ tenants.length }}</div>
          <p class="text-muted">tenant terdaftar</p>
        </div>
      </div>

      <!-- Plan Distribution -->
      <div class="card glass-card">
        <div class="card-header">
          <h3>Paket</h3>
        </div>
        <div class="card-body">
          <div class="plan-stats">
            <div class="plan-stat"><span class="badge badge-lite">LITE</span> {{ planCounts.lite }}</div>
            <div class="plan-stat"><span class="badge badge-pro">PRO</span> {{ planCounts.pro }}</div>
            <div class="plan-stat"><span class="badge badge-ultimate">ULTIMATE</span> {{ planCounts.ultimate }}</div>
          </div>
        </div>
      </div>
      <!-- Voucher Billing Card -->
      <div class="card glass-card">
        <div class="card-header">
          <h3>Voucher Billing</h3>
          <button class="btn btn-sm" style="background: rgba(59,130,246,0.15); color: #60a5fa; border: 1px solid rgba(59,130,246,0.3); padding: 0.25rem 0.6rem; font-size: 0.75rem;" @click="openVoucherList">
            Lihat Daftar
          </button>
        </div>
        <div class="card-body">
          <p class="text-muted" style="margin-bottom: 1rem; font-size: 0.85rem;">
            Generate link aktivasi instan untuk B2B.
          </p>
          <button class="btn btn-primary" style="width: 100%; padding: 0.5rem;" @click="openGenerateVoucher">
            Generate Voucher
          </button>
        </div>
      </div>

    </div>

    <!-- Tenant Table -->
    <div class="card glass-card" style="margin-top: 1.5rem;">
      <div class="card-header">
        <h3>Daftar Tenant</h3>
        <div>
          <button class="btn btn-primary" style="padding: 0.35rem 0.75rem; font-size: 0.8rem; margin-right: 0.5rem;" @click="showAddTenant = true">
            Tambah Tenant Baru
          </button>
          <button class="btn" style="background: rgba(168, 85, 247, 0.15); color: #a855f7; border: 1px solid rgba(168, 85, 247, 0.3); padding: 0.35rem 0.75rem; font-size: 0.8rem; margin-right: 0.5rem;" @click="openPlanEditor">
            Kelola Paket
          </button>
          <button class="btn" style="background: rgba(59, 130, 246, 0.15); color: #3b82f6; border: 1px solid rgba(59, 130, 246, 0.3); padding: 0.35rem 0.75rem; font-size: 0.8rem; margin-right: 0.5rem;" @click="openAddonEditor">
            Kelola Add-on
          </button>
          <button class="btn btn-secondary" style="padding: 0.35rem 0.75rem; font-size: 0.8rem;" @click="fetchTenants"
            :disabled="loadingTenants">
            {{ loadingTenants ? '...' : 'Refresh' }}
          </button>
        </div>
      </div>
      <div class="table-container">
        <table class="data-table">
          <thead>
            <tr>
              <th>Nama Usaha</th>
              <th>Owner</th>
              <th>Phone</th>
              <th>Users</th>
              <th>Paket</th>
              <th>Xendit Merchant</th>
              <th>Terdaftar</th>
              <th>Aksi</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="t in tenants" :key="t.id">
              <td>
                <div style="display: flex; align-items: center; gap: 0.4rem;">
                  <strong>{{ t.name }}</strong>
                  <button class="btn-edit" style="padding: 0.15rem 0.3rem; opacity: 0.6;" @click="copyToClipboard(t.id)" title="Copy Tenant ID">
                    <svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                      <rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/>
                    </svg>
                  </button>
                </div>
              </td>
              <td>{{ t.owner_username || '-' }}</td>
              <td>{{ t.owner_phone || '-' }}</td>
              <td>{{ t.user_count ?? 0 }}</td>
              <td><span :class="['badge', 'badge-' + t.plan]">{{ t.plan?.toUpperCase() }}</span></td>
              <td>
                <code v-if="t.xendit_merchant_id" style="font-size: 0.75rem; color: var(--accent-primary); background: rgba(99,102,241,0.1); padding: 0.15rem 0.4rem; border-radius: 4px;">{{ t.xendit_merchant_id }}</code>
                <span v-else style="font-size: 0.75rem; color: var(--text-muted);">— SaaS</span>
              </td>
              <td>{{ t.created_at ? new Date(t.created_at).toLocaleDateString('id-ID') : '-' }}</td>
              <td>
                <button class="btn-edit" @click="openEditProfile(t)" title="Edit profil tenant">
                  <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none"
                    stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <path d="M17 3a2.828 2.828 0 1 1 4 4L7.5 20.5 2 22l1.5-5.5L17 3z" />
                  </svg>
                </button>
                <button v-if="!isMyOwnTenant(t)" class="btn-delete" @click="confirmDelete(t)" title="Hapus tenant ini">
                  <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none"
                    stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <polyline points="3 6 5 6 21 6" />
                    <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" />
                  </svg>
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Modal Tambah Tenant Baru -->
    <div v-if="showAddTenant" class="modal-overlay" @click.self="showAddTenant = false">
      <div class="modal-card" style="max-width: 450px;">
        <h3 style="margin: 0 0 1.25rem 0;">Tambah Tenant Baru</h3>
        
        <div class="form-group">
          <label>Nama Usaha (Tenant)</label>
          <input v-model="formData.name" type="text" class="form-control" placeholder="cth: Toko Sembako Budi" />
        </div>

        <div class="form-group">
          <label>Username Pemilik</label>
          <input v-model="formData.username" @input="formData.username = formData.username.replace(/ /g, '_').toLowerCase()" type="text" class="form-control" placeholder="cth: budi_sembako" />
        </div>

        <div class="form-group">
          <label>Email Pemilik</label>
          <input v-model="formData.email" type="email" class="form-control" placeholder="cth: budi@contoh.com" />
        </div>

        <div class="form-group">
          <label>Nomor WhatsApp (Aktif)</label>
          <input v-model="formData.phone_number" type="text" class="form-control" placeholder="cth: 081234567890" />
        </div>

        <div class="form-group">
          <label>Subdomain (Opsional)</label>
          <input v-model="formData.subdomain" type="text" class="form-control" placeholder="cth: sembako.saas.com" />
        </div>

        <div class="form-group">
          <label>Custom Domain (Opsional)</label>
          <input v-model="formData.custom_domain" type="text" class="form-control" placeholder="cth: www.tokosembako.com" />
        </div>

        <div class="form-group">
          <label>Pilih Paket Langganan</label>
          <select v-model="formData.plan" class="form-control">
            <option v-for="plan in planOptions" :key="plan.id" :value="plan.id">{{ plan.name }} (Rp {{ (plan.price_monthly/100).toLocaleString('id-ID') }})</option>
          </select>
        </div>

        <div style="display: flex; gap: 0.75rem; justify-content: flex-end; margin-top: 1.5rem;">
          <button class="btn btn-secondary" @click="closeAddModal">Batal</button>
          <button class="btn btn-primary" @click="saveNewTenant">Simpan</button>
        </div>
      </div>
    </div>

    <!-- My Profile Modal -->
    <div v-if="showMyProfile" class="modal-overlay" @click.self="showMyProfile = false">
      <div class="modal-card" style="max-width: 440px;">
        <h3 style="margin: 0 0 1.25rem 0;">Edit Profil Saya</h3>

        <div class="form-group"><label>Username</label><input v-model="myProfile.username" class="form-control" /></div>
        <div class="form-group"><label>Nomor HP</label><input v-model="myProfile.phone_number" class="form-control"
            placeholder="0812..." /></div>
        <div class="form-group"><label>Password Lama</label><input v-model="myProfile.old_password" type="password"
            class="form-control" placeholder="(untuk ganti password)" /></div>
        <div class="form-group"><label>Password Baru</label><input v-model="myProfile.new_password" type="password"
            class="form-control" placeholder="(kosongkan jika tidak diganti)" /></div>

        <div style="display: flex; gap: 0.75rem; justify-content: flex-end; margin-top: 1.5rem;">
          <button class="btn btn-secondary" @click="showMyProfile = false" :disabled="savingMyProfile">Batal</button>
          <button class="btn btn-primary" @click="saveMyProfile" :disabled="savingMyProfile">
            {{ savingMyProfile ? 'Menyimpan...' : 'Simpan' }}
          </button>
        </div>
        <div v-if="myProfileError" style="margin-top: 0.75rem; color: #ef4444; font-size: 0.85rem;">
          {{ myProfileError }}</div>
      </div>
    </div>

    <!-- Edit Profile Modal -->
    <div v-if="editTarget" class="modal-overlay" @click.self="editTarget = null">
      <div class="modal-card" style="max-width: 540px; max-height: 85vh; overflow-y: auto;">
        <h3 style="margin: 0 0 0.25rem 0;">Edit Profil Tenant</h3>
        <p style="color: var(--text-secondary); font-size: 0.85rem; margin-bottom: 1.25rem;">
          {{ editTarget.name }} ({{ editTarget.owner_username || 'no owner' }})
        </p>

        <div class="logo-upload-section">
          <div class="logo-preview">
            <img v-if="editLogoPreview" :src="editLogoPreview" alt="Logo preview" />
            <img v-else-if="editForm.logo_url" :src="editForm.logo_url.startsWith('http') ? editForm.logo_url : API_BASE + editForm.logo_url + '?t=' + Date.now()"
              alt="Current logo" />
            <div v-else class="logo-placeholder">No Logo</div>
          </div>
          <label class="file-input-label">
            <input type="file" accept="image/png,image/jpeg,image/webp" @change="onLogoFileChange"
              style="display:none" />
            <span class="btn btn-secondary" style="padding: 0.35rem 0.75rem; font-size: 0.8rem; cursor: pointer;">Pilih
              Logo</span>
          </label>
          <button v-if="editLogoFile" class="btn"
            style="background: transparent; color: var(--accent-primary); border: 1px solid var(--border-color); padding: 0.35rem 0.75rem; font-size: 0.8rem;"
            @click="uploadLogo" :disabled="uploadingLogo">
            {{ uploadingLogo ? 'Uploading...' : 'Upload Logo' }}
          </button>
        </div>

        <div class="form-group"><label>Nama Toko</label><input v-model="editForm.business_name" class="form-control"
            placeholder="Nama usaha/toko" /></div>
        <div class="form-group"><label>Nama Usaha (Tenant)</label><input v-model="editForm.name" class="form-control" />
        </div>
        <div class="form-group"><label>Subdomain</label><input v-model="editForm.subdomain" class="form-control" placeholder="opsional" /></div>
        <div class="form-group"><label>Custom Domain</label><input v-model="editForm.custom_domain" class="form-control" placeholder="opsional" /></div>
        <div class="form-group"><label>Xendit Merchant ID <span style="color: var(--text-secondary); font-weight: 400; font-size: 0.75rem;">(kosongkan jika pakai SaaS pool)</span></label>
          <input v-model="editForm.xendit_merchant_id" class="form-control" placeholder="opsional — untuk tenant B2B dengan akun Xendit sendiri" />
        </div>
        <div class="form-group"><label>Nomor WA Toko (CS)</label><input v-model="editForm.wa_number" class="form-control"
            placeholder="0812..." /></div>
        <div class="form-group"><label>Nomor WA Owner (Login)</label><input v-model="editForm.owner_phone" class="form-control"
            placeholder="0812..." /></div>
        <div class="form-group"><label>Alamat Usaha</label><textarea v-model="editForm.business_address"
            class="form-control" rows="2" placeholder="Alamat lengkap" /></div>
        <div class="form-group"><label>Jenis Usaha</label>
          <select v-model="editForm.business_type" class="form-control">
            <option v-for="bt in businessTypes" :key="bt.id" :value="bt.id">{{ bt.name }}</option>
          </select>
        </div>
        <div class="form-group"><label>Paket</label>
          <select v-model="editForm.plan" class="form-control">
            <option v-for="plan in planOptions" :key="plan.id" :value="plan.id">{{ plan.name }}</option>
          </select>
        </div>
        <div class="form-group"><label>Reset Password Owner <span
              style="color: var(--text-secondary); font-weight: 400; font-size: 0.8rem;">(kosongkan jika tidak ingin
              diubah)</span></label>
          <input v-model="editForm.new_password" type="password" class="form-control" placeholder="Password baru" />
        </div>

        <div style="display: flex; gap: 0.75rem; justify-content: flex-end; margin-top: 1.5rem;">
          <button class="btn btn-secondary" @click="editTarget = null" :disabled="savingProfile">Batal</button>
          <button class="btn btn-primary" @click="saveProfile" :disabled="savingProfile">
            {{ savingProfile ? 'Menyimpan...' : 'Simpan' }}
          </button>
        </div>
        <div v-if="profileError" style="margin-top: 0.75rem; color: #ef4444; font-size: 0.85rem;">{{ profileError }}
        </div>
      </div>
    </div>

    <!-- Modal Delete Confirmation -->
    <div v-if="deleteTarget" class="modal-overlay" @click.self="deleteTarget = null">
      <div class="modal-card">
        <h3 style="margin: 0 0 0.75rem 0;">Hapus Tenant?</h3>
        <p style="margin-bottom: 0.5rem;">Anda akan menghapus:</p>
        <p style="margin-bottom: 1.5rem; font-weight: 600;">{{ deleteTarget.name }} ({{ deleteTarget.owner_username ||
          'no owner' }})</p>
        <p style="margin-bottom: 1.5rem; color: #ef4444; font-size: 0.85rem;">
          ⚠️ Semua data: users, produk, jurnal, transaksi, dan data tenant akan dihapus permanen.
        </p>
        <div style="display: flex; gap: 0.75rem; justify-content: flex-end;">
          <button class="btn btn-secondary" @click="deleteTarget = null" :disabled="deleting">Batal</button>
          <button class="btn" style="background: #ef4444; color: white; border: none;" @click="executeDelete"
            :disabled="deleting">
            {{ deleting ? 'Menghapus...' : 'Hapus Permanen' }}
          </button>
        </div>
        <div v-if="deleteError" style="margin-top: 1rem; color: #ef4444; font-size: 0.85rem;">{{ deleteError }}</div>
      </div>
    </div>

    <!-- Modal Generate Voucher -->
    <div v-if="showGenerateVoucherModal" class="modal-overlay" @click.self="showGenerateVoucherModal = false">
      <div class="modal-card" style="max-width: 520px;">
        <h3 style="margin: 0 0 0.25rem 0;">Generate Voucher</h3>
        <p style="color: var(--text-secondary); font-size: 0.85rem; margin-bottom: 1.5rem;">
          Generate kode voucher untuk distribusi ke customer B2B.
        </p>

        <div class="form-group">
          <label>Program Name <span style="color: var(--text-secondary); font-weight: 400; font-size: 0.75rem;">(opsional, untuk grouping)</span></label>
          <input v-model="voucherForm.program_name" type="text" class="form-control" placeholder="cth: Program B2B Juni 2026" />
        </div>

        <div class="form-group">
          <label>Tipe Voucher</label>
          <select v-model="voucherForm.voucher_type" class="form-control">
            <option value="bonus_months">Bonus Bulan (Akses Gratis)</option>
            <option value="discount_percent">Diskon Persentase (%)</option>
            <option value="discount_fixed">Potongan Harga Tetap (Rp)</option>
          </select>
        </div>

        <div class="form-group" v-if="voucherForm.voucher_type !== 'bonus_months'">
          <label>Nilai Diskon <span v-if="voucherForm.voucher_type === 'discount_percent'">(%)</span><span v-else>(Rp)</span></label>
          <input v-model.number="voucherForm.discount_value" type="number" class="form-control" min="1" :max="voucherForm.voucher_type === 'discount_percent' ? 100 : undefined" placeholder="Nominal diskon" />
        </div>

        <div class="form-group">
          <label>Paket</label>
          <select v-model="voucherForm.plan_id" class="form-control">
            <option value="">-- Pilih Paket --</option>
            <option v-for="plan in planOptions" :key="plan.id" :value="plan.id">{{ plan.name }} — Rp {{ (plan.price_monthly/100).toLocaleString('id-ID') }}/bln</option>
          </select>
        </div>

        <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 0.75rem;">
          <div class="form-group">
            <label>Jumlah Voucher</label>
            <input v-model.number="voucherForm.quantity" type="number" class="form-control" min="1" max="1000" placeholder="1-1000" />
          </div>
          <div class="form-group">
            <label>Masa Aktif (hari)</label>
            <input v-model.number="voucherForm.validity_days" type="number" class="form-control" min="1" max="3650" placeholder="cth: 30" />
          </div>
        </div>

        <div class="form-group">
          <label>Max Uses per Voucher <span style="color: var(--text-secondary); font-weight: 400; font-size: 0.75rem;">(opsional)</span></label>
          <input v-model.number="voucherForm.max_uses" type="number" class="form-control" min="1" placeholder="default: unlimited" />
        </div>

        <div v-if="voucherError" style="color: #ef4444; font-size: 0.85rem; margin-bottom: 0.75rem;">{{ voucherError }}</div>

        <!-- Result: show generated codes -->
        <div v-if="generatedVoucherCodes.length > 0" class="voucher-result">
          <div class="voucher-result-header">
            <span>{{ generatedVoucherCodes.length }} kode berhasil di-generate</span>
            <button class="btn btn-sm" style="background: var(--accent-primary); color: white; border: none; padding: 0.2rem 0.5rem; font-size: 0.7rem;" @click="downloadVoucherCSV">
              📥 Download CSV
            </button>
          </div>
          <div class="voucher-codes-list">
            <div v-for="v in generatedVoucherCodes" :key="v.code" class="voucher-code-row">
              <code>{{ v.code }}</code>
              <button class="copy-btn" @click="copyText(v.code)" title="Copy">📋</button>
            </div>
            <div v-if="generatedVoucherCodes.length > 20" style="text-align: center; color: var(--text-secondary); font-size: 0.8rem; padding: 0.5rem;">
              + {{ generatedVoucherCodes.length - 20 }} lagi — download CSV untuk semua
            </div>
          </div>
        </div>

        <div style="display: flex; gap: 0.75rem; justify-content: flex-end; margin-top: 1.5rem;">
          <button class="btn btn-secondary" @click="showGenerateVoucherModal = false" :disabled="generatingVoucher">Tutup</button>
          <button v-if="generatedVoucherCodes.length === 0" class="btn btn-primary" @click="executeGenerateVoucher" :disabled="generatingVoucher || !voucherForm.plan_id || !voucherForm.quantity || !voucherForm.validity_days">
            {{ generatingVoucher ? 'Generating...' : 'Generate Sekarang' }}
          </button>
          <button v-else class="btn btn-primary" @click="generatedVoucherCodes = []">
            Generate Lagi
          </button>
        </div>
      </div>
    </div>

    <!-- Modal Voucher List -->
    <div v-if="showVoucherListModal" class="modal-overlay" @click.self="showVoucherListModal = false">
      <div class="modal-card" style="max-width: 1100px; max-height: 90vh; overflow-y: auto; width: 90vw;">
        <div style="display: flex; align-items: center; justify-content: space-between; margin-bottom: 0.5rem;">
          <h3 style="margin: 0;">Daftar Voucher</h3>
          <span style="font-size: 0.8rem; color: var(--text-secondary);">{{ voucherList.length }} voucher</span>
        </div>
        <p style="color: var(--text-secondary); font-size: 0.85rem; margin-bottom: 1rem;">
          Semua voucher yang pernah di-generate. Voucher yang sudah digunakan tidak bisa di-redeem ulang.
        </p>

        <div style="display: flex; gap: 0.75rem; margin-bottom: 1.25rem; flex-wrap: wrap; align-items: center;">
          <select v-model="voucherListFilter.used" class="form-control" style="width: auto; min-width: 150px;" @change="fetchVoucherList">
            <option value="">Semua</option>
            <option value="false">Belum Terpakai</option>
            <option value="true">Sudah Terpakai</option>
          </select>
          <select v-model="voucherListFilter.plan_id" class="form-control" style="width: auto; min-width: 150px;" @change="fetchVoucherList">
            <option value="">Semua Paket</option>
            <option v-for="plan in planOptions" :key="plan.id" :value="plan.id">{{ plan.name }}</option>
          </select>
          <button class="btn btn-secondary" style="padding: 0.5rem 1rem; font-size: 0.85rem;" @click="fetchVoucherList" :disabled="loadingVoucherList">
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
                      @click="copyToClipboard(v.code)"
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
                    @click="deleteVoucher(v.id, v.code)"
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
          <button class="btn btn-secondary" @click="showVoucherListModal = false">Tutup</button>
        </div>
      </div>
    </div>

    <!-- Modal Kelola Add-on -->
    <div v-if="showAddonEditor" class="modal-overlay" @click.self="showAddonEditor = false">
      <div class="modal-card" style="max-width: 600px;">
        <h3 style="margin: 0 0 0.25rem 0;">Kelola Harga Add-on</h3>
        <p style="color: var(--text-secondary); font-size: 0.85rem; margin-bottom: 1.5rem;">
          Ubah harga fitur add-on untuk marketplace wallet.
        </p>

        <div v-if="loadingAddons" style="text-align: center; padding: 2rem; color: var(--text-secondary);">
          Memuat data add-on...
        </div>
        <div v-else style="max-height: 60vh; overflow-y: auto; padding-right: 0.5rem;">
          <div v-for="addon in addonOptions" :key="addon.addon_key" class="form-group" style="background: var(--surface-0); border: 1px solid var(--border-color); padding: 1rem; border-radius: var(--radius-sm); margin-bottom: 1rem;">
            <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 0.75rem;">
              <div style="font-weight: 600; text-transform: uppercase;">{{ addon.addon_key.replace(/_/g, ' ') }}</div>
              <label style="display: flex; align-items: center; gap: 0.5rem; margin: 0; font-size: 0.85rem; font-weight: 400;">
                <input type="checkbox" v-model="addon.is_active" /> Aktif
              </label>
            </div>

            <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 0.75rem;">
              <div>
                <label style="font-size: 0.75rem; margin-bottom: 0.25rem; display: block;">Harga (Rp)</label>
                <div style="position: relative;">
                  <span style="position: absolute; left: 0.75rem; top: 50%; transform: translateY(-50%); color: var(--text-secondary); font-size: 0.85rem;">Rp</span>
                  <input v-model.number="addon.price" type="number" class="form-control" style="padding-left: 2rem;" />
                </div>
              </div>
              <div>
                <label style="font-size: 0.75rem; margin-bottom: 0.25rem; display: block;">Unit (mis: bulan, trx)</label>
                <input v-model="addon.unit" type="text" class="form-control" />
              </div>
            </div>
            <div style="margin-top: 0.75rem;">
              <label style="font-size: 0.75rem; margin-bottom: 0.25rem; display: block;">Deskripsi</label>
              <input v-model="addon.description" type="text" class="form-control" />
            </div>
          </div>

          <div v-if="addonSaveMsg" :style="{ color: addonSaveMsg.includes('gagal') ? '#ef4444' : '#10b981', fontSize: '0.85rem', marginBottom: '1rem', textAlign: 'center' }">
            {{ addonSaveMsg }}
          </div>
        </div>

        <div style="display: flex; justify-content: flex-end; gap: 0.75rem; margin-top: 1.5rem;">
          <button class="btn btn-secondary" @click="showAddonEditor = false">Tutup</button>
          <button class="btn btn-primary" @click="saveAddons" :disabled="loadingAddons || savingAddons">
            {{ savingAddons ? 'Menyimpan...' : 'Simpan Perubahan' }}
          </button>
        </div>
      </div>
    </div>

    <!-- Modal Paket Langganan (Edit Harga) -->
    <div v-if="showPlanEditor" class="modal-overlay" @click.self="showPlanEditor = false">
      <div class="modal-card" style="max-width: 520px;">
        <h3 style="margin: 0 0 0.25rem 0;">Kelola Paket Langganan</h3>
        <p style="color: var(--text-secondary); font-size: 0.85rem; margin-bottom: 1.5rem;">
          Ubah harga paket SaaS untuk semua tenant.
        </p>

        <div v-if="loadingPlans" style="text-align: center; padding: 2rem; color: var(--text-secondary);">
          Memuat...
        </div>

        <div v-else>
          <div v-for="plan in editablePlans" :key="plan.id" class="plan-editor-row">
            <div class="plan-editor-header">
              <div>
                <span class="badge" :class="['badge-' + plan.id]">{{ plan.name.toUpperCase() }}</span>
              </div>
              <div style="display: flex; gap: 0.5rem; align-items: center;">
                <label style="font-size: 0.75rem; color: var(--text-secondary);">Aktif</label>
                <label class="toggle-switch">
                  <input type="checkbox" v-model="plan.is_active" />
                  <span class="toggle-slider"></span>
                </label>
              </div>
            </div>
            <div class="plan-editor-fields">
              <div class="form-group">
                <label>Harga Bulanan (Rp)</label>
                <div style="display: flex; align-items: center; gap: 0.25rem;">
                  <span style="font-size: 0.85rem; color: var(--text-secondary);">Rp</span>
                  <input v-model.number="plan.price_monthly_display" type="number" class="form-control" min="0"
                    step="1000" style="width: 100%;" @input="syncPlanPrice(plan, 'monthly')" />
                </div>
                <small style="color: var(--text-secondary); font-size: 0.7rem;">
                  Dalam sen: {{ plan.price_monthly?.toLocaleString() }} sen
                </small>
              </div>
              <div class="form-group">
                <label>Harga Tahunan (Rp)</label>
                <div style="display: flex; align-items: center; gap: 0.25rem;">
                  <span style="font-size: 0.85rem; color: var(--text-secondary);">Rp</span>
                  <input v-model.number="plan.price_yearly_display" type="number" class="form-control" min="0"
                    step="1000" style="width: 100%;" @input="syncPlanPrice(plan, 'yearly')" />
                </div>
                <small style="color: var(--text-secondary); font-size: 0.7rem;">
                  Dalam sen: {{ plan.price_yearly?.toLocaleString() }} sen
                </small>
              </div>
            </div>
          </div>
        </div>

        <div style="display: flex; gap: 0.75rem; justify-content: flex-end; margin-top: 1.5rem;">
          <button class="btn btn-secondary" @click="showPlanEditor = false" :disabled="savingPlans">Tutup</button>
          <button class="btn btn-primary" @click="savePlanPrices" :disabled="savingPlans">
            {{ savingPlans ? 'Menyimpan...' : 'Simpan Semua' }}
          </button>
        </div>
        <div v-if="planError" style="margin-top: 0.75rem; color: #ef4444; font-size: 0.85rem;">{{ planError }}</div>
      </div>
    </div>

    <!-- Toast -->
    <Teleport to="body">
      <div v-if="toast.visible" :class="['toast-notification', `toast-${toast.type}`]" :style="{ top: toastTop + 'px' }">
        {{ toast.message }}
      </div>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { superadminApi } from '../superadminApi'
import { api, API_BASE } from '../api'
import { useModalState } from '../utils/modalState'

const { openModal, closeModal } = useModalState()

const verifierStatus = ref<'connected' | 'disconnected'>('disconnected')
const verifierJID = ref('')
const qrCode = ref('')
const loadingQR = ref(false)
const checkingStatus = ref(false)
const disconnecting = ref(false)
const tenants = ref<any[]>([])
const deleting = ref(false)
const deleteTarget = ref<any>(null)
const deleteError = ref('')
const loadingTenants = ref(false)

const showAddTenant = ref(false)
const formData = ref({
  name: '',
  username: '',
  password: 'Password123',
  email: '',
  phone_number: '',
  role: 'owner',
  plan: 'lite',
  subdomain: '',
  custom_domain: ''
})

const editTarget = ref<any>(null)
const editForm = ref({
  tenant_id: '',
  name: '',
  business_name: '',
  wa_number: '',
  owner_phone: '',
  business_address: '',
  business_type: '',
  plan: 'lite',
  new_password: '',
  logo_url: '',
  subdomain: '',
  custom_domain: '',
  xendit_merchant_id: ''
})
const editLogoFile = ref<File | null>(null)
const editLogoPreview = ref('')
const uploadingLogo = ref(false)
const savingProfile = ref(false)
const profileError = ref('')

// Track all modals for body blur
watch(showAddTenant, (v) => { if (v) openModal(); else closeModal(); });
watch(editTarget, (v) => { if (v) openModal(); else closeModal(); });
watch(deleteTarget, (v) => { if (v) openModal(); else closeModal(); });

const showGenerateVoucherModal = ref(false)
const showVoucherListModal = ref(false)
const showPlanEditor = ref(false)
const showAddonEditor = ref(false)

watch(showGenerateVoucherModal, (v) => { if (v) openModal(); else closeModal(); });
watch(showVoucherListModal, (v) => { if (v) openModal(); else closeModal(); });
watch(showPlanEditor, (v) => { if (v) openModal(); else closeModal(); });
watch(showAddonEditor, (v) => { if (v) openModal(); else closeModal(); });

const businessTypes = [
  { id: 'umum', name: 'Umum / General' },
  { id: 'warung', name: 'Warung / Toko Kelontong' },
  { id: 'laundry', name: 'Laundry' },
  { id: 'industri_kreatif', name: 'Industri Kreatif' },
  { id: 'toko_online', name: 'Toko Online / E-Commerce' },
  { id: 'restoran', name: 'Restoran / F&B' },
  { id: 'jasa', name: 'Jasa / Service' },
]

const toast = ref({ visible: false, message: '', type: 'success' })
const toastTop = ref(0)
const showToast = (message: string, type: 'success' | 'error' = 'success') => {
  toast.value = { visible: true, message, type }
  toastTop.value = window.scrollY + 16
  setTimeout(() => { toast.value.visible = false }, 3000)
}

// Self-profile (super admin's own profile)
const showMyProfile = ref(false)
const myProfile = ref({ username: '', phone_number: '', old_password: '', new_password: '' })
const savingMyProfile = ref(false)
const role = computed(() => localStorage.getItem('role') || 'Super Admin')
const myProfileError = ref('')

watch(showMyProfile, (v) => { if (v) openModal(); else closeModal(); })

const loadMyProfile = async () => {
  try {
    const data = await api.get('/api/profile')
    if (data.success && data.data) {
      myProfile.value = {
        username: data.data.username || '',
        phone_number: data.data.phone_number || '',
        old_password: '',
        new_password: '',
      }
    }
  } catch (e) {
    console.error('Failed to load my profile', e)
  }
}

const saveMyProfile = async () => {
  savingMyProfile.value = true
  myProfileError.value = ''
  try {
    const payload: any = {}
    if (myProfile.value.username) payload.username = myProfile.value.username
    if (myProfile.value.phone_number) payload.phone_number = myProfile.value.phone_number
    if (myProfile.value.new_password) {
      if (!myProfile.value.old_password) {
        myProfileError.value = 'Password lama harus diisi untuk mengganti password'
        savingMyProfile.value = false
        return
      }
      payload.old_password = myProfile.value.old_password
      payload.new_password = myProfile.value.new_password
    }
    const data = await api.put('/api/profile', payload)
    if (data.success) {
      showToast('Profil berhasil disimpan')
      myProfile.value.old_password = ''
      myProfile.value.new_password = ''
      showMyProfile.value = false
    } else {
      myProfileError.value = data.message || 'Gagal menyimpan'
    }
  } catch (e) {
    myProfileError.value = 'Kesalahan jaringan'
  } finally {
    savingMyProfile.value = false
  }
}

const planCounts = computed(() => {
  const counts = { free: 0, lite: 0, pro: 0, ultimate: 0 }
  tenants.value.forEach((t: any) => {
    const plan = t.plan as keyof typeof counts
    if (counts[plan] !== undefined) {
      counts[plan]++
    }
  })
  return counts
})

const fetchTenants = async () => {
  loadingTenants.value = true
  try {
    const data = await superadminApi.getTenants()
    if (data.success && data.data) {
      tenants.value = data.data
    }
  } catch (e) {
    console.error('Failed to fetch tenants', e)
  } finally {
    loadingTenants.value = false
  }
}

const confirmDelete = (tenant: any) => {
  deleteTarget.value = tenant
  deleteError.value = ''
}

// Get current superadmin's own tenant ID (stored after login)
const myTenantId = computed(() => localStorage.getItem('tenant_id') || '')

// Check if a tenant is the superadmin's own tenant
const isMyOwnTenant = (tenant: any) => tenant.id === myTenantId.value

const executeDelete = async () => {
  if (!deleteTarget.value) return
  deleting.value = true
  deleteError.value = ''

  try {
    const data = await superadminApi.deleteTenant(deleteTarget.value.id)
    if (data.success) {
      tenants.value = tenants.value.filter((t: any) => t.id !== deleteTarget.value.id)
      showToast('Tenant berhasil dihapus', 'success')
      deleteTarget.value = null
    } else {
      deleteError.value = data.message || 'Gagal menghapus tenant'
    }
  } catch (e) {
    deleteError.value = 'Kesalahan jaringan saat menghapus tenant'
  } finally {
    deleting.value = false
  }
}

const checkVerifierStatus = async () => {
  checkingStatus.value = true
  try {
    const data = await superadminApi.getVerifierStatus()
    if (data.success && data.data) {
      verifierStatus.value = data.data.status === 'connected' ? 'connected' : 'disconnected'
      verifierJID.value = data.data.jid || ''
      if (verifierStatus.value === 'connected') {
        qrCode.value = ''
        showToast('WhatsApp Verifier berhasil terhubung!', 'success')
      }
    }
  } catch (e) {
    showToast('Gagal memeriksa status verifier', 'error')
  } finally {
    checkingStatus.value = false
  }
}

const connectVerifier = async () => {
  loadingQR.value = true
  try {
    const data = await superadminApi.getVerifierQR()
    if (data.success && data.data) {
      if (data.data.status === 'qr') {
        qrCode.value = data.data.qr_code
      } else if (data.data.status === 'connected') {
        verifierStatus.value = 'connected'
        verifierJID.value = data.data.jid || ''
        showToast('WhatsApp Verifier sudah terhubung!', 'success')
      }
    } else {
      showToast(data.message || 'Gagal mendapatkan QR code', 'error')
    }
  } catch (e) {
    showToast('Gagal menghubungkan verifier', 'error')
  } finally {
    loadingQR.value = false
  }
}

const disconnectVerifier = async () => {
  disconnecting.value = true
  try {
    const data = await superadminApi.disconnectVerifier()
    if (data.success) {
      verifierStatus.value = 'disconnected'
      verifierJID.value = ''
      qrCode.value = ''
      showToast('WhatsApp Verifier telah diputuskan', 'success')
    } else {
      showToast(data.message || 'Gagal memutuskan verifier', 'error')
    }
  } catch (e) {
    showToast('Gagal memutuskan verifier', 'error')
  } finally {
    disconnecting.value = false
  }
}

const openEditProfile = async (tenant: any) => {
  editTarget.value = tenant
  profileError.value = ''
  editLogoFile.value = null
  editLogoPreview.value = ''
  editForm.value.new_password = ''
  editForm.value.logo_url = ''

  try {
    const data = await superadminApi.getTenantProfile(tenant.id)
    if (data.success && data.data) {
      const p = data.data
      editForm.value.name = p.name || ''
      editForm.value.business_name = p.business_name || ''
      editForm.value.wa_number = p.wa_number || ''
      editForm.value.business_address = p.business_address || ''
      editForm.value.business_type = p.business_type || 'umum'
      editForm.value.plan = p.plan || 'lite'
      editForm.value.logo_url = p.logo_url || ''
      editForm.value.owner_phone = p.owner_phone || ''
      editForm.value.subdomain = p.subdomain || ''
      editForm.value.custom_domain = p.custom_domain || ''
      editForm.value.xendit_merchant_id = p.xendit_merchant_id || ''
    } else {
      profileError.value = 'Gagal memuat profil'
    }
  } catch (e) {
    profileError.value = 'Kesalahan jaringan'
  }
}

const onLogoFileChange = (e: Event) => {
  const file = (e.target as HTMLInputElement).files?.[0]
  if (!file) return
  editLogoFile.value = file
  editLogoPreview.value = URL.createObjectURL(file)
}

const uploadLogo = async () => {
  if (!editLogoFile.value || !editTarget.value) return
  uploadingLogo.value = true
  try {
    const result = await superadminApi.uploadTenantLogo(editTarget.value.id, editLogoFile.value)
    if (result.success) {
      editForm.value.logo_url = result.data?.logo_url || ''
      editLogoFile.value = null
      showToast('Logo berhasil diupload', 'success')
    } else {
      profileError.value = result.message || 'Gagal upload logo'
    }
  } catch (e) {
    profileError.value = 'Kesalahan jaringan saat upload'
  } finally {
    uploadingLogo.value = false
  }
}

const saveProfile = async () => {
  if (!editTarget.value) return
  savingProfile.value = true
  profileError.value = ''

  try {
    if (editLogoFile.value) {
      const logoResult = await superadminApi.uploadTenantLogo(editTarget.value.id, editLogoFile.value)
      if (!logoResult.success) {
        profileError.value = logoResult.message || 'Gagal upload logo'
        savingProfile.value = false
        return
      }
      editForm.value.logo_url = logoResult.data?.logo_url || ''
      editLogoFile.value = null
    }

    const payload: any = {
      tenant_id: editTarget.value.id,
      name: editForm.value.name,
      business_name: editForm.value.business_name,
      wa_number: editForm.value.wa_number,
      owner_phone: editForm.value.owner_phone,
      business_address: editForm.value.business_address,
      business_type: editForm.value.business_type,
      plan: editForm.value.plan,
      subdomain: editForm.value.subdomain,
      custom_domain: editForm.value.custom_domain,
      xendit_merchant_id: editForm.value.xendit_merchant_id
    }
    if (editForm.value.new_password) {
      payload.new_password = editForm.value.new_password
    }

    const result = await superadminApi.updateTenantProfile(payload)
    if (result.success) {
      showToast('Profil tenant berhasil disimpan', 'success')
      editTarget.value = null
      fetchTenants()
    } else {
      profileError.value = result.message || 'Gagal menyimpan'
    }
  } catch (e) {
    profileError.value = 'Kesalahan jaringan'
  } finally {
    savingProfile.value = false
  }
}

const closeAddModal = () => {
  showAddTenant.value = false
  formData.value = { name: '', username: '', password: 'Password123', email: '', phone_number: '', role: 'owner', plan: 'lite', subdomain: '', custom_domain: '' }
}

// ── Plan Editor ──────────────────────────────────────────────────────────────────

const editablePlans = ref<any[]>([])
const loadingPlans = ref(false)
const savingPlans = ref(false)
const planError = ref('')
const planOptions = ref<any[]>([])

// ── Voucher Generation ────────────────────────────────────────────────────────
const generatingVoucher = ref(false)
const voucherError = ref('')
const voucherForm = ref({
  program_name: '',
  plan_id: '',
  quantity: 10,
  validity_days: 30,
  voucher_type: 'bonus_months',
  discount_value: 0,
  max_uses: null as number | null,
})
const voucherList = ref<any[]>([])
const loadingVoucherList = ref(false)
const voucherListFilter = ref({ used: '', plan_id: '' })
const generatedVoucherCodes = ref<any[]>([])
const deletingVoucherId = ref<string | null>(null)

const openGenerateVoucher = async () => {
  voucherError.value = ''
  generatedVoucherCodes.value = []
  voucherForm.value = { program_name: '', plan_id: '', quantity: 10, validity_days: 30, voucher_type: 'bonus_months', discount_value: 0, max_uses: null }
  showGenerateVoucherModal.value = true
  // Fetch latest plan prices from backend
  try {
    const data = await superadminApi.getPlans()
    const plans = data.data || (data.success && data.data)
    if (plans && Array.isArray(plans)) {
      planOptions.value = plans
    }
  } catch (e) {
    console.error('Failed to fetch plan options', e)
  }
}

const openVoucherList = async () => {
  voucherListFilter.value = { used: '', plan_id: '' }
  fetchVoucherList()
  showVoucherListModal.value = true
  // Fetch latest plan prices from backend
  try {
    const data = await superadminApi.getPlans()
    const plans = data.data || (data.success && data.data)
    if (plans && Array.isArray(plans)) {
      planOptions.value = plans
    }
  } catch (e) {
    console.error('Failed to fetch plan options', e)
  }
}

const executeGenerateVoucher = async () => {
  if (!voucherForm.value.plan_id || !voucherForm.value.quantity || !voucherForm.value.validity_days) return
  generatingVoucher.value = true
  voucherError.value = ''
  try {
    const data = await superadminApi.generateVouchers({
      plan_id: voucherForm.value.plan_id,
      validity_days: voucherForm.value.validity_days,
      quantity: voucherForm.value.quantity,
      voucher_type: voucherForm.value.voucher_type,
      discount_value: voucherForm.value.discount_value,
      program_name: voucherForm.value.program_name || undefined,
      max_uses: voucherForm.value.max_uses || undefined,
    })
    if (data.success || data.status === 200) {
      generatedVoucherCodes.value = data.data?.codes || []
      showToast(`Berhasil generate ${data.data?.count || 0} voucher!`, 'success')
    } else {
      voucherError.value = data.message || 'Gagal generate voucher'
    }
  } catch (e) {
    voucherError.value = 'Kesalahan jaringan'
  } finally {
    generatingVoucher.value = false
  }
}

const copyToClipboard = async (text: string, msg?: string) => {
  try {
    await navigator.clipboard.writeText(text)
    showToast(msg || 'Berhasil disalin!', 'success')
  } catch {
    showToast('Gagal menyalin', 'error')
  }
}

const deleteVoucher = async (id: string, code: string) => {
  if (!confirm(`Hapus voucher "${code}"? Voucher yang belum terpakai akan dihapus permanen.`)) return
  deletingVoucherId.value = id
  try {
    const data = await superadminApi.deleteVoucher(id)
    if (data.success || data.status === 200) {
      voucherList.value = voucherList.value.filter(v => v.id !== id)
      showToast(`Voucher ${code} berhasil dihapus`, 'success')
    } else {
      showToast(data.message || 'Gagal menghapus voucher', 'error')
    }
  } catch (e) {
    showToast('Kesalahan jaringan', 'error')
  } finally {
    deletingVoucherId.value = null
  }
}

const downloadVoucherCSV = () => {
  if (!generatedVoucherCodes.value.length) return
  const header = 'code,validity_days\n'
  const rows = generatedVoucherCodes.value.map((v: any) => `${v.code},${v.days}`).join('\n')
  const csv = header + rows
  const blob = new Blob([csv], { type: 'text/csv' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `vouchers-${Date.now()}.csv`
  a.click()
  URL.revokeObjectURL(url)
}

const copyText = (text: string) => {
  navigator.clipboard.writeText(text).then(() => {
    showToast('Kode berhasil disalin!', 'success')
  })
}

const fetchVoucherList = async () => {
  loadingVoucherList.value = true
  try {
    const data = await superadminApi.listVouchers({
      used: voucherListFilter.value.used || undefined,
      plan_id: voucherListFilter.value.plan_id || undefined,
      limit: 200,
    })
    if ((data.success || data.status === 200) && data.data?.codes) {
      voucherList.value = data.data.codes
    } else if (data.data && Array.isArray(data.data)) {
      voucherList.value = data.data
    } else {
      voucherList.value = []
    }
  } catch (e) {
    console.error('Failed to fetch voucher list', e)
    voucherList.value = []
  } finally {
    loadingVoucherList.value = false
  }
}

// Add-on state & functions
const loadingAddons = ref(false)
const savingAddons = ref(false)
const addonOptions = ref<any[]>([])
const addonSaveMsg = ref('')

const openAddonEditor = async () => {
  showAddonEditor.value = true
  addonSaveMsg.value = ''
  loadingAddons.value = true
  try {
    const data = await superadminApi.getAddonPrices()
    if (data && data.success) {
      addonOptions.value = (data.data || []).map((a: any) => ({
        ...a,
        price: Math.round((a.price_cents || 0) / 100)
      }))
    } else {
      addonSaveMsg.value = data?.message || 'Gagal memuat add-on'
    }
  } catch (e) {
    addonSaveMsg.value = 'Kesalahan jaringan memuat add-on'
  } finally {
    loadingAddons.value = false
  }
}

const saveAddons = async () => {
  savingAddons.value = true
  addonSaveMsg.value = ''
  let hasError = false
  try {
    for (const addon of addonOptions.value) {
      const payload = {
        price_cents: Math.round((addon.price || 0) * 100),
        unit: addon.unit,
        description: addon.description,
        is_active: addon.is_active
      }
      const result = await superadminApi.updateAddonPrice(addon.addon_key, payload)
      if (!result.success) hasError = true
    }
    addonSaveMsg.value = hasError ? 'Beberapa perubahan gagal disimpan' : 'Semua perubahan berhasil disimpan'
  } catch (e) {
    addonSaveMsg.value = 'Kesalahan jaringan saat menyimpan'
  } finally {
    savingAddons.value = false
  }
}

const openPlanEditor = async () => {
  showPlanEditor.value = true
  planError.value = ''
  loadingPlans.value = true
  try {
    const data = await superadminApi.getPlans()
    const plans = data.data || (data.success && data.data)
    if (plans && Array.isArray(plans)) {
      editablePlans.value = plans.map((p: any) => ({
        ...p,
        price_monthly_display: Math.round((p.price_monthly || 0) / 100),
        price_yearly_display: Math.round((p.price_yearly || 0) / 100),
      }))
    } else {
      planError.value = 'Gagal memuat daftar paket'
    }
  } catch (e) {
    planError.value = 'Kesalahan jaringan'
  } finally {
    loadingPlans.value = false
  }
}

const syncPlanPrice = (plan: any, kind: 'monthly' | 'yearly') => {
  if (kind === 'monthly') {
    plan.price_monthly = (plan.price_monthly_display || 0) * 100
  } else {
    plan.price_yearly = (plan.price_yearly_display || 0) * 100
  }
}

const savePlanPrices = async () => {
  savingPlans.value = true
  planError.value = ''
  try {
    let allOk = true
    for (const plan of editablePlans.value) {
      const result = await superadminApi.updatePlan(plan.id, {
        price_monthly: plan.price_monthly || 0,
        price_yearly: plan.price_yearly || 0,
        is_active: plan.is_active,
        sort_order: plan.sort_order || 0,
      })
      if (!result.success && result.status !== 200) {
        allOk = false
        planError.value = `Gagal menyimpan paket ${plan.name}: ${result.message}`
      }
    }
    if (allOk) {
      showToast('Harga paket berhasil diperbarui')
      showPlanEditor.value = false
    }
  } catch (e) {
    planError.value = 'Kesalahan jaringan'
  } finally {
    savingPlans.value = false
  }
}

const saveNewTenant = async () => {
  try {
    const data = await api.post('/api/umkm/admin/tenants', {
      name: formData.value.name,
      username: formData.value.username,
      email: formData.value.email,
      phone_number: formData.value.phone_number,
      plan: formData.value.plan,
      subdomain: formData.value.subdomain,
      custom_domain: formData.value.custom_domain
    })
    if (data.success) {
      showToast("Berhasil mendaftarkan UMKM baru!", "success")
      closeAddModal()
      fetchTenants()
    } else {
      showToast("Gagal: " + data.message, "error")
    }
  } catch (e) {
    console.error(e)
    showToast("Terjadi kesalahan jaringan.", "error")
  }
}

onMounted(async () => {
  checkVerifierStatus()
  fetchTenants()
  loadMyProfile()
  // Load plan options for dropdowns
  try {
    const data = await superadminApi.getPlans()
    const plans = data.data || (data.success && data.data)
    if (plans && Array.isArray(plans)) {
      planOptions.value = plans
    }
  } catch (e) {
    console.error('Failed to fetch plan options', e)
  }
})
</script>

<style scoped>
.superadmin-dashboard {
  max-width: 1200px;
  margin: 0 auto;
}

.dashboard-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 1.5rem;
}

@media (max-width: 768px) {
  .dashboard-grid {
    grid-template-columns: 1fr;
  }
}

.card {
  padding: 1.5rem;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1rem;
}

.card-header h3 {
  margin: 0;
  font-size: 1rem;
  color: var(--text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.status-badge {
  padding: 0.3rem 0.75rem;
  border-radius: 20px;
  font-size: 0.75rem;
  font-weight: 600;
}

.status-connected {
  background: rgba(16, 185, 129, 0.15);
  color: #10b981;
}

.status-disconnected {
  background: rgba(239, 68, 68, 0.15);
  color: #ef4444;
}

.verifier-actions {
  text-align: center;
  padding: 1rem 0;
}

.verifier-actions p {
  margin-bottom: 1.25rem;
}

.qr-section {
  text-align: center;
}

.qr-image {
  width: 200px;
  height: 200px;
  border-radius: 8px;
  border: 2px solid var(--border-color);
}

.qr-actions {
  display: flex;
  justify-content: center;
  gap: 1rem;
  margin-top: 1rem;
}

.verifier-info {
  padding: 0.5rem 0;
}

.info-row {
  display: flex;
  justify-content: space-between;
  padding: 0.75rem 0;
  border-bottom: 1px solid var(--border-color);
}

.info-label {
  color: var(--text-secondary);
  font-size: 0.875rem;
}

.info-value {
  font-weight: 600;
  font-size: 0.9rem;
}

.stat-number {
  font-size: 2.5rem;
  font-weight: 700;
  color: var(--accent-primary);
}

.plan-stats {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.plan-stat {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  font-weight: 600;
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

.toast-notification {
  position: fixed;
  top: 2rem;
  right: 2rem;
  padding: 1rem 1.5rem;
  border-radius: 8px;
  color: #fff;
  font-weight: 500;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.2);
  z-index: 9999;
  animation: slideDown 0.3s ease-out;
}

.toast-success {
  background-color: #10b981;
}

.toast-error {
  background-color: #ef4444;
}

@keyframes slideDown {
  from {
    transform: translateY(-100%);
    opacity: 0;
  }

  to {
    transform: translateY(0);
    opacity: 1;
  }
}

.btn-edit {
  background: transparent;
  border: 1px solid rgba(59, 130, 246, 0.3);
  color: #60a5fa;
  padding: 0.35rem 0.5rem;
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: all 0.2s;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  margin-right: 0.35rem;
}

.btn-edit:hover {
  background: rgba(59, 130, 246, 0.15);
  border-color: #60a5fa;
}

.logo-upload-section {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  margin-bottom: 1.25rem;
  flex-wrap: wrap;
}

.logo-preview {
  width: 80px;
  height: 80px;
  border-radius: 8px;
  border: 1px solid var(--border-color);
  overflow: hidden;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--bg-tertiary);
  flex-shrink: 0;
}

.logo-preview img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.logo-placeholder {
  font-size: 0.7rem;
  color: var(--text-secondary);
  text-align: center;
}

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

.btn-delete {
  background: transparent;
  border: 1px solid rgba(239, 68, 68, 0.3);
  color: #ef4444;
  padding: 0.35rem 0.5rem;
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: all 0.2s;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.btn-delete:hover {
  background: rgba(239, 68, 68, 0.15);
  border-color: #ef4444;
}

@media (max-width: 480px) {

  .data-table th,
  .data-table td {
    padding: 0.5rem;
    font-size: 0.8rem;
  }
}

/* Plan Editor */
.plan-editor-row {
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  padding: 1rem;
  margin-bottom: 1rem;
}

.plan-editor-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 0.75rem;
}

.plan-editor-fields {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0.75rem;
}

/* Toggle Switch */
.toggle-switch {
  position: relative;
  display: inline-block;
  width: 40px;
  height: 22px;
  cursor: pointer;
}

.toggle-switch input {
  opacity: 0;
  width: 0;
  height: 0;
}

.toggle-slider {
  position: absolute;
  inset: 0;
  background-color: rgba(100, 116, 139, 0.4);
  border-radius: 22px;
  transition: 0.2s;
}

.toggle-slider::before {
  content: '';
  position: absolute;
  height: 16px;
  width: 16px;
  left: 3px;
  bottom: 3px;
  background-color: white;
  border-radius: 50%;
  transition: 0.2s;
}

.toggle-switch input:checked + .toggle-slider {
  background-color: #10b981;
}

.toggle-switch input:checked + .toggle-slider::before {
  transform: translateX(18px);
}

@media (max-width: 480px) {
  .plan-editor-fields {
    grid-template-columns: 1fr;
  }
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
