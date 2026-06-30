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

			<!-- Admin links — all tenant management moved to superadmin-web port 3202 -->
			<div class="card glass-card" style="grid-column: span 3;">
				<div class="card-header">
					<h3>Manajemen Tenant</h3>
					<span class="badge" style="background: rgba(245,158,11,0.15); color: #f59e0b; border: 1px solid rgba(245,158,11,0.3); padding: 0.2rem 0.6rem; font-size: 0.7rem; border-radius: 4px;">Superadmin Panel</span>
				</div>
				<div class="card-body">
					<p class="text-muted" style="margin-bottom: 1rem; font-size: 0.85rem;">
						Semua fitur manajemen tenant, paket, voucher, dan landing page sekarang tersedia di dashboard superadmin dedicated.
					</p>
					<div style="display: flex; gap: 0.5rem; flex-wrap: wrap;">
						<a href="http://localhost:3202/tenants" target="_blank" class="admin-link-btn">👥 Tenant</a>
						<a href="http://localhost:3202/vouchers/programs" target="_blank" class="admin-link-btn">🎫 Voucher Programs</a>
						<a href="http://localhost:3202/vouchers/generate" target="_blank" class="admin-link-btn">🔗 Generate Links</a>
						<a href="http://localhost:3202/landing-editor" target="_blank" class="admin-link-btn">🌐 Landing Editor</a>
						<a href="http://localhost:3202/plan-features" target="_blank" class="admin-link-btn">💰 Paket</a>
						<a href="http://localhost:3202/feature-matrix" target="_blank" class="admin-link-btn">🔲 Feature Matrix</a>
						<a href="http://localhost:3202/addon-pricing" target="_blank" class="admin-link-btn">📦 Add-on</a>
						<a href="http://localhost:3202/referral-config" target="_blank" class="admin-link-btn">🤝 Referral</a>
						<a href="http://localhost:3202/frozen-accounts" target="_blank" class="admin-link-btn">❄️ Frozen Accounts</a>
						<a href="http://localhost:3202/campaign-licenses" target="_blank" class="admin-link-btn">🏛️ Campaign Licenses</a>
					</div>
				</div>
			</div>
		</div>

		<!-- My Profile Modal -->
		<SuperAdminMyProfileModal
			v-if="showMyProfile"
			:show-my-profile="showMyProfile"
			:my-profile="myProfile"
			:saving-my-profile="savingMyProfile"
			:my-profile-error="myProfileError"
			@update:show-my-profile="showMyProfile = $event"
			@save="saveMyProfile"
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
import SuperAdminMyProfileModal from './SuperAdminMyProfileModal.vue'

const {
	verifierStatus, verifierJID, qrCode, loadingQR, checkingStatus, disconnecting,
	checkVerifierStatus, connectVerifier, disconnectVerifier,
	showMyProfile, myProfile, savingMyProfile, saveMyProfile, role, myProfileError,
	toast, toastTop,
} = useSuperAdmin()
</script>

<style scoped>
@import '../assets/superadmin-dashboard.css';

.admin-link-btn {
	display: inline-flex;
	align-items: center;
	gap: 0.3rem;
	padding: 0.4rem 0.8rem;
	background: rgba(59, 130, 246, 0.1);
	border: 1px solid rgba(59, 130, 246, 0.25);
	color: #60a5fa;
	border-radius: 6px;
	font-size: 0.8rem;
	text-decoration: none;
	transition: background 0.2s, border-color 0.2s;
}
.admin-link-btn:hover {
	background: rgba(59, 130, 246, 0.2);
	border-color: rgba(59, 130, 246, 0.4);
	text-decoration: none;
}
</style>
