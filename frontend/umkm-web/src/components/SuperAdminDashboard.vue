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
          <button class="btn" style="background: rgba(16, 185, 129, 0.15); color: #10b981; border: 1px solid rgba(16, 185, 129, 0.3); padding: 0.35rem 0.75rem; font-size: 0.8rem;" @click="openFeatureMatrix">
            Feature Matrix
          </button>
          <button class="btn" style="background: rgba(245, 158, 11, 0.15); color: #f59e0b; border: 1px solid rgba(245, 158, 11, 0.3); padding: 0.35rem 0.75rem; font-size: 0.8rem;" @click="showLandingEditor = true">
            🌐 Edit Landing Page
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


    <!-- Modals extracted to sub-components -->
    <SuperAdminAddTenantModal
      v-if="showAddTenant"
      :show-add-tenant="showAddTenant"
      :form-data="formData"
      :plan-options="planOptions"
      @update:show-add-tenant="showAddTenant = $event"
      @close="closeAddModal"
      @save="saveNewTenant"
    />

    <SuperAdminMyProfileModal
      v-if="showMyProfile"
      :show-my-profile="showMyProfile"
      :my-profile="myProfile"
      :saving-my-profile="savingMyProfile"
      :my-profile-error="myProfileError"
      @update:show-my-profile="showMyProfile = $event"
      @save="saveMyProfile"
    />

    <SuperAdminEditProfileModal
      v-if="editTarget"
      :edit-target="editTarget"
      :edit-form="editForm"
      :edit-logo-file="editLogoFile"
      :edit-logo-preview="editLogoPreview"
      :uploading-logo="uploadingLogo"
      :saving-profile="savingProfile"
      :profile-error="profileError"
      :business-types="businessTypes"
      :plan-options="planOptions"
      @update:edit-target="editTarget = $event"
      @logo-change="onLogoFileChange"
      @upload-logo="uploadLogo"
      @save="saveProfile"
    />

    <SuperAdminDeleteModal
      v-if="deleteTarget"
      :delete-target="deleteTarget"
      :deleting="deleting"
      :delete-error="deleteError"
      @update:delete-target="deleteTarget = $event"
      @confirm="executeDelete"
    />

    <SuperAdminVoucherModal
      :show-generate-voucher-modal="showGenerateVoucherModal"
      :show-voucher-list-modal="showVoucherListModal"
      :voucher-form="voucherForm"
      :plan-options="planOptions"
      :generated-voucher-codes="generatedVoucherCodes"
      :voucher-list="voucherList"
      :loading-voucher-list="loadingVoucherList"
      :voucher-list-filter="voucherListFilter"
      :generating-voucher="generatingVoucher"
      :voucher-error="voucherError"
      :deleting-voucher-id="deletingVoucherId"
      @update:show-generate-voucher-modal="showGenerateVoucherModal = $event"
      @update:show-voucher-list-modal="showVoucherListModal = $event"
      @generate="executeGenerateVoucher"
      @delete-voucher="deleteVoucher"
      @fetch-list="fetchVoucherList"
      @download-csv="downloadVoucherCSV"
      @copy="copyToClipboard"
    />

    <SuperAdminAddonModal
      v-if="showAddonEditor"
      :show-addon-editor="showAddonEditor"
      :loading-addons="loadingAddons"
      :saving-addons="savingAddons"
      :addon-options="addonOptions"
      :addon-save-msg="addonSaveMsg"
      :show-add-addon-form="showAddAddonForm"
      :deleting-addon="deletingAddon"
      :new-addon="newAddon"
      @update:show-addon-editor="showAddonEditor = $event"
      @save="saveAddons"
      @create-addon="createAddon"
      @delete-addon="deleteAddon"
    />

    <SuperAdminPlanModal
      v-if="showPlanEditor"
      :show-plan-editor="showPlanEditor"
      :loading-plans="loadingPlans"
      :saving-plans="savingPlans"
      :editable-plans="editablePlans"
      :plan-error="planError"
      @update:show-plan-editor="showPlanEditor = $event"
      @save="savePlanPrices"
    />

    <SuperAdminFeatureMatrixModal
      v-if="showFeatureMatrix"
      :show-feature-matrix="showFeatureMatrix"
      :feature-matrix-loading="featureMatrixLoading"
      :feature-matrix-plans="featureMatrixPlans"
      :feature-matrix-plan-ids="featureMatrixPlanIds"
      :feature-matrix-order="featureMatrixOrder"
      :feature-matrix-data="featureMatrixData"
      :addon-gating-loading="addonGatingLoading"
      :addon-gating-list="addonGatingList"
      @update:show-feature-matrix="showFeatureMatrix = $event"
      @toggle="toggleFeature"
      @save-gating="saveAddonMinTier"
    />

    <!-- F065: Landing Page Content Editor -->
    <SuperAdminLandingEditorModal
      :show="showLandingEditor"
      @close="showLandingEditor = false"
    />

    <!-- Toast -->
    <Teleport to="body">
      <div v-if="toast.visible" :class="['toast-notification', `toast-${toast.type}`]" :style="{ top: toastTop + 'px' }">
        {{ toast.message }}
      </div>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { useSuperAdmin } from '../composables/useSuperAdmin'
import { ref } from 'vue'
import SuperAdminAddTenantModal from './SuperAdminAddTenantModal.vue'
import SuperAdminMyProfileModal from './SuperAdminMyProfileModal.vue'
import SuperAdminEditProfileModal from './SuperAdminEditProfileModal.vue'
import SuperAdminDeleteModal from './SuperAdminDeleteModal.vue'
import SuperAdminVoucherModal from './SuperAdminVoucherModal.vue'
import SuperAdminAddonModal from './SuperAdminAddonModal.vue'
import SuperAdminPlanModal from './SuperAdminPlanModal.vue'
import SuperAdminFeatureMatrixModal from './SuperAdminFeatureMatrixModal.vue'
import SuperAdminLandingEditorModal from './SuperAdminLandingEditorModal.vue'

const {
  verifierStatus, verifierJID, qrCode, loadingQR, checkingStatus, disconnecting,
  tenants, deleting, deleteTarget, deleteError, loadingTenants,
  showAddTenant, formData, editTarget, editForm, editLogoFile, editLogoPreview,
  uploadingLogo, savingProfile, profileError,
  showGenerateVoucherModal, showVoucherListModal, showPlanEditor, showAddonEditor,
  showFeatureMatrix, featureMatrixLoading, featureMatrixPlans, featureMatrixPlanIds,
  featureMatrixOrder, featureMatrixData, addonGatingLoading, addonGatingList,
  businessTypes, toast, toastTop,
  showMyProfile, myProfile, savingMyProfile, saveMyProfile, role, myProfileError,
  planCounts, fetchTenants, confirmDelete, isMyOwnTenant, executeDelete,
  checkVerifierStatus, connectVerifier, disconnectVerifier,
  openEditProfile, onLogoFileChange, uploadLogo, saveProfile, closeAddModal,
  editablePlans, loadingPlans, savingPlans, planError, planOptions,
  generatingVoucher, voucherError, voucherForm, voucherList, loadingVoucherList,
  voucherListFilter, generatedVoucherCodes, deletingVoucherId,
  openGenerateVoucher, openVoucherList, executeGenerateVoucher, copyToClipboard,
  deleteVoucher, downloadVoucherCSV, fetchVoucherList,
  loadingAddons, savingAddons, addonOptions, addonSaveMsg, showAddAddonForm,
  deletingAddon, newAddon, openAddonEditor, createAddon, deleteAddon, saveAddons,
  openPlanEditor, savePlanPrices,
  openFeatureMatrix, toggleFeature,
  saveAddonMinTier, saveNewTenant,
} = useSuperAdmin()

// F065: Landing page content editor state (lokal di komponen)
const showLandingEditor = ref(false)

</script>

<style scoped>
@import '../assets/superadmin-dashboard.css';
</style>
