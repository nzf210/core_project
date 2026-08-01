export const API_BASE = import.meta.env.VITE_API_URL || (import.meta.env.PROD ? globalThis.location.origin : 'http://localhost:8000')

// Sanitization functions for localStorage security
export function sanitizeUUID(v: unknown): string {
  const str = String(v ?? '')
  return /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(str) ? str : ''
}

export function sanitizeJWT(v: unknown): string {
  const str = String(v ?? '')
  return /^[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+$/.test(str) ? str.substring(0, 2048) : ''
}

export function sanitizeRole(v: unknown): string {
  const str = String(v ?? '').toLowerCase()
  return ['owner', 'admin', 'staff', 'kasir', 'superadmin'].includes(str) ? str : ''
}

export function sanitizeText(v: unknown, maxLen = 200): string {
  return String(v ?? '').replace(/[<>"'`&]/g, '').substring(0, maxLen)
}

export function sanitizeURL(v: unknown): string {
  const str = String(v ?? '')
  try {
    const url = new URL(str)
    return ['http:', 'https:'].includes(url.protocol) ? str.substring(0, 500) : ''
  } catch {
    return ''
  }
}

export async function initDomain() {
  const hostname = globalThis.window.location.hostname
  try {
    const controller = new AbortController()
    const timeoutId = setTimeout(() => controller.abort(), 3000)
    const res = await fetch(`${API_BASE}/auth/public/tenant/resolve?domain=${hostname}`, { signal: controller.signal })
    clearTimeout(timeoutId)
    if (res.ok) {
      const data = await res.json()
      if (data.success && data.data) {
        localStorage.setItem('active_domain_tenant_id', sanitizeUUID(data.data.tenant_id))
        localStorage.setItem('active_domain_business_name', sanitizeText(data.data.business_name))
        localStorage.setItem('active_domain_logo_url', sanitizeURL(data.data.logo_url))
        return
      }
    }
    // If not successful or not found, remove
    localStorage.removeItem('active_domain_tenant_id')
    localStorage.removeItem('active_domain_business_name')
    localStorage.removeItem('active_domain_logo_url')
  } catch(e: unknown) {
    if (e instanceof Error && e.name === 'AbortError') {
      // timeout — domain resolve skipped, app mounts anyway
      localStorage.removeItem('active_domain_tenant_id')
      localStorage.removeItem('active_domain_business_name')
      localStorage.removeItem('active_domain_logo_url')
      return
    }
    console.error('Failed to resolve domain', e)
  }
}

function getTenantID(): string {
  return localStorage.getItem('tenant_id') || ''
}

function getToken(): string {
  return localStorage.getItem('access_token') || ''
}

function headers(withAuth = true): Record<string, string> {
  const h: Record<string, string> = {
    'Content-Type': 'application/json',
    'X-Tenant-ID': getTenantID(),
  }
  if (withAuth) {
    h['Authorization'] = `Bearer ${getToken()}`
  }
  return h
}

export const api = {
  // Clinic Queue System (F045)
  async getClinicSettings() {
    const res = await fetch(`${API_BASE}/api/umkm/clinic/settings`, { headers: headers() })
    return res.json()
  },
  async updateClinicSettings(settings: any) {
    const res = await fetch(`${API_BASE}/api/umkm/clinic/settings`, { method: 'PUT', headers: headers(), body: JSON.stringify(settings) })
    return res.json()
  },
  async bookClinicAppointment(appointment: any) {
    const res = await fetch(`${API_BASE}/api/umkm/clinic/appointments/book`, { method: 'POST', headers: headers(), body: JSON.stringify(appointment) })
    return res.json()
  },
  async cancelClinicAppointment(appointmentId: string, performedBy: string) {
    const res = await fetch(`${API_BASE}/api/umkm/clinic/appointments/cancel`, { method: 'PUT', headers: headers(), body: JSON.stringify({ appointment_id: appointmentId, performed_by: performedBy }) })
    return res.json()
  },
  async getClinicQueue() {
    const res = await fetch(`${API_BASE}/api/umkm/clinic/appointments/queue`, { headers: headers() })
    return res.json()
  },
  async callClinicAppointment(appointmentId: string) {
    const res = await fetch(`${API_BASE}/api/umkm/clinic/appointments/call`, { method: 'PUT', headers: headers(), body: JSON.stringify({ appointment_id: appointmentId }) })
    return res.json()
  },

  // F047: Medical Records
  async getClinicMedicalRecords() {
    const res = await fetch(`${API_BASE}/api/umkm/clinic/medical-records`, { headers: headers() })
    return res.json()
  },
  async createClinicMedicalRecord(record: Record<string, any>) {
    const res = await fetch(`${API_BASE}/api/umkm/clinic/medical-records`, {
      method: 'POST',
      headers: headers(),
      body: JSON.stringify(record),
    })
    return res.json()
  },

  // F047: Doctor Schedules
  async getClinicDoctors() {
    const res = await fetch(`${API_BASE}/api/umkm/clinic/doctors`, { headers: headers() })
    return res.json()
  },
  async createClinicDoctor(doctor: Record<string, any>) {
    const res = await fetch(`${API_BASE}/api/umkm/clinic/doctors`, {
      method: 'POST',
      headers: headers(),
      body: JSON.stringify(doctor),
    })
    return res.json()
  },

  // Auth (unauthenticated)
  async login(username: string, password: string) {
    const expectedTenantId = localStorage.getItem('active_domain_tenant_id') || ''
    const res = await fetch(`${API_BASE}/auth/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username, password, expectedTenantId }),
    })
    return res.json()
  },

  async phoneLogin(phoneNumber: string) {
    const res = await fetch(`${API_BASE}/auth/phone-login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ phoneNumber }),
    })
    return res.json()
  },

  async verifyPhoneLogin(phoneNumber: string, otp: string) {
    const expectedTenantId = localStorage.getItem('active_domain_tenant_id') || ''
    const res = await fetch(`${API_BASE}/auth/verify-phone-login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ phoneNumber, otp, expectedTenantId }),
    })
    return res.json()
  },

  async registerWA(body: Record<string, any>) {
    const res = await fetch(`${API_BASE}/auth/register`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        phoneNumber: body.phoneNumber,
        password: body.password,
        username: body.username,
        email: body.email || '',
        businessName: body.businessName || '',
        businessType: body.businessType || 'umum',
      }),
    })
    return res.json()
  },

  async telegramRegister(body: { telegramChatId: string; phoneNumber: string; password: string; username: string; email?: string; businessName?: string; businessType?: string }) {
    const res = await fetch(`${API_BASE}/auth/telegram/register`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        telegramChatId: body.telegramChatId,
        phoneNumber: body.phoneNumber,
        password: body.password,
        username: body.username,
        email: body.email || '',
        businessName: body.businessName || '',
        businessType: body.businessType || 'umum',
      }),
    })
    return res.json()
  },

  async telegramLogin(telegramChatId: string, phoneNumber: string) {
    const res = await fetch(`${API_BASE}/auth/telegram/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ telegramChatId, phoneNumber }),
    })
    return res.json()
  },

  async verifyOTP(phoneNumber: string, otp: string) {
    const res = await fetch(`${API_BASE}/auth/verify-otp`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ phoneNumber, otp }),
    })
    return res.json()
  },

  /**
   * GET /api/me — lightweight, GET-only summary of the current user + tenant.
   * Used by the frontend router guard to re-sync onboarding_completed, plan,
   * role, is_frozen on every page reload. Fixes the onboarding redirect loop
   * when localStorage flags are missing (e.g. login on a new device).
   */
  async me() {
    try {
      const res = await fetch(`${API_BASE}/api/me`, {
        method: 'GET',
        headers: headers(),
      })
      if (!res.ok) {
        return { success: false, message: `HTTP ${res.status}` }
      }
      return res.json()
    } catch (e: any) {
      return { success: false, message: e?.message || 'Network error' }
    }
  },

  // Chatbot config (F020) — per-tenant AI Customer Service setup
  async getChatbotConfig() {
    const res = await fetch(`${API_BASE}/api/umkm/chatbot/config`, {
      method: 'GET',
      headers: headers(),
    })
    return res.json()
  },
  async updateChatbotConfig(partial: Record<string, any>) {
    const res = await fetch(`${API_BASE}/api/umkm/chatbot/config`, {
      method: 'PUT',
      headers: headers(),
      body: JSON.stringify(partial),
    })
    return res.json()
  },
  async testChatbotConfig(message: string) {
    const res = await fetch(`${API_BASE}/api/umkm/chatbot/config/test`, {
      method: 'POST',
      headers: headers(),
      body: JSON.stringify({ message }),
    })
    return res.json()
  },
  async getChatbotPermissions() {
    // F048: WA provider addon permissions
    const res = await fetch(`${API_BASE}/api/umkm/chatbot/permissions`, {
      method: 'GET',
      headers: headers(),
    })
    return res.json()
  },

  // WA Setup (Ultimate tier) — provider choice + credit tracking
  async getWASetup() {
    const res = await fetch(`${API_BASE}/api/umkm/wa/setup`, {
      method: 'GET',
      headers: headers(),
    })
    return res.json()
  },
  async updateWAProvider(provider: 'auto' | 'whatsmeow' | 'cloud_api') {
    const res = await fetch(`${API_BASE}/api/umkm/wa/connect`, {
      method: 'POST',
      headers: headers(),
      body: JSON.stringify({ provider }),
    })
    return res.json()
  },

  // F048: Cloud API credential (per-tenant Meta credentials)
  async getCloudAPICredential() {
    const res = await fetch(`${API_BASE}/api/umkm/wa/cloud-api-credential`, {
      method: 'GET',
      headers: headers(),
    })
    return res.json()
  },
  async saveCloudAPICredential(cred: { phone_number_id: string; waba_id: string; access_token: string; verify_token: string }) {
    const res = await fetch(`${API_BASE}/api/umkm/wa/cloud-api-credential`, {
      method: 'POST',
      headers: headers(),
      body: JSON.stringify(cred),
    })
    return res.json()
  },

  async validateCloudAPICredential(cred: { access_token: string; phone_number_id: string; waba_id?: string }) {
    const res = await fetch(`${API_BASE}/api/wa/validate`, {
      method: 'POST',
      headers: headers(),
      body: JSON.stringify(cred),
    })
    if (!res.ok) {
      const err = await res.json().catch(() => ({ message: `HTTP ${res.status}` }))
      throw new Error(err.message || `Validation failed: ${res.statusText}`)
    }
    return res.json()
  },

  // Cash Flow PDF (F021) — trigger browser download via window.location
  cashFlowPDFUrl(from: string, to: string) {
    return `${API_BASE}/api/umkm/reports/cash-flow/pdf?from=${from}&to=${to}`
  },

  // Income Statement & Balance Sheet PDF (B)
  incomeStatementPDFUrl(from: string, to: string) {
    return `${API_BASE}/api/umkm/reports/income-statement/pdf?from=${from}&to=${to}`
  },
  balanceSheetPDFUrl(date: string) {
    return `${API_BASE}/api/umkm/reports/balance-sheet/pdf?date=${date}`
  },

  // Import / Export (F022) — returns blob URL for download or JSON for import
  async exportFile(endpoint: string, format: 'xlsx' | 'csv', extraParams: Record<string, string> = {}) {
    const params = new URLSearchParams({ format, ...extraParams })
    const res = await fetch(`${API_BASE}${endpoint}?${params}`, {
      method: 'GET',
      headers: headers(),
    })
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    const blob = await res.blob()
    return URL.createObjectURL(blob)
  },

  async importFile(endpoint: string, file: File) {
    const form = new FormData()
    form.append('file', file)
    // Don't set Content-Type — let browser set multipart boundary
    const h: Record<string, string> = {
      'X-Tenant-ID': getTenantID(),
    }
    const token = getToken()
    if (token) h['Authorization'] = `Bearer ${token}`
    const res = await fetch(`${API_BASE}${endpoint}`, {
      method: 'POST',
      headers: h,
      body: form,
    })
    return res.json()
  },

  templateURL(entity: 'products' | 'contacts' | 'journal', format: 'xlsx' | 'csv' = 'csv') {
    return `${API_BASE}/api/umkm/import/template?entity=${entity}&format=${format}`
  },

  async register(body: Record<string, any>) {
    const res = await fetch(`${API_BASE}/api/umkm/admin/tenants`, {
      method: 'POST',
      headers: headers(),
      body: JSON.stringify(body),
    })
    return res.json()
  },

  async forgotPassword(email: string) {
    const res = await fetch(`${API_BASE}/auth/forgot-password`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email }),
    })
    return res.json()
  },

  async resetPassword(token: string, newPassword: string) {
    const res = await fetch(`${API_BASE}/auth/reset-password`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ token, newPassword }),
    })
    return res.json()
  },

  // F055 v2: Request password reset OTP via chat (WA or Telegram)
  async requestPasswordResetOTP(phoneNumber: string, channel: 'wa' | 'telegram' = 'wa') {
    const res = await fetch(`${API_BASE}/auth/reset-password-request`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ phoneNumber, channel }),
    })
    return res.json()
  },

  // F055 v2: Verify OTP and set new password
  async verifyPasswordReset(phoneNumber: string, otp: string, newPassword: string, channel: 'wa' | 'telegram' = 'wa') {
    const res = await fetch(`${API_BASE}/auth/reset-password-verify`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ phoneNumber, otp, newPassword, channel }),
    })
    return res.json()
  },

  // UMKM Data (authenticated + tenant)
  async get(url: string) {
    const res = await fetch(`${API_BASE}${url}`, { headers: headers() })
    return res.json()
  },

  async post(url: string, body?: any, isMultipart = false) {
    const h = headers()
    if (isMultipart) delete h['Content-Type']
    let requestBody: any
    if (isMultipart) requestBody = body
    else if (body) requestBody = JSON.stringify(body)

    const res = await fetch(`${API_BASE}${url}`, {
      method: 'POST',
      headers: h,
      body: requestBody,
    })
    return res.json()
  },

  async put(url: string, body?: any) {
    const res = await fetch(`${API_BASE}${url}`, {
      method: 'PUT',
      headers: headers(),
      body: body ? JSON.stringify(body) : undefined,
    })
    return res.json()
  },

  async del(url: string) {
    const res = await fetch(`${API_BASE}${url}`, { method: 'DELETE', headers: headers() })
    return res.json()
  },

  // WA Gateway (via API Gateway)
  async wa(method: string, body?: Record<string, any>) {
    const res = await fetch(`${API_BASE}/api/wa/${method}`, {
      method: body ? 'POST' : 'GET',
      headers: headers(),
      body: body ? JSON.stringify(body) : undefined,
    })
    return res.json()
  },

  // Quota usage (F025, Task 2.9) — superadmin-only dashboard endpoint
  // GET /api/superadmin/billing/quota/{tenant_id}
  // (api-gateway strips /api/superadmin/billing, forwards to billing-service /admin/quota/{id})
  async getQuotaUsage(tenantId: string) {
    const res = await fetch(`${API_BASE}/api/superadmin/billing/quota/${tenantId}`, {
      method: 'GET',
      headers: headers(),
    })
    return res.json()
  },

  // F059: Landing page — public plan/pricing data (no auth required)
  async getPublicPlans() {
    const res = await fetch(`${API_BASE}/plans`)
    return res.json()
  },

  // F060: Landing page — public dynamic content (no auth required)
  async getLandingConfigs() {
    const res = await fetch(`${API_BASE}/landing-configs`)
    return res.json()
  },

  // F034: Wallet
  async getWallet() {
    const res = await fetch(`${API_BASE}/api/billing/wallet`, { headers: headers() })
    return res.json()
  },
  async topupWallet(amountRupiah: number) {
    const res = await fetch(`${API_BASE}/api/billing/wallet/topup`, {
      method: 'POST',
      headers: headers(),
      body: JSON.stringify({ amount_rupiah: amountRupiah })
    })
    return res.json()
  },

  // F036: Affiliate & Referral
  async getAffiliateLeaderboard() {
    const res = await fetch(`${API_BASE}/api/public/affiliate-leaderboard`)
    return res.json()
  },
  async getAffiliateProfile() {
    const res = await fetch(`${API_BASE}/affiliate/profile`, { headers: headers() })
    return res.json()
  },
  async registerAffiliate() {
    const res = await fetch(`${API_BASE}/affiliate/register`, { method: 'POST', headers: headers() })
    return res.json()
  },
  async withdrawAffiliate(amountRupiah: number) {
    const res = await fetch(`${API_BASE}/affiliate/withdraw`, {
      method: 'POST',
      headers: headers(),
      body: JSON.stringify({ amount_rupiah: amountRupiah })
    })
    return res.json()
  },
  async redeemReferral(referralCode: string) {
    const res = await fetch(`${API_BASE}/affiliate/redeem-referral`, {
      method: 'POST',
      headers: headers(),
      body: JSON.stringify({ referral_code: referralCode })
    })
    return res.json()
  },

  // F053: Addon Marketplace & Purchase
  async getAddons() {
    const res = await fetch(`${API_BASE}/api/umkm/addons`, { headers: headers() })
    return res.json()
  },
  async purchaseAddon(addonKey: string) {
    const res = await fetch(`${API_BASE}/api/umkm/addons/purchase`, {
      method: 'POST',
      headers: headers(),
      body: JSON.stringify({ addon_key: addonKey })
    })
    return res.json()
  },
  async getMyAddons() {
    const res = await fetch(`${API_BASE}/api/umkm/addons`, { headers: headers() })
    return res.json()
  },
}

// ─── Quota usage types (F025, Task 2.9) ─────────────────────────
// Mirrors billing-service handleAdminQuotaUsage response shape.
export interface QuotaCounter {
  feature: string
  used: number
  reset_at: string
}

export interface QuotaPlanLimits {
  max_users: number
  max_transactions: number
  max_ai_text: number
  max_ai_vision: number
  max_ai_audio_minutes: number
  max_image_gen: number
  max_products: number
  max_customers: number
  max_storage_mb: number
  api_rate_limit_per_min: number
  data_retention_months: number
}

export interface QuotaUsage {
  tenant_id: string
  tier: string
  plan_name: string
  period: string
  limits: QuotaPlanLimits
  usage: QuotaCounter[]
}

// Typed convenience wrapper around api.getQuotaUsage.
// Returns null if the request fails or returns non-200 status.
export async function getQuotaUsage(tenantId: string): Promise<QuotaUsage | null> {
  const res = await api.getQuotaUsage(tenantId)
  if (res?.status === 200 && res?.data) return res.data as QuotaUsage
  return null
}

// F050: Staff Management API (owner-only)
export const authApi = {
  async getStaffList() {
    const res = await fetch(`${API_BASE}/auth/staff`, { headers: headers() })
    return res.json()
  },
  async updateStaff(data: { id?: string; username?: string; phone_number?: string; password?: string }) {
    const res = await fetch(`${API_BASE}/auth/staff/update`, {
      method: 'PUT',
      headers: headers(),
      body: JSON.stringify(data),
    })
    return res.json()
  },
  async deleteStaff(id: string) {
    const res = await fetch(`${API_BASE}/auth/staff/delete?id=${encodeURIComponent(id)}`, {
      method: 'DELETE',
      headers: headers(),
    })
    return res.json()
  },
}

// F060: Sales Dashboard Chart API
export interface SalesChartData {
  period: string
  labels: string[]
  revenue: number[]
  expense: number[]
  profit: number[]
}

export interface TopProduct {
  name: string
  revenue_rupiah: number
  transaction_count: number
}

export interface RecentTransaction {
  id: string
  date: string
  description: string
  amount_rupiah: number
}

export const reportsApi = {
  async getSalesChart(period: 'week' | 'month' | 'year' = 'week') {
    return api.get(`/api/umkm/reports/sales-chart?period=${period}`)
  },
  async getTopProducts(limit = 5) {
    return api.get(`/api/umkm/reports/top-products?limit=${limit}`)
  },
  async getRecentTransactions(limit = 5) {
    return api.get(`/api/umkm/reports/recent-transactions?limit=${limit}`)
  },
}
