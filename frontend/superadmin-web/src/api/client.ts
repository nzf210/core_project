// Lightweight API client for superadmin. All requests go through /api which
// is reverse-proxied to api-gateway (port 8000) by vite dev server.
// Api-gateway then injects X-User-Role and routes to billing-service.

const API_BASE = ''  // relative, vite proxy handles it

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
  getDashboard: () => request('/admin/dashboard'),

  // Voucher programs
  listVoucherPrograms: () => request('/admin/voucher-programs'),
  createVoucherProgram: (body: any) =>
    request('/admin/voucher-programs', { method: 'POST', body: JSON.stringify(body) }),
  updateVoucherProgram: (id: string, body: any) =>
    request(`/admin/voucher-programs/${id}`, { method: 'PUT', body: JSON.stringify(body) }),
  deleteVoucherProgram: (id: string) =>
    request(`/admin/voucher-programs/${id}`, { method: 'DELETE' }),
  bulkUploadVoucherPrograms: (file: File) => {
    const formData = new FormData()
    formData.append('file', file)
    const headers: Record<string, string> = {}
    const tok = getToken()
    if (tok) headers['Authorization'] = `Bearer ${tok}`
    return fetch(`${API_BASE}/admin/voucher-programs/bulk-upload`, {
      method: 'POST',
      headers,
      body: formData,
    }).then(r => r.json())
  },
  getVoucherAnalytics: (programId?: string) =>
    request(`/admin/voucher-analytics${programId ? `?program_id=${programId}` : ''}`),

  // Voucher link generation
  generateVoucherLinks: (body: any) =>
    request('/admin/voucher-links/generate', { method: 'POST', body: JSON.stringify(body) }),
  listVoucherLinks: (params: { program_id?: string; redeemed?: boolean } = {}) => {
    const qs = new URLSearchParams()
    if (params.program_id) qs.set('program_id', params.program_id)
    if (params.redeemed) qs.set('redeemed', 'true')
    return request(`/admin/voucher-links?${qs.toString()}`)
  },

  // Plans & features
  listPlans: () => request('/admin/plans'),
  listPlanFeatures: (planId?: string) =>
    request(`/admin/plan-features${planId ? `?plan_id=${planId}` : ''}`),
  fetchPlanFeatureMatrix: (planId: string) =>
    request(`/admin/plan-features-matrix/${planId}`),
  updatePlanFeatureNumeric: (planId: string, features: Record<string, number>) =>
    request(`/admin/plan-features-matrix/${planId}`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(features),
    }),

  // Frozen accounts
  getFrozenAccounts: () => request('/admin/dashboard').then((d: any) => d.data?.recent_frozen || []),

  // Monitoring / HA status
  getHealthStatus: () => request('/admin/health-status'),

  // F063: WA Center — platform-level WhatsApp for REG/OTP/VERIF
  getWAStatus: () => request('/admin/wa/status', {
    headers: { 'X-Tenant-ID': 'system' }
  }),
  getWAQR: () => request('/admin/wa/qr', {
    headers: { 'X-Tenant-ID': 'system' }
  }),
  // F064: Platform WA provider selector
  getPlatformProvider: () => request('/admin/wa/platform-provider'),
  setPlatformProvider: (wa_provider: string) => request('/admin/wa/platform-provider', {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ wa_provider }),
  }),
}

export { isAuthed, getRole, logout, getToken, request }
