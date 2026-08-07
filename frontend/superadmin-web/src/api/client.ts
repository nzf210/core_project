// Lightweight API client for superadmin. All requests go through /api which
// is reverse-proxied to api-gateway (port 8000) by vite dev server.
// Api-gateway then injects X-User-Role and routes to billing-service.

const API_BASE = import.meta.env.VITE_API_URL || ''

function getToken(): string | null {
  return localStorage.getItem('access_token')
}

function getRole(): string {
  const tok = getToken()
  if (!tok) return ''
  try {
    const payload = JSON.parse(atob(tok.split('.')[1]))
    return payload.role || ''
  } catch {
    return ''
  }
}

function isTokenExpired(): boolean {
  const tok = getToken()
  if (!tok) return true
  try {
    const payload = JSON.parse(atob(tok.split('.')[1]))
    return payload.exp * 1000 < Date.now()
  } catch {
    return true
  }
}

function isAuthed(): boolean {
  return !!getToken() && getRole() === 'superadmin' && !isTokenExpired()
}

function logout() {
  localStorage.removeItem('access_token')
}

async function request(path: string, options: RequestInit = {}) {
  const optHeaders = (options.headers || {}) as Record<string, string>
  const headers: Record<string, string> = {}
  // Merge custom headers first, then defaults (caller wins for Content-Type)
  for (const k in optHeaders) headers[k] = optHeaders[k]
  if (!headers['Content-Type']) headers['Content-Type'] = 'application/json'
  const tok = getToken()
  if (tok) headers['Authorization'] = `Bearer ${tok}`

  const res = await fetch(`${API_BASE}${path}`, { ...options, headers })
  const data = await res.json().catch(() => ({}))
  if (!res.ok) {
    if (res.status === 401) {
      localStorage.removeItem('access_token')
      window.location.href = '/login'
    }
    throw new Error(data.message || `HTTP ${res.status}`)
  }
  return data
}

export const api = {
  // Dashboard
  getDashboard: () => request('/api/superadmin/dashboard'),

  // Voucher programs
  listVoucherPrograms: () => request('/api/superadmin/billing/voucher-programs'),
  createVoucherProgram: (body: any) =>
    request('/api/superadmin/billing/voucher-programs', { method: 'POST', body: JSON.stringify(body) }),
  updateVoucherProgram: (id: string, body: any) =>
    request(`/api/superadmin/billing/voucher-programs/${id}`, { method: 'PUT', body: JSON.stringify(body) }),
  deleteVoucherProgram: (id: string) =>
    request(`/api/superadmin/billing/voucher-programs/${id}`, { method: 'DELETE' }),
  bulkUploadVoucherPrograms: (file: File) => {
    const formData = new FormData()
    formData.append('file', file)
    const headers: Record<string, string> = {}
    const tok = getToken()
    if (tok) headers['Authorization'] = `Bearer ${tok}`
    return fetch(`${API_BASE}/api/superadmin/billing/voucher-programs/bulk-upload`, {
      method: 'POST',
      headers,
      body: formData,
    }).then(r => r.json())
  },
  getVoucherAnalytics: (programId?: string) =>
    request(`/api/superadmin/billing/voucher-analytics${programId ? `?program_id=${programId}` : ''}`),

  // Voucher link generation
  generateVoucherLinks: (body: any) =>
    request('/api/superadmin/billing/voucher-links/generate', { method: 'POST', body: JSON.stringify(body) }),
  listVoucherLinks: (params: { program_id?: string; redeemed?: boolean } = {}) => {
    const qs = new URLSearchParams()
    if (params.program_id) qs.set('program_id', params.program_id)
    if (params.redeemed) qs.set('redeemed', 'true')
    return request(`/api/superadmin/billing/voucher-links?${qs.toString()}`)
  },

  // Plans & features
  listPlans: () => request('/api/superadmin/billing/plans'),

  // Tenant management
  getTenants: () => request('/api/superadmin/tenants'),
  createTenant: (body: any) =>
    request('/api/superadmin/tenants', { method: 'POST', body: JSON.stringify(body) }),
  getTenantProfile: (tenantId: string) =>
    request(`/api/superadmin/tenants/profile?id=${encodeURIComponent(tenantId)}`),
  updateTenantProfile: (data: any) =>
    request('/api/superadmin/tenants/profile', { method: 'PUT', body: JSON.stringify(data) }),
  uploadTenantLogo: (tenantId: string, file: File) => {
    const formData = new FormData()
    formData.append('logo', file)
    const tok = getToken()
    return fetch(`${API_BASE}/api/superadmin/tenants/profile/logo?id=${encodeURIComponent(tenantId)}`, {
      method: 'POST',
      headers: { 'Authorization': `Bearer ${tok}` },
      body: formData,
    }).then(r => r.json())
  },
  deleteTenant: (tenantId: string) =>
    request(`/api/superadmin/tenants?id=${encodeURIComponent(tenantId)}`, { method: 'DELETE' }),
  impersonateTenant: (tenantId: string) =>
    request(`/api/superadmin/tenants/${encodeURIComponent(tenantId)}/impersonate`, { method: 'POST' }),
  listPlanFeatures: (planId?: string) =>
    request(`/api/superadmin/billing/plan-features${planId ? `?plan_id=${planId}` : ''}`),
  fetchPlanFeatureMatrix: (planId: string) =>
    request(`/api/superadmin/billing/plan-features-matrix/${planId}`),
  updatePlanFeatureNumeric: (planId: string, features: Record<string, number>) =>
    request(`/api/superadmin/billing/plan-features-matrix/${planId}`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(features),
    }),

  // Frozen accounts
  getFrozenAccounts: () => request('/api/superadmin/dashboard').then((d: any) => d.data?.recent_frozen || []),

  // Monitoring / HA status
  getHealthStatus: () => request('/api/superadmin/billing/health-status'),

  // F063: WA Center — platform-level WhatsApp for REG/OTP/VERIF
  getWAStatus: () => request('/api/superadmin/wa/status', {
    headers: { 'X-Tenant-ID': 'system' }
  }),
  getWAQR: () => request('/api/superadmin/wa/qr', {
    headers: { 'X-Tenant-ID': 'system' }
  }),
  // F064: Platform WA provider selector
  getPlatformProvider: () => request('/api/superadmin/wa/platform-provider'),
  setPlatformProvider: (wa_provider: string) => request('/api/superadmin/wa/platform-provider', {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ wa_provider }),
  }),
}

export { isAuthed, getRole, logout, getToken, request }
