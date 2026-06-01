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

function isAuthed(): boolean {
  return !!getToken() && getRole() === 'superadmin'
}

function logout() {
  localStorage.removeItem('access_token')
}

async function request(path: string, options: RequestInit = {}) {
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(options.headers as Record<string, string>),
  }
  const tok = getToken()
  if (tok) headers['Authorization'] = `Bearer ${tok}`

  const res = await fetch(`${API_BASE}${path}`, { ...options, headers })
  const data = await res.json().catch(() => ({}))
  if (!res.ok) {
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

  // Frozen accounts
  getFrozenAccounts: () => request('/admin/dashboard').then((d: any) => d.data?.recent_frozen || []),
}

export { isAuthed, getRole, logout, getToken }
