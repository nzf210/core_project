export const API_BASE = import.meta.env.VITE_API_URL || (import.meta.env.PROD ? window.location.origin : 'http://localhost:8000')

export async function initDomain() {
  const hostname = window.location.hostname
  try {
    const res = await fetch(`${API_BASE}/auth/public/tenant/resolve?domain=${hostname}`)
    if (res.ok) {
      const data = await res.json()
      if (data.success && data.data) {
        localStorage.setItem('active_domain_tenant_id', data.data.tenant_id)
        localStorage.setItem('active_domain_business_name', data.data.business_name)
        localStorage.setItem('active_domain_logo_url', data.data.logo_url)
        return
      }
    }
    // If not successful or not found, remove
    localStorage.removeItem('active_domain_tenant_id')
    localStorage.removeItem('active_domain_business_name')
    localStorage.removeItem('active_domain_logo_url')
  } catch(e) {
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
    const res = await fetch(`${API_BASE}/clinic/settings`, { headers: headers() })
    return res.json()
  },
  async updateClinicSettings(settings: any) {
    const res = await fetch(`${API_BASE}/clinic/settings`, { method: 'PUT', headers: headers(), body: JSON.stringify(settings) })
    return res.json()
  },
  async bookClinicAppointment(appointment: any) {
    const res = await fetch(`${API_BASE}/clinic/appointments/book`, { method: 'POST', headers: headers(), body: JSON.stringify(appointment) })
    return res.json()
  },
  async cancelClinicAppointment(appointmentId: string, performedBy: string) {
    const res = await fetch(`${API_BASE}/clinic/appointments/cancel`, { method: 'PUT', headers: headers(), body: JSON.stringify({ appointment_id: appointmentId, performed_by: performedBy }) })
    return res.json()
  },
  async getClinicQueue() {
    const res = await fetch(`${API_BASE}/clinic/appointments/queue`, { headers: headers() })
    return res.json()
  },
  async callClinicAppointment(appointmentId: string) {
    const res = await fetch(`${API_BASE}/clinic/appointments/call`, { method: 'PUT', headers: headers(), body: JSON.stringify({ appointment_id: appointmentId }) })
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
      }),
    })
    return res.json()
  },

  async telegramRegister(body: { telegramChatId: string; phoneNumber: string; password: string; username: string; email?: string; businessName?: string }) {
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

  // UMKM Data (authenticated + tenant)
  async get(url: string) {
    const res = await fetch(`${API_BASE}${url}`, { headers: headers() })
    return res.json()
  },

  async post(url: string, body?: any, isMultipart = false) {
    const h = headers()
    if (isMultipart) delete h['Content-Type']
    const res = await fetch(`${API_BASE}${url}`, {
      method: 'POST',
      headers: h,
      body: isMultipart ? body : (body ? JSON.stringify(body) : undefined),
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
  // GET /api/superadmin/billing/admin/quota/{tenant_id}
  // (api-gateway strips /api/superadmin/billing, forwards to billing-service /admin/quota/{id})
  async getQuotaUsage(tenantId: string) {
    const res = await fetch(`${API_BASE}/api/superadmin/billing/admin/quota/${tenantId}`, {
      method: 'GET',
      headers: headers(),
    })
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
// Returns null if the request fails or returns success:false (e.g. 403 for non-superadmin).
export async function getQuotaUsage(tenantId: string): Promise<QuotaUsage | null> {
  const res = await api.getQuotaUsage(tenantId)
  if (res && res.success && res.data) return res.data as QuotaUsage
  return null
}
